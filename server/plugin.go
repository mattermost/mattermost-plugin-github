package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-server/mlog"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

const (
	GITHUB_TOKEN_KEY    = "_githubtoken"
	GITHUB_STATE_KEY    = "_githubstate"
	WS_EVENT_CONNECT    = "connect"
	WS_EVENT_DISCONNECT = "disconnect"
)

type Plugin struct {
	plugin.MattermostPlugin
	githubClient *github.Client

	GitHubOrg               string
	Username                string
	GitHubOAuthClientID     string
	GitHubOAuthClientSecret string
	WebhookSecret           string
	EncryptionKey           string
}

func githubConnect(token oauth2.Token) *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&token)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	return client
}

func (p *Plugin) OnActivate() error {
	p.API.RegisterCommand(getCommand())
	return nil
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

	if p.EncryptionKey == "" {
		return fmt.Errorf("Must have an encryption key")
	}

	/*if p.Username == "" {
		return fmt.Errorf("Need a username to make posts as.")
	}*/

	return nil
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

type GitHubUserInfo struct {
	Token          *oauth2.Token
	GitHubUsername string
}

func (p *Plugin) getGitHubUserInfo(userID string) (*GitHubUserInfo, *APIErrorResponse) {
	var userInfo GitHubUserInfo

	if infoBytes, err := p.API.KVGet(userID + GITHUB_TOKEN_KEY); err != nil || infoBytes == nil {
		return nil, &APIErrorResponse{ID: API_ERROR_ID_NOT_CONNECTED, Message: "Must connect user account to GitHub first.", StatusCode: http.StatusBadRequest}
	} else if err := json.Unmarshal(infoBytes, &userInfo); err != nil {
		return nil, &APIErrorResponse{ID: "", Message: "Unable to parse token.", StatusCode: http.StatusInternalServerError}
	}

	unencryptedToken, err := decrypt([]byte(p.EncryptionKey), userInfo.Token.AccessToken)
	if err != nil {
		mlog.Error(err.Error())
		return nil, &APIErrorResponse{ID: "", Message: "Unable to decrypt access token.", StatusCode: http.StatusInternalServerError}
	}

	fmt.Println(unencryptedToken)
	userInfo.Token.AccessToken = unencryptedToken

	return &userInfo, nil
}

func (p *Plugin) disconnectGitHubAccount(userID string) {
	p.API.KVDelete(userID + GITHUB_TOKEN_KEY)
	p.API.PublishWebSocketEvent(
		WS_EVENT_DISCONNECT,
		nil,
		&model.WebsocketBroadcast{UserId: userID},
	)
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
