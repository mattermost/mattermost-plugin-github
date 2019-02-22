package main

import (
	"context"
	"encoding/json"
	"fmt"
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
	GITHUB_ICON_URL            = "https://github.githubassets.com/images/modules/logos_page/GitHub-Mark.png"
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
	config := p.getConfiguration()

	if err := config.IsValid(); err != nil {
		http.Error(w, "This plugin is not configured.", http.StatusNotImplemented)
		return
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
	case "/api/v1/yourprs":
		p.getYourPrs(w, r)
	case "/api/v1/yourassignments":
		p.getYourAssignments(w, r)
	case "/api/v1/mentions":
		p.getMentions(w, r)
	case "/api/v1/unreads":
		p.getUnreads(w, r)
	case "/api/v1/settings":
		p.updateSettings(w, r)
	case "/api/v1/user":
		p.getGitHubUser(w, r)
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

	state := fmt.Sprintf("%v_%v", model.NewId()[0:15], userID)

	p.API.KVSet(state, []byte(state))

	url := conf.AuthCodeURL(state, oauth2.AccessTypeOffline)

	http.Redirect(w, r, url, http.StatusFound)
}

func (p *Plugin) completeConnectUserToGitHub(w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

	ctx := context.Background()
	conf := p.getOAuthConfig()

	code := r.URL.Query().Get("code")
	if len(code) == 0 {
		http.Error(w, "missing authorization code", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")

	if storedState, err := p.API.KVGet(state); err != nil {
		fmt.Println(err.Error())
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
		fmt.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	githubClient := p.githubConnect(*tok)
	gitUser, _, err := githubClient.Users.Get(ctx, "")
	if err != nil {
		fmt.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userInfo := &GitHubUserInfo{
		UserID:         userID,
		Token:          tok,
		GitHubUsername: gitUser.GetLogin(),
		LastToDoPostAt: model.GetMillis(),
		Settings: &UserSettings{
			SidebarButtons: SETTING_BUTTONS_TEAM,
			DailyReminder:  true,
			Notifications:  true,
		},
		AllowedPrivateRepos: config.EnablePrivateRepo,
	}

	if err := p.storeGitHubUserInfo(userInfo); err != nil {
		fmt.Println(err.Error())
		http.Error(w, "Unable to connect user to GitHub", http.StatusInternalServerError)
		return
	}

	if err := p.storeGitHubToUserIDMapping(gitUser.GetLogin(), userID); err != nil {
		fmt.Println(err.Error())
	}

	// Post intro post
	message := fmt.Sprintf("#### Welcome to the Mattermost GitHub Plugin!\n"+
		"You've connected your Mattermost account to [%s](%s) on GitHub. Read about the features of this plugin below:\n\n"+
		"##### Daily Reminders\n"+
		"The first time you log in each day, you will get a post right here letting you know what messages you need to read and what pull requests are awaiting your review.\n"+
		"Turn off reminders with `/github settings reminders off`.\n\n"+
		"##### Notifications\n"+
		"When someone mentions you, requests your review, comments on or modifies one of your pull requests/issues, or assigns you, you'll get a post here about it.\n"+
		"Turn off notifications with `/github settings notifications off`.\n\n"+
		"##### Sidebar Buttons\n"+
		"Check out the buttons in the left-hand sidebar of Mattermost.\n"+
		"* The first button tells you how many pull requests you have submitted.\n"+
		"* The second shows the number of PR that are awaiting your review.\n"+
		"* The third shows the number of PR and issues your are assiged to.\n"+
		"* The fourth tracks the number of unread messages you have.\n"+
		"* The fifth will refresh the numbers.\n\n"+
		"Click on them!\n\n"+
		"##### Slash Commands\n"+
		strings.Replace(COMMAND_HELP, "|", "`", -1), gitUser.GetLogin(), gitUser.GetHTMLURL())
	p.CreateBotDMPost(userID, message, "custom_git_welcome")

	p.API.PublishWebSocketEvent(
		WS_EVENT_CONNECT,
		map[string]interface{}{
			"connected":        true,
			"github_username":  userInfo.GitHubUsername,
			"github_client_id": config.GitHubOAuthClientID,
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
		<p>Completed connecting to GitHub. Please close this window.</p>
	</body>
</html>
`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

type ConnectedResponse struct {
	Connected         bool          `json:"connected"`
	GitHubUsername    string        `json:"github_username"`
	GitHubClientID    string        `json:"github_client_id"`
	EnterpriseBaseURL string        `json:"enterprise_base_url,omitempty"`
	Organization      string        `json:"organization"`
	Settings          *UserSettings `json:"settings"`
}

type GitHubUserRequest struct {
	UserID string `json:"user_id"`
}

type GitHubUserResponse struct {
	Username string `json:"username"`
}

func (p *Plugin) getGitHubUser(w http.ResponseWriter, r *http.Request) {
	requestorID := r.Header.Get("Mattermost-User-ID")
	if requestorID == "" {
		writeAPIError(w, &APIErrorResponse{ID: "", Message: "Not authorized.", StatusCode: http.StatusUnauthorized})
		return
	}

	req := &GitHubUserRequest{}
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil || req.UserID == "" {
		if err != nil {
			mlog.Error("Error decoding JSON body: " + err.Error())
		}
		writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a JSON object with a non-blank user_id field.", StatusCode: http.StatusBadRequest})
		return
	}

	userInfo, apiErr := p.getGitHubUserInfo(req.UserID)
	if apiErr != nil {
		if apiErr.ID == API_ERROR_ID_NOT_CONNECTED {
			writeAPIError(w, &APIErrorResponse{ID: "", Message: "User is not connected to a GitHub account.", StatusCode: http.StatusNotFound})
		} else {
			writeAPIError(w, apiErr)
		}
		return
	}

	if userInfo == nil {
		writeAPIError(w, &APIErrorResponse{ID: "", Message: "User is not connected to a GitHub account.", StatusCode: http.StatusNotFound})
		return
	}

	resp := &GitHubUserResponse{Username: userInfo.GitHubUsername}
	b, jsonErr := json.Marshal(resp)
	if jsonErr != nil {
		mlog.Error("Error encoding JSON response: " + jsonErr.Error())
		writeAPIError(w, &APIErrorResponse{ID: "", Message: "Encountered an unexpected error. Please try again.", StatusCode: http.StatusInternalServerError})
	}
	w.Write(b)
}

func (p *Plugin) getConnected(w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		writeAPIError(w, &APIErrorResponse{ID: "", Message: "Not authorized.", StatusCode: http.StatusUnauthorized})
		return
	}

	resp := &ConnectedResponse{
		Connected:         false,
		EnterpriseBaseURL: config.EnterpriseBaseURL,
		Organization:      config.GitHubOrg,
	}

	info, _ := p.getGitHubUserInfo(userID)
	if info != nil && info.Token != nil {
		resp.Connected = true
		resp.GitHubUsername = info.GitHubUsername
		resp.GitHubClientID = config.GitHubOAuthClientID
		resp.Settings = info.Settings

		if info.Settings.DailyReminder && r.URL.Query().Get("reminder") == "true" {
			lastPostAt := info.LastToDoPostAt

			var timezone *time.Location
			offset, _ := strconv.Atoi(r.Header.Get("X-Timezone-Offset"))
			timezone = time.FixedZone("local", -60*offset)

			// Post to do message if it's the next day and been more than an hour since the last post
			now := model.GetMillis()
			nt := time.Unix(now/1000, 0).In(timezone)
			lt := time.Unix(lastPostAt/1000, 0).In(timezone)
			if nt.Sub(lt).Hours() >= 1 && (nt.Day() != lt.Day() || nt.Month() != lt.Month() || nt.Year() != lt.Year()) {
				p.PostToDo(info)
				info.LastToDoPostAt = now
				p.storeGitHubUserInfo(info)
			}
		}

		privateRepoStoreKey := info.UserID + GITHUB_PRIVATE_REPO_KEY
		if config.EnablePrivateRepo && !info.AllowedPrivateRepos {
			hasBeenNotified := false
			if val, err := p.API.KVGet(privateRepoStoreKey); err == nil {
				hasBeenNotified = val != nil
			} else {
				mlog.Error("Unable to get private repo key value, err=" + err.Error())
			}

			if !hasBeenNotified {
				p.CreateBotDMPost(info.UserID, "Private repositories have been enabled for this plugin. To be able to use them you must disconnect and reconnect your GitHub account. To reconnect your account, use the following slash commands: `/github disconnect` followed by `/github connect`.", "")
				if err := p.API.KVSet(privateRepoStoreKey, []byte("1")); err != nil {
					mlog.Error("Unable to set private repo key value, err=" + err.Error())
				}
			}
		}
	}

	b, _ := json.Marshal(resp)
	w.Write(b)
}

func (p *Plugin) getMentions(w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

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
		githubClient = p.githubConnect(*info.Token)
		username = info.GitHubUsername
	}

	result, _, err := githubClient.Search.Issues(ctx, getMentionSearchQuery(username, config.GitHubOrg), &github.SearchOptions{})
	if err != nil {
		mlog.Error(err.Error())
	}

	resp, _ := json.Marshal(result.Issues)
	w.Write(resp)
}

func (p *Plugin) getUnreads(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	ctx := context.Background()

	var githubClient *github.Client

	if info, err := p.getGitHubUserInfo(userID); err != nil {
		writeAPIError(w, err)
		return
	} else {
		githubClient = p.githubConnect(*info.Token)
	}

	notifications, _, err := githubClient.Activity.ListNotifications(ctx, &github.NotificationListOptions{})
	if err != nil {
		mlog.Error(err.Error())
	}

	filteredNotifications := []*github.Notification{}
	for _, n := range notifications {
		if n.GetReason() == "subscribed" {
			continue
		}

		if p.checkOrg(n.GetRepository().GetOwner().GetLogin()) != nil {
			continue
		}

		filteredNotifications = append(filteredNotifications, n)
	}

	resp, _ := json.Marshal(filteredNotifications)
	w.Write(resp)
}

func (p *Plugin) getReviews(w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

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
		githubClient = p.githubConnect(*info.Token)
		username = info.GitHubUsername
	}

	result, _, err := githubClient.Search.Issues(ctx, getReviewSearchQuery(username, config.GitHubOrg), &github.SearchOptions{})
	if err != nil {
		mlog.Error(err.Error())
	}

	resp, _ := json.Marshal(result.Issues)
	w.Write(resp)
}

func (p *Plugin) getYourPrs(w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

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
		githubClient = p.githubConnect(*info.Token)
		username = info.GitHubUsername
	}

	result, _, err := githubClient.Search.Issues(ctx, getYourPrsSearchQuery(username, config.GitHubOrg), &github.SearchOptions{})
	if err != nil {
		mlog.Error(err.Error())
	}

	resp, _ := json.Marshal(result.Issues)
	w.Write(resp)
}

func (p *Plugin) getYourAssignments(w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

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
		githubClient = p.githubConnect(*info.Token)
		username = info.GitHubUsername
	}

	result, _, err := githubClient.Search.Issues(ctx, getYourAssigneeSearchQuery(username, config.GitHubOrg), &github.SearchOptions{})
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
		githubClient = p.githubConnect(*info.Token)
		username = info.GitHubUsername
	}

	text, err := p.GetToDo(context.Background(), username, githubClient)
	if err != nil {
		mlog.Error(err.Error())
		writeAPIError(w, &APIErrorResponse{ID: "", Message: "Encountered an error getting the to do items.", StatusCode: http.StatusUnauthorized})
		return
	}

	if err := p.CreateBotDMPost(userID, text, "custom_git_todo"); err != nil {
		writeAPIError(w, &APIErrorResponse{ID: "", Message: "Encountered an error posting the to do items.", StatusCode: http.StatusUnauthorized})
	}

	w.Write([]byte("{\"status\": \"OK\"}"))
}

func (p *Plugin) updateSettings(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	var settings *UserSettings
	json.NewDecoder(r.Body).Decode(&settings)
	if settings == nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	info, err := p.getGitHubUserInfo(userID)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	info.Settings = settings

	if err := p.storeGitHubUserInfo(info); err != nil {
		mlog.Error(err.Error())
		http.Error(w, "Encountered error updating settings", http.StatusInternalServerError)
	}

	resp, _ := json.Marshal(info.Settings)
	w.Write(resp)
}
