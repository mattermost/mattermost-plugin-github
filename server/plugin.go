package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

const (
	GITHUB_TOKEN_KEY           = "_githubtoken"
	GITHUB_STATE_KEY           = "_githubstate"
	API_ERROR_ID_NOT_CONNECTED = "not_connected"
)

type Plugin struct {
	plugin.MattermostPlugin
	githubClient *github.Client

	GitHubOrg               string
	Username                string
	GitHubOAuthClientID     string
	GitHubOAuthClientSecret string
	WebhookSecret           string
}

func githubConnect(token string) *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	return client
}

func (p *Plugin) IsValid() error {
	/*if p.GitHubOrg == "" {
		return fmt.Errorf("Must have a github org")
	}*/

	if p.GitHubOAuthClientID == "" {
		return fmt.Errorf("Must have a github oauth client id")
	}

	if p.GitHubOAuthClientSecret == "" {
		return fmt.Errorf("Must have a github oauth client secret")
	}

	/*if p.Username == "" {
		return fmt.Errorf("Need a username to make posts as.")
	}*/

	return nil
}

func (p *Plugin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := p.IsValid(); err != nil {
		http.Error(w, "This plugin is not configured.", http.StatusNotImplemented)
	}

	w.Header().Set("Content-Type", "application/json")

	switch path := r.URL.Path; path {
	case "/webhook":
		p.handleWebhook(w, r)
	case "/oauth/connect":
		p.connectUserToGitHub(w, r)
	case "/oauth/complete":
		p.completeConnectUserToGitHub(w, r)
	case "/api/v1/connected":
		p.getConnected(w, r)
	case "/api/v1/reviews":
		p.getReviews(w, r)
	case "/api/v1/mentions":
		p.getMentions(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (p *Plugin) getOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.GitHubOAuthClientID,
		ClientSecret: p.GitHubOAuthClientSecret,
		Scopes:       []string{"repo"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://github.com/login/oauth/authorize",
			TokenURL: "https://github.com/login/oauth/access_token",
		},
	}
}

/*type PullRequestWaitingReview struct {
	GitHubRepo        string `url:"github_repo"`
	GitHubUserName    string `url:"github_username"`
	PullRequestNumber int    `url:"pullrequest_number"`
	PullRequestURL    string `url:"pullrequest_url"`
}

type PullRequestWaitingReviews []PullRequestWaitingReview*/
func (p *Plugin) connectUserToGitHub(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	conf := p.getOAuthConfig()

	state := fmt.Sprintf("%v_%v", model.NewId(), userID)

	p.API.KVSet(state, []byte(state))

	url := conf.AuthCodeURL(state, oauth2.AccessTypeOffline)

	http.Redirect(w, r, url, http.StatusFound)
}

type GitHubUserInfo struct {
	Token          *oauth2.Token
	GitHubUsername string
}

func (p *Plugin) completeConnectUserToGitHub(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	conf := p.getOAuthConfig()

	code := r.URL.Query().Get("code")
	if len(code) == 0 {
		http.Error(w, "missing authorization code", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")

	if storedState, err := p.API.KVGet(state); err != nil {
		http.Error(w, "missing stored state", http.StatusBadRequest)
		return
	} else if string(storedState) != state {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}

	userID := strings.Split(state, "_")[1]

	p.API.KVDelete(state)

	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	githubClient := githubConnect(tok.AccessToken)
	user, _, err := githubClient.Users.Get(ctx, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userInfo := &GitHubUserInfo{
		Token:          tok,
		GitHubUsername: *user.Login,
	}

	jsonInfo, err := json.Marshal(userInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p.API.KVSet(userID+GITHUB_TOKEN_KEY, jsonInfo)

	// Post intro post
	channel, _ := p.API.GetDirectChannel(userID, userID)
	post := &model.Post{
		UserId:    userID,
		ChannelId: channel.Id,
		Message:   "##### Welcome to the Mattermost GitHub Plugin!\nCheck out the buttons in the bottom left corner of Mattermost.\n* The first button there tells you how many pull requests are awaiting your review\n* The second tracks the number of open issues/pull requests you have mentions in\n* The third will refresh the numbers\n\nClick on them!",
		Type:      "custom_git_welcome",
		Props: map[string]interface{}{
			"from_webhook":      "true",
			"override_username": "GitHub Plugin",
			"override_icon_url": "https://assets-cdn.github.com/images/modules/logos_page/GitHub-Mark.png",
		},
	}

	if _, err := p.API.CreatePost(post); err != nil {
		mlog.Error(err.Error())
	}

	http.Redirect(w, r, "http://localhost:8065", http.StatusFound)
}

type APIErrorResponse struct {
	ID         string `json:"id"`
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
}

func writeAPIError(w http.ResponseWriter, err *APIErrorResponse) {
	b, _ := json.Marshal(err)
	w.WriteHeader(err.StatusCode)
	w.Write(b)
}

func (p *Plugin) getGitHubUserInfo(userID string) (*GitHubUserInfo, *APIErrorResponse) {
	var userInfo GitHubUserInfo

	if infoBytes, err := p.API.KVGet(userID + GITHUB_TOKEN_KEY); err != nil || infoBytes == nil {
		return nil, &APIErrorResponse{ID: API_ERROR_ID_NOT_CONNECTED, Message: "Must connect user account to GitHub first.", StatusCode: http.StatusBadRequest}
	} else if err := json.Unmarshal(infoBytes, &userInfo); err != nil {
		return nil, &APIErrorResponse{ID: "", Message: "Unable to parse token.", StatusCode: http.StatusInternalServerError}
	}

	return &userInfo, nil
}

type ConnectedResponse struct {
	Connected      bool   `json:"connected"`
	GitHubUsername string `json:"github_username"`
	GitHubClientID string `json:"github_client_id"`
}

func (p *Plugin) getConnected(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		writeAPIError(w, &APIErrorResponse{ID: "", Message: "Not authorized.", StatusCode: http.StatusUnauthorized})
		return
	}

	resp := &ConnectedResponse{Connected: false}

	info, _ := p.getGitHubUserInfo(userID)
	if info != nil && info.Token != nil {
		resp.Connected = true
		resp.GitHubUsername = info.GitHubUsername
		resp.GitHubClientID = p.GitHubOAuthClientID
	}

	b, _ := json.Marshal(resp)
	w.Write(b)
}

func (p *Plugin) getMentions(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	ctx := context.Background()

	var githubClient *github.Client
	username := ""

	if info, err := p.getGitHubUserInfo(userID); err != nil {
		writeAPIError(w, err)
		return
	} else {
		githubClient = githubConnect(info.Token.AccessToken)
		username = info.GitHubUsername
	}

	result, _, err := githubClient.Search.Issues(ctx, fmt.Sprintf("is:open mentions:%v archived:false", username), &github.SearchOptions{})
	if err != nil {
		mlog.Error(err.Error())
	}

	resp, _ := json.Marshal(result.Issues)
	w.Write(resp)
}

func (p *Plugin) getReviews(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	ctx := context.Background()

	var githubClient *github.Client
	username := ""

	if info, err := p.getGitHubUserInfo(userID); err != nil {
		writeAPIError(w, err)
		return
	} else {
		githubClient = githubConnect(info.Token.AccessToken)
		username = info.GitHubUsername
	}

	result, _, err := githubClient.Search.Issues(ctx, fmt.Sprintf("is:pr is:open review-requested:%v archived:false", username), &github.SearchOptions{})
	if err != nil {
		mlog.Error(err.Error())
	}

	resp, _ := json.Marshal(result.Issues)
	w.Write(resp)

	/*
		message := fmt.Sprintf("You have %v pull requests awaiting your review\n", result.GetTotal())
		for _, issue := range result.Issues {
			message += fmt.Sprintf("* %v\n", issue.GetHTMLURL())
		}

		post := &model.Post{
			UserID:    userID,
			ChannelID: "agahtb7e7b8uiccjy7a9mahptr",
			Message:   message,
		}

		if _, err := p.API.CreatePost(post); err != nil {
			mlog.Error(err.Error())
		}*/
}

func verifyWebhookSignature(secret []byte, signature string, body []byte) bool {

	const signaturePrefix = "sha1="
	const signatureLength = 45

	if len(signature) != signatureLength || !strings.HasPrefix(signature, signaturePrefix) {
		fmt.Println(signature)
		fmt.Println(len(signature))
		fmt.Println("HIT0")
		return false
	}

	actual := make([]byte, 20)
	hex.Decode(actual, []byte(signature[5:]))

	return hmac.Equal(signBody(secret, body), actual)
}

func signBody(secret, body []byte) []byte {
	computed := hmac.New(sha1.New, secret)
	computed.Write(body)
	return []byte(computed.Sum(nil))
}

func (p *Plugin) handleWebhook(w http.ResponseWriter, r *http.Request) {
	signature := r.Header.Get("X-Hub-Signature")

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request body", http.StatusBadRequest)
		return
	}

	if !verifyWebhookSignature([]byte(p.WebhookSecret), signature, body) {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	event, err := github.ParseWebHook(github.WebHookType(r), body)
	if err != nil {
		fmt.Println("Err2: " + err.Error())
		return
	}

	switch event := event.(type) {
	case *github.PullRequestEvent:
		fmt.Println("Stufff")
		fmt.Println(*event)
		fmt.Println(*event.Repo)
	}
}

/*
func (p *Plugin) SendTodoPost(message, userID, channelID string) {
	props := map[string]interface{}{}

	post := &model.Post{
		UserID:    userID,
		ChannelID: channelID,
		Message:   message,
		Type:      model.POST_DEFAULT,
		Props:     props,
	}
	p.api.CreatePost(post)
}

func NewString(st string) *string {
	return &st
}

func githubUserListToUsernames(users []*github.User) *[]string {
	var output []string
	for _, user := range users {
		output = append(output, *user.Login)
	}
	return &output
}

func processLables(labels []*github.Label) *[]map[string]string {
	var output []map[string]string
	for _, label := range labels {
		entry := map[string]string{
			"text":  *label.Name,
			"color": *label.Color,
		}
		output = append(output, entry)
	}

	return &output
}

func (p *Plugin) postFromPullRequest(org, repository string, pullRequest *github.PullRequest) *model.Post {
	props := map[string]interface{}{}
	props["number"] = fmt.Sprint(*pullRequest.Number)
	props["summary"] = pullRequest.Body
	props["title"] = pullRequest.Title
	props["assignees"] = githubUserListToUsernames(pullRequest.Assignees)
	prReviewers, _, _ := p.githubClient.PullRequests.ListReviewers(context.Background(), org, repository, pullRequest.GetNumber(), nil)
	props["reviewers"] = githubUserListToUsernames(prReviewers.Users)
	//labels, _, _ := p.githubClient.Issues.ListLabelsByIssue(context.Background(), org, repository, pullRequest.GetNumber(), nil)
	//props["labels"] = processLables(labels)
	props["submitted_at"] = fmt.Sprint(pullRequest.CreatedAt.Unix())

	return &model.Post{
		UserID:  p.userID,
		Message: "Joram screwed up",
		Type:    "custom_github_pull_request",
		Props:   props,
	}
}

func (p *Plugin) pullRequestOpened(repo string, pullRequest *github.PullRequest) {
	subscriptions, err := NewSubscriptionsFromKVStore(p.api.KeyValueStore())
	if err != nil {
		fmt.Println("Error: " + err.Error())
	}
	fmt.Println("Subscriptions:")
	fmt.Println(*subscriptions)
	fmt.Println("Repo: " + repo)

	gob.Register([]map[string]string{})

	channels := subscriptions.GetChannelsForRepository(repo)
	values := strings.Split(repo, "/")
	post := p.postFromPullRequest(values[0], values[1], pullRequest)
	for _, channel := range channels {
		post.ChannelID = channel
		_, err := p.api.CreatePost(post)
		fmt.Println("Chan: " + channel)
		if err != nil {
			fmt.Println("Chanerr: " + err.Error())
		}
	}
}

type AddReviewersToPR struct {
	PullRequestID int      `json:"pull_request_id"`
	Org           string   `json:"org"`
	Repo          string   `json:"repo"`
	Reviewers     []string `json:"reviewers"`
}

func (p *Plugin) handleReviewers(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	var req AddReviewersToPR
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	b, err := p.api.KeyValueStore().Get(userID + GITHUB_TOKEN_KEY)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	gitHubUserToken := string(b)

	githubClient := githubConnect(gitHubUserToken)

	reviewers := github.ReviewersRequest{
		Reviewers: req.Reviewers,
	}

	pr, _, err2 := githubClient.PullRequests.RequestReviewers(ctx, req.Org, req.Repo, req.PullRequestID, reviewers)
	if err2 != nil {
		http.Error(w, err2.Error(), http.StatusBadRequest)
		return
	}

	w.Write([]byte(fmt.Sprintf("%v", pr.GetHTMLURL())))
}
*/
