package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/google/go-github/github"
	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"golang.org/x/oauth2"
)

const (
	GITHUB_TOKEN_KEY        = "_githubtoken"
	GITHUB_STATE_KEY        = "_githubstate"
	GITHUB_USERNAME_KEY     = "_githubusername"
	WS_EVENT_CONNECT        = "connect"
	WS_EVENT_DISCONNECT     = "disconnect"
	WS_EVENT_REFRESH        = "refresh"
	SETTING_BUTTONS_TEAM    = "team"
	SETTING_BUTTONS_CHANNEL = "channel"
	SETTING_BUTTONS_OFF     = "off"
	SETTING_NOTIFICATIONS   = "notifications"
	SETTING_REMINDERS       = "reminders"
	SETTING_ON              = "on"
	SETTING_OFF             = "off"
)

type Plugin struct {
	plugin.MattermostPlugin
	githubClient *github.Client

	BotUserID string

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

	baseURL, _ := url.Parse(p.EnterpriseBaseURL)
	baseURL.Path = path.Join(baseURL.Path, "api", "v3")

	uploadURL, _ := url.Parse(p.EnterpriseUploadURL)
	uploadURL.Path = path.Join(uploadURL.Path, "api", "v3")

	client, err := github.NewEnterpriseClient(baseURL.String(), uploadURL.String(), tc)
	if err != nil {
		mlog.Error(err.Error())
		return github.NewClient(tc)
	}
	return client
}

func (p *Plugin) OnActivate() error {
	if err := p.IsValid(); err != nil {
		return err
	}
	p.API.RegisterCommand(getCommand())
	user, err := p.API.GetUserByUsername(p.Username)
	if err != nil {
		mlog.Error(err.Error())
		return fmt.Errorf("Unable to find user with configured username: %v", p.Username)
	}

	p.BotUserID = user.Id
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
		return fmt.Errorf("Need a user to make posts as")
	}

	return nil
}

