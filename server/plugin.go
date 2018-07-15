package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost-server/mlog"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

const (
	GITHUB_TOKEN_KEY        = "_githubtoken"
	GITHUB_STATE_KEY        = "_githubstate"
	GITHUB_USERNAME_KEY     = "_githubusername"
	WS_EVENT_CONNECT        = "connect"
	WS_EVENT_DISCONNECT     = "disconnect"
	SETTING_BUTTONS_TEAM    = "team"
	SETTING_BUTTONS_CHANNEL = "channel"
	SETTING_BUTTONS_OFF     = "off"
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
	EnterpriseBaseURL       string
	EnterpriseUploadURL     string
}

func (p *Plugin) githubConnect(token oauth2.Token) *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&token)
	tc := oauth2.NewClient(ctx, ts)

	if len(p.EnterpriseBaseURL) == 0 || len(p.EnterpriseUploadURL) == 0 {
		return github.NewClient(tc)
	}

	client, err := github.NewEnterpriseClient(p.EnterpriseBaseURL, p.EnterpriseUploadURL, tc)
	if err != nil {
		mlog.Error(err.Error())
		return github.NewClient(tc)
	}
	return client
}

func (p *Plugin) OnActivate() error {
	p.API.RegisterCommand(getCommand())
	return nil
}

func (p *Plugin) IsValid() error {
	if p.GitHubOAuthClientID == "" {
		return fmt.Errorf("Must have a github oauth client id")
	}

	if p.GitHubOAuthClientSecret == "" {
		return fmt.Errorf("Must have a github oauth client secret")
	}

	if p.EncryptionKey == "" {
		return fmt.Errorf("Must have an encryption key")
	}

	if p.Username == "" {
		return fmt.Errorf("Need a username to make posts as")
	}

	return nil
}

func (p *Plugin) getOAuthConfig() *oauth2.Config {
	baseURL := "https://github.com/"
	if len(p.EnterpriseBaseURL) > 0 {
		baseURL = p.EnterpriseBaseURL
	}

	return &oauth2.Config{
		ClientID:     p.GitHubOAuthClientID,
		ClientSecret: p.GitHubOAuthClientSecret,
		Scopes:       []string{"public_repo,notifications"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  baseURL + "login/oauth/authorize",
			TokenURL: baseURL + "login/oauth/access_token",
		},
	}
}

type GitHubUserInfo struct {
	UserID         string
	Token          *oauth2.Token
	GitHubUsername string
	LastToDoPostAt int64
	Settings       *UserSettings
}

type UserSettings struct {
	SidebarButtons string `json:"sidebar_buttons"`
	DailyReminder  bool   `json:"daily_reminder"`
}

func (p *Plugin) storeGitHubUserInfo(info *GitHubUserInfo) error {
	encryptedToken, err := encrypt([]byte(p.EncryptionKey), info.Token.AccessToken)
	if err != nil {
		return err
	}

	info.Token.AccessToken = encryptedToken

	jsonInfo, err := json.Marshal(info)
	if err != nil {
		return err
	}

	if err := p.API.KVSet(info.UserID+GITHUB_TOKEN_KEY, jsonInfo); err != nil {
		return err
	}

	return nil
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

	userInfo.Token.AccessToken = unencryptedToken

	return &userInfo, nil
}

func (p *Plugin) storeGitHubToUserIDMapping(githubUsername, userID string) error {
	if err := p.API.KVSet(githubUsername+GITHUB_USERNAME_KEY, []byte(userID)); err != nil {
		return fmt.Errorf("Encountered error saving github username mapping")
	}
	return nil
}

func (p *Plugin) getGitHubToUserIDMapping(githubUsername string) string {
	userID, _ := p.API.KVGet(githubUsername + GITHUB_USERNAME_KEY)
	return string(userID)
}

func (p *Plugin) disconnectGitHubAccount(userID string) {
	userInfo, _ := p.getGitHubUserInfo(userID)
	if userInfo == nil {
		return
	}

	p.API.KVDelete(userID + GITHUB_TOKEN_KEY)
	p.API.KVDelete(userInfo.GitHubUsername + GITHUB_USERNAME_KEY)

	if user, err := p.API.GetUser(userID); err == nil && user.Props != nil && len(user.Props["git_user"]) > 0 {
		delete(user.Props, "git_user")
		p.API.UpdateUser(user)
	}

	p.API.PublishWebSocketEvent(
		WS_EVENT_DISCONNECT,
		nil,
		&model.WebsocketBroadcast{UserId: userID},
	)
}

func (p *Plugin) PostToDo(info *GitHubUserInfo) {
	text, err := p.GetToDo(context.Background(), info.GitHubUsername, p.githubConnect(*info.Token))
	if err != nil {
		mlog.Error(err.Error())
		return
	}

	channel, _ := p.API.GetDirectChannel(info.UserID, info.UserID)
	if channel == nil {
		mlog.Error("Couldn't get user's self DM channel", mlog.String("user_id", info.UserID))
		return
	}

	post := &model.Post{
		UserId:    info.UserID,
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
	}
}

func (p *Plugin) GetToDo(ctx context.Context, username string, githubClient *github.Client) (string, error) {
	issueResults, _, err := githubClient.Search.Issues(ctx, getReviewSearchQuery(username, p.GitHubOrg), &github.SearchOptions{})
	if err != nil {
		return "", err
	}

	text := "##### Unread Messages\n"

	notifications, _, err := githubClient.Activity.ListNotifications(ctx, &github.NotificationListOptions{})
	if err != nil {
		return "", err
	}

	notificationCount := 0
	notificationContent := ""
	for _, n := range notifications {
		if n.GetReason() == "subscribed" {
			continue
		}

		if p.checkOrg(n.GetRepository().GetOwner().GetLogin()) != nil {
			continue
		}

		url := n.GetSubject().GetURL()
		url = strings.Replace(url, "api.", "", 1)
		url = strings.Replace(url, "repos/", "", 1)
		url = strings.Replace(url, "/pulls/", "/pull/", 1)

		notificationContent += fmt.Sprintf("* %v\n", url)
		notificationCount++
	}

	if notificationCount == 0 {
		text += "You don't have any unread messages.\n"
	} else {
		text += fmt.Sprintf("You have %v unread messages:\n", notificationCount)
		text += notificationContent
	}

	text += "##### Review Requests\n"

	if issueResults.GetTotal() == 0 {
		text += "You have don't have any pull requests awaiting your review."
	} else {
		text += fmt.Sprintf("You have %v pull requests awaiting your review:\n", issueResults.GetTotal())

		for _, pr := range issueResults.Issues {
			text += fmt.Sprintf("* %v\n", pr.GetHTMLURL())
		}
	}

	return text, nil
}

func (p *Plugin) checkOrg(org string) error {
	configOrg := strings.TrimSpace(p.GitHubOrg)
	if configOrg != "" && configOrg != org {
		return fmt.Errorf("Only repositories in the %v organization are supported", configOrg)
	}

	return nil
}
