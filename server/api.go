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
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"

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

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
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
	case "/api/v1/todo":
		p.postToDo(w, r)
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

	githubClient := githubConnect(*tok)
	gitUser, _, err := githubClient.Users.Get(ctx, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if user, err := p.API.GetUser(userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		if user.Props == nil {
			user.Props = model.StringMap{}
		}
		user.Props["git_user"] = *gitUser.Login
		_, err = p.API.UpdateUser(user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	userInfo := &GitHubUserInfo{
		UserID:         userID,
		Token:          tok,
		GitHubUsername: *gitUser.Login,
		LastToDoPostAt: model.GetMillis(),
	}

	if err := p.storeGitHubUserInfo(userInfo); err != nil {
		mlog.Error(err.Error())
		http.Error(w, "Unable to connect user to GitHub", http.StatusInternalServerError)
		return
	}

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

	html := `
<!DOCTYPE html>
<html>
	<head>
		<script>
			window.close();
		</script>
	</head>
	<body>
		<p>Completed connecting to GitHub.</p>
	</body>
</html>
`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
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
		lastPostAt := info.LastToDoPostAt

		var timezone *time.Location
		offset, err := strconv.Atoi(r.Header.Get("X-Timezone-Offset"))
		if err == nil {
			timezone = time.FixedZone("local", -60*offset)
		}

		// Post to do message if it's the next day
		now := model.GetMillis()
		nt := time.Unix(now/1000, 0).In(timezone)
		lt := time.Unix(lastPostAt/1000, 0).In(timezone)
		if nt.Sub(lt).Hours() >= 1 && (nt.Day() != lt.Day() || nt.Month() != lt.Month() || nt.Year() != lt.Year()) {
			p.PostToDo(info)
			info.LastToDoPostAt = now
			p.storeGitHubUserInfo(info)
		}
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
}

func (p *Plugin) postToDo(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		writeAPIError(w, &APIErrorResponse{ID: "", Message: "Not authorized.", StatusCode: http.StatusUnauthorized})
		return
	}

	var githubClient *github.Client
	username := ""

	if info, err := p.getGitHubUserInfo(userID); err != nil {
		writeAPIError(w, err)
		return
	} else {
		githubClient = githubConnect(*info.Token)
		username = info.GitHubUsername
	}

	text, err := p.GetToDo(context.Background(), username, githubClient)
	if err != nil {
		mlog.Error(err.Error())
		writeAPIError(w, &APIErrorResponse{ID: "", Message: "Encountered an error getting the to do items.", StatusCode: http.StatusUnauthorized})
		return
	}

	channel, _ := p.API.GetDirectChannel(userID, userID)
	post := &model.Post{
		UserId:    userID,
		ChannelId: channel.Id,
		Message:   text,
		Type:      "custom_git_todo",
		Props: map[string]interface{}{
			"from_webhook":      "true",
			"override_username": GITHUB_USERNAME,
			"override_icon_url": GITHUB_ICON_URL,
		},
	}

	if _, err := p.API.CreatePost(post); err != nil {
		mlog.Error(err.Error())
		writeAPIError(w, &APIErrorResponse{ID: "", Message: "Encountered an error posting the to do items.", StatusCode: http.StatusUnauthorized})
		return
	}

	w.Write([]byte("{\"status\": \"OK\"}"))
}

func verifyWebhookSignature(secret []byte, signature string, body []byte) bool {

	const signaturePrefix = "sha1="
	const signatureLength = 45

	if len(signature) != signatureLength || !strings.HasPrefix(signature, signaturePrefix) {
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
