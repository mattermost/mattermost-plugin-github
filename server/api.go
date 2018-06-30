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

	"github.com/google/go-github/github"
	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"

	"golang.org/x/oauth2"
)

const (
	API_ERROR_ID_NOT_CONNECTED = "not_connected"
	GITHUB_ICON_URL            = "https://assets-cdn.github.com/images/modules/logos_page/GitHub-Mark.png"
	GITHUB_USERNAME            = "GitHub Plugin"
)

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

	fmt.Println(tok.AccessToken)

	encryptedToken, err := encrypt([]byte(p.EncryptionKey), tok.AccessToken)
	if err != nil {
		mlog.Error(err.Error())
		http.Error(w, "Error encrypting access token", http.StatusInternalServerError)
		return
	}

	githubClient := githubConnect(*tok)
	user, _, err := githubClient.Users.Get(ctx, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Println("HIT0")

	tok.AccessToken = encryptedToken

	fmt.Println("HIT1")

	userInfo := &GitHubUserInfo{
		Token:          tok,
		GitHubUsername: *user.Login,
	}

	fmt.Println("HIT2")
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
			"override_username": GITHUB_USERNAME,
			"override_icon_url": GITHUB_ICON_URL,
		},
	}

	if _, err := p.API.CreatePost(post); err != nil {
		mlog.Error(err.Error())
	}

	p.API.PublishWebSocketEvent(
		WS_EVENT_CONNECT,
		map[string]interface{}{
			"connected":        true,
			"github_username":  userInfo.GitHubUsername,
			"github_client_id": p.GitHubOAuthClientID,
		},
		&model.WebsocketBroadcast{UserId: userID},
	)

	http.Redirect(w, r, "http://localhost:8065", http.StatusFound)
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
		githubClient = githubConnect(*info.Token)
		username = info.GitHubUsername
	}

	result, _, err := githubClient.Search.Issues(ctx, getMentionSearchQuery(username, p.GitHubOrg), &github.SearchOptions{})
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
		githubClient = githubConnect(*info.Token)
		username = info.GitHubUsername
	}

	result, _, err := githubClient.Search.Issues(ctx, getReviewSearchQuery(username, p.GitHubOrg), &github.SearchOptions{})
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