func (p *Plugin) getOAuthConfig() *oauth2.Config {
	authURL, _ := url.Parse("https://github.com/")
	tokenURL, _ := url.Parse("https://github.com/")
	if len(p.EnterpriseBaseURL) > 0 {
		authURL, _ = url.Parse(p.EnterpriseBaseURL)
		tokenURL, _ = url.Parse(p.EnterpriseBaseURL)
	}

	authURL.Path = path.Join(authURL.Path, "login", "oauth", "authorize")
	tokenURL.Path = path.Join(tokenURL.Path, "login", "oauth", "access_token")

	return &oauth2.Config{
		ClientID:     p.GitHubOAuthClientID,
		ClientSecret: p.GitHubOAuthClientSecret,
		Scopes:       []string{"public_repo,notifications"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL.String(),
			TokenURL: tokenURL.String(),
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
	Notifications  bool   `json:"notifications"`
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

func (p *Plugin) CreateBotPostWithProps(channelID, userID, message, postType string, props map[string]interface{}) *model.AppError {
	props["from_webhook"] = "true"
	props["override_username"] = GITHUB_USERNAME
	props["override_icon_url"] = GITHUB_ICON_URL
	fmt.Println(props)

	post := &model.Post{
		UserId:    p.BotUserID,
		ChannelId: channelID,
		Message:   message,
		Type:      postType,
		Props:     props,
	}

	if _, err := p.API.CreatePost(post); err != nil {
		mlog.Error(err.Error())
		return err
	}

	return nil
}

func (p *Plugin) CreateBotDMPostWithProps(userID, message, postType string, props map[string]interface{}) *model.AppError {
	channel, err := p.API.GetDirectChannel(userID, p.BotUserID)
	if err != nil {
		mlog.Error("Couldn't get bot's DM channel", mlog.String("user_id", userID))
		return err
	}

	return p.CreateBotPostWithProps(channel.Id, userID, message, postType, props)
}

func (p *Plugin) CreateBotDMPost(userID, message, postType string) *model.AppError {
	return p.CreateBotDMPostWithProps(userID, message, postType, map[string]interface{}{})
}

func (p *Plugin) PostToDo(info *GitHubUserInfo) {
	todo, err := p.GetToDo(context.Background(), info.GitHubUsername, p.githubConnect(*info.Token))
	if err != nil {
		mlog.Error(err.Error())
		return
	}

	p.CreateBotDMPostWithProps(info.UserID, todo.Text, "custom_git_todo", todo.Props)
}

type ToDoMessage struct {
	Text       string
	HasUnreads bool
	Props      map[string]interface{}
}

func (p *Plugin) GetToDo(ctx context.Context, username string, githubClient *github.Client) (ToDoMessage, error) {
	todo := ToDoMessage{}

	issueResults, _, err := githubClient.Search.Issues(ctx, getReviewSearchQuery(username, p.GitHubOrg), &github.SearchOptions{})
	if err != nil {
		return todo, err
	}

	notifications, _, err := githubClient.Activity.ListNotifications(ctx, &github.NotificationListOptions{})
	if err != nil {
		return todo, err
	}

	yourPrs, _, err := githubClient.Search.Issues(ctx, getYourPrsSearchQuery(username, p.GitHubOrg), &github.SearchOptions{})
	if err != nil {
		return todo, err
	}

	yourAssignments, _, err := githubClient.Search.Issues(ctx, getYourAssigneeSearchQuery(username, p.GitHubOrg), &github.SearchOptions{})
	if err != nil {
		return todo, err
	}

	notificationProps := []map[string]interface{}{}

	text := "##### Unread Messages\n"

	notificationCount := 0
	notificationContent := ""
	for _, n := range notifications {
		if n.GetReason() == "subscribed" {
			continue
		}

		if n.GetRepository() == nil {
			p.API.LogError("Unable to get repository for notification in todo list. Skipping.")
			continue
		}

		if p.checkOrg(n.GetRepository().GetOwner().GetLogin()) != nil {
			continue
		}

		url := fixGithubNotificationSubjectURL(n.GetSubject().GetURL())

		switch n.GetSubject().GetType() {
		case "RepositoryVulnerabilityAlert":
			message := fmt.Sprintf("[Vulnerability Alert for %v](%v)", n.GetRepository().GetFullName(), fixGithubNotificationSubjectURL(n.GetSubject().GetURL()))
			notificationContent += fmt.Sprintf("* %v\n", message)
		default:
			notificationContent += fmt.Sprintf("* %v\n", url)
		}

		notificationProps = append(notificationProps, map[string]interface{}{"url": url, "title": n.GetSubject().GetTitle(), "type": n.GetSubject().GetType(), "thread_id": n.GetID(), "repo": n.GetRepository().GetFullName(), "number": getIssueNumberFromURL(url)})

		notificationCount++
	}

	if notificationCount == 0 {
		text += "You don't have any unread messages.\n"
	} else {
		text += fmt.Sprintf("You have %v unread messages:\n", notificationCount)
		text += notificationContent
	}

	reviewProps := []map[string]interface{}{}
	text += "##### Review Requests\n"

	if issueResults.GetTotal() == 0 {
		text += "You have don't have any pull requests awaiting your review.\n"
	} else {
		text += fmt.Sprintf("You have %v pull requests awaiting your review:\n", issueResults.GetTotal())

		for _, pr := range issueResults.Issues {
			url := pr.GetHTMLURL()
			text += fmt.Sprintf("* %v\n", url)
			reviewProps = append(reviewProps, map[string]interface{}{"url": url, "title": pr.GetTitle(), "type": "PullRequest"})
		}
	}

	openProps := []map[string]interface{}{}
	text += "##### Your Open Pull Requests\n"

	if yourPrs.GetTotal() == 0 {
		text += "You have don't have any open pull requests.\n"
	} else {
		text += fmt.Sprintf("You have %v open pull requests:\n", yourPrs.GetTotal())

		for _, pr := range yourPrs.Issues {
			url := pr.GetHTMLURL()
			text += fmt.Sprintf("* %v\n", pr.GetHTMLURL())
			openProps = append(openProps, map[string]interface{}{"url": url, "title": pr.GetTitle(), "type": "PullRequest"})
		}
	}

	assignProps := []map[string]interface{}{}
	text += "##### Your Assignments\n"

	if yourAssignments.GetTotal() == 0 {
		text += "You have don't have any assignments.\n"
	} else {
		text += fmt.Sprintf("You have %v assignments:\n", yourAssignments.GetTotal())

		for _, assign := range yourAssignments.Issues {
			url := assign.GetHTMLURL()
			text += fmt.Sprintf("* %v\n", assign.GetHTMLURL())
			assignProps = append(assignProps, map[string]interface{}{"url": url, "title": assign.GetTitle(), "type": getIssueTypeFromURL(url)})
		}
	}

	todo.Text = text
	todo.Props = map[string]interface{}{}
	b, _ := json.Marshal(notificationProps)
	todo.Props["notifications"] = string(b)
	b, _ = json.Marshal(reviewProps)
	todo.Props["reviews"] = string(b)
	b, _ = json.Marshal(openProps)
	todo.Props["open"] = string(b)
	b, _ = json.Marshal(assignProps)
	todo.Props["assigns"] = string(b)

	return todo, nil
}

func (p *Plugin) checkOrg(org string) error {
	configOrg := strings.TrimSpace(p.GitHubOrg)
	if configOrg != "" && configOrg != org {
		return fmt.Errorf("Only repositories in the %v organization are supported", configOrg)
	}

	return nil
}

func (p *Plugin) sendRefreshEvent(userID string) {
	p.API.PublishWebSocketEvent(
		WS_EVENT_REFRESH,
		nil,
		&model.WebsocketBroadcast{UserId: userID},
	)
}
