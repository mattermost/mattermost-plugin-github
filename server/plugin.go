package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/gorilla/mux"

	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/pkg/errors"

	"github.com/google/go-github/v25/github"
	"golang.org/x/oauth2"
)

const (
	githubTokenKey       = "_githubtoken"
	githubUsernameKey    = "_githubusername"
	githubPrivateRepoKey = "_githubprivate"

	wsEventConnect    = "connect"
	wsEventDisconnect = "disconnect"
	wsEventRefresh    = "refresh"

	settingButtonsTeam   = "team"
	settingNotifications = "notifications"
	settingReminders     = "reminders"
	settingOn            = "on"
	settingOff           = "off"

	notificationReasonSubscribed = "subscribed"
)

type Plugin struct {
	plugin.MattermostPlugin
	// githubPermalinkRegex is used to parse github permalinks in post messages.
	githubPermalinkRegex *regexp.Regexp

	BotUserID string

	CommandHandlers map[string]CommandHandleFunc

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	router *mux.Router
}

// NewPlugin returns an instance of a Plugin.
func NewPlugin() *Plugin {
	p := &Plugin{
		githubPermalinkRegex: regexp.MustCompile(`https?://(?P<haswww>www\.)?github\.com/(?P<user>[\w-]+)/(?P<repo>[\w-]+)/blob/(?P<commit>\w+)/(?P<path>[\w-/.]+)#(?P<line>[\w-]+)?`),
	}

	p.CommandHandlers = map[string]CommandHandleFunc{
		"subscribe":   p.handleSubscribe,
		"unsubscribe": p.handleUnsubscribe,
		"disconnect":  p.handleDisconnect,
		"todo":        p.handleTodo,
		"me":          p.handleMe,
		"help":        p.handleHelp,
		"":            p.handleEmpty,
		"settings":    p.handleSettings,
	}

	return p
}

func (p *Plugin) githubConnect(token oauth2.Token) *github.Client {
	config := p.getConfiguration()

	ts := oauth2.StaticTokenSource(&token)
	tc := oauth2.NewClient(context.Background(), ts)

	if len(config.EnterpriseBaseURL) == 0 || len(config.EnterpriseUploadURL) == 0 {
		return github.NewClient(tc)
	}

	baseURL, _ := url.Parse(config.EnterpriseBaseURL)
	baseURL.Path = path.Join(baseURL.Path, "api", "v3")

	uploadURL, _ := url.Parse(config.EnterpriseUploadURL)
	uploadURL.Path = path.Join(uploadURL.Path, "api", "v3")

	client, err := github.NewEnterpriseClient(baseURL.String(), uploadURL.String(), tc)
	if err != nil {
		mlog.Error(err.Error())
		return github.NewClient(tc)
	}
	return client
}

func (p *Plugin) OnActivate() error {
	config := p.getConfiguration()

	if err := config.IsValid(); err != nil {
		return err
	}

	p.initializeAPI()

	if err := p.API.RegisterCommand(getCommand()); err != nil {
		return err
	}

	botID, err := p.Helpers.EnsureBot(&model.Bot{
		Username:    "github",
		DisplayName: "GitHub",
		Description: "Created by the GitHub plugin.",
	})
	if err != nil {
		return errors.Wrap(err, "failed to ensure github bot")
	}
	p.BotUserID = botID

	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		return errors.Wrap(err, "couldn't get bundle path")
	}

	profileImage, err := ioutil.ReadFile(filepath.Join(bundlePath, "assets", "profile.png"))
	if err != nil {
		return errors.Wrap(err, "couldn't read profile image")
	}

	appErr := p.API.SetProfileImage(botID, profileImage)
	if appErr != nil {
		return errors.Wrap(appErr, "couldn't set profile image")
	}

	registerGitHubToUsernameMappingCallback(p.getGitHubToUsernameMapping)

	return nil
}

func (p *Plugin) MessageWillBePosted(c *plugin.Context, post *model.Post) (*model.Post, string) {
	// If not enabled in config, ignore.
	config := p.getConfiguration()
	if !config.EnableCodePreview {
		return nil, ""
	}

	if post.UserId == "" {
		return nil, ""
	}

	msg := post.Message
	info, err := p.getGitHubUserInfo(post.UserId)
	if err != nil {
		p.API.LogError("error in getting user info", "error", err.Message)
		return nil, ""
	}
	// TODO: make this part of the Plugin struct and reuse it.
	ghClient := p.githubConnect(*info.Token)

	replacements := p.getReplacements(msg)
	post.Message = p.makeReplacements(msg, replacements, ghClient)
	return post, ""
}

func (p *Plugin) getOAuthConfig(privateAllowed bool) *oauth2.Config {
	config := p.getConfiguration()

	baseURL := p.getBaseURL()
	authURL, _ := url.Parse(baseURL)
	tokenURL, _ := url.Parse(baseURL)

	authURL.Path = path.Join(authURL.Path, "login", "oauth", "authorize")
	tokenURL.Path = path.Join(tokenURL.Path, "login", "oauth", "access_token")

	repo := github.ScopePublicRepo
	if config.EnablePrivateRepo && privateAllowed {
		// means that asks scope for private repositories
		repo = github.ScopeRepo
	}

	return &oauth2.Config{
		ClientID:     config.GitHubOAuthClientID,
		ClientSecret: config.GitHubOAuthClientSecret,
		Scopes:       []string{string(repo), string(github.ScopeNotifications), string(github.ScopeReadOrg)},
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL.String(),
			TokenURL: tokenURL.String(),
		},
	}
}

type GitHubUserInfo struct {
	UserID              string
	Token               *oauth2.Token
	GitHubUsername      string
	LastToDoPostAt      int64
	Settings            *UserSettings
	AllowedPrivateRepos bool
}

type UserSettings struct {
	SidebarButtons string `json:"sidebar_buttons"`
	DailyReminder  bool   `json:"daily_reminder"`
	Notifications  bool   `json:"notifications"`
}

func (p *Plugin) storeGitHubUserInfo(info *GitHubUserInfo) error {
	config := p.getConfiguration()

	encryptedToken, err := encrypt([]byte(config.EncryptionKey), info.Token.AccessToken)
	if err != nil {
		return err
	}

	info.Token.AccessToken = encryptedToken

	jsonInfo, err := json.Marshal(info)
	if err != nil {
		return err
	}

	if err := p.API.KVSet(info.UserID+githubTokenKey, jsonInfo); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) getGitHubUserInfo(userID string) (*GitHubUserInfo, *APIErrorResponse) {
	config := p.getConfiguration()

	var userInfo GitHubUserInfo

	infoBytes, appErr := p.API.KVGet(userID + githubTokenKey)
	if appErr != nil || infoBytes == nil {
		return nil, &APIErrorResponse{ID: apiErrorIDNotConnected, Message: "Must connect user account to GitHub first.", StatusCode: http.StatusBadRequest}
	}

	if err := json.Unmarshal(infoBytes, &userInfo); err != nil {
		return nil, &APIErrorResponse{ID: "", Message: "Unable to parse token.", StatusCode: http.StatusInternalServerError}
	}

	unencryptedToken, err := decrypt([]byte(config.EncryptionKey), userInfo.Token.AccessToken)
	if err != nil {
		mlog.Error(err.Error())
		return nil, &APIErrorResponse{ID: "", Message: "Unable to decrypt access token.", StatusCode: http.StatusInternalServerError}
	}

	userInfo.Token.AccessToken = unencryptedToken

	return &userInfo, nil
}

func (p *Plugin) storeGitHubToUserIDMapping(githubUsername, userID string) error {
	if err := p.API.KVSet(githubUsername+githubUsernameKey, []byte(userID)); err != nil {
		return fmt.Errorf("encountered error saving github username mapping")
	}
	return nil
}

func (p *Plugin) getGitHubToUserIDMapping(githubUsername string) string {
	userID, _ := p.API.KVGet(githubUsername + githubUsernameKey)
	return string(userID)
}

// getGitHubToUsernameMapping maps a GitHub username to the corresponding Mattermost username, if any.
func (p *Plugin) getGitHubToUsernameMapping(githubUsername string) string {
	user, _ := p.API.GetUser(p.getGitHubToUserIDMapping(githubUsername))
	if user == nil {
		return ""
	}

	return user.Username
}

func (p *Plugin) disconnectGitHubAccount(userID string) {
	userInfo, _ := p.getGitHubUserInfo(userID)
	if userInfo == nil {
		return
	}

	if err := p.API.KVDelete(userID + githubTokenKey); err != nil {
		p.API.LogWarn("Failed to delete github token", "userID", userID, "error", err.Error())
	}

	if err := p.API.KVDelete(userInfo.GitHubUsername + githubUsernameKey); err != nil {
		p.API.LogWarn("Failed to delete github token", "userID", userID, "error", err.Error())
	}

	user, err := p.API.GetUser(userID)
	if err != nil {
		p.API.LogWarn("Failed to get user props", "userID", userID, "error", err.Error())
	} else {
		_, ok := user.Props["git_user"]
		if ok {
			delete(user.Props, "git_user")
			_, err := p.API.UpdateUser(user)
			if err != nil {
				p.API.LogWarn("Failed to get update user props", "userID", userID, "error", err.Error())
			}
		}
	}

	p.API.PublishWebSocketEvent(
		wsEventDisconnect,
		nil,
		&model.WebsocketBroadcast{UserId: userID},
	)
}

// CreateBotDMPost posts a direct message using the bot account.
// Any error are not returned and instead logged.
func (p *Plugin) CreateBotDMPost(userID, message, postType string) {
	channel, err := p.API.GetDirectChannel(userID, p.BotUserID)
	if err != nil {
		p.API.LogWarn("Couldn't get bot's DM channel", "userID", userID, "error", err.Error())
		return
	}

	post := &model.Post{
		UserId:    p.BotUserID,
		ChannelId: channel.Id,
		Message:   message,
		Type:      postType,
	}

	if _, err := p.API.CreatePost(post); err != nil {
		p.API.LogWarn("Failed to create DM post", "userID", userID, "error", err.Error())
		return
	}
}

func (p *Plugin) PostToDo(info *GitHubUserInfo) {
	text, err := p.GetToDo(context.Background(), info.GitHubUsername, p.githubConnect(*info.Token))
	if err != nil {
		p.API.LogWarn("Failed to get todo text", "userID", info.UserID, "error", err.Error())
		return
	}

	p.CreateBotDMPost(info.UserID, text, "custom_git_todo")
}

func (p *Plugin) GetToDo(ctx context.Context, username string, githubClient *github.Client) (string, error) {
	config := p.getConfiguration()
	baseURL := p.getBaseURL()

	issueResults, _, err := githubClient.Search.Issues(ctx, getReviewSearchQuery(username, config.GitHubOrg), &github.SearchOptions{})
	if err != nil {
		return "", err
	}

	notifications, _, err := githubClient.Activity.ListNotifications(ctx, &github.NotificationListOptions{})
	if err != nil {
		return "", err
	}

	yourPrs, _, err := githubClient.Search.Issues(ctx, getYourPrsSearchQuery(username, config.GitHubOrg), &github.SearchOptions{})
	if err != nil {
		return "", err
	}

	yourAssignments, _, err := githubClient.Search.Issues(ctx, getYourAssigneeSearchQuery(username, config.GitHubOrg), &github.SearchOptions{})
	if err != nil {
		return "", err
	}

	text := "##### Unread Messages\n"

	notificationCount := 0
	notificationContent := ""
	for _, n := range notifications {
		if n.GetReason() == notificationReasonSubscribed {
			continue
		}

		if n.GetRepository() == nil {
			p.API.LogError("Unable to get repository for notification in todo list. Skipping.")
			continue
		}

		if p.checkOrg(n.GetRepository().GetOwner().GetLogin()) != nil {
			continue
		}

		notificationSubject := n.GetSubject()
		notificationType := notificationSubject.GetType()
		switch notificationType {
		case "RepositoryVulnerabilityAlert":
			message := fmt.Sprintf("[Vulnerability Alert for %v](%v)", n.GetRepository().GetFullName(), fixGithubNotificationSubjectURL(n.GetSubject().GetURL()))
			notificationContent += fmt.Sprintf("* %v\n", message)
		default:
			notificationTitle := notificationSubject.GetTitle()
			notificationURL := fixGithubNotificationSubjectURL(notificationSubject.GetURL())
			notificationContent += getToDoDisplayText(baseURL, notificationTitle, notificationURL, notificationType)
		}

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
		text += "You don't have any pull requests awaiting your review.\n"
	} else {
		text += fmt.Sprintf("You have %v pull requests awaiting your review:\n", issueResults.GetTotal())

		for _, pr := range issueResults.Issues {
			text += getToDoDisplayText(baseURL, pr.GetTitle(), pr.GetHTMLURL(), "")
		}
	}

	text += "##### Your Open Pull Requests\n"

	if yourPrs.GetTotal() == 0 {
		text += "You don't have any open pull requests.\n"
	} else {
		text += fmt.Sprintf("You have %v open pull requests:\n", yourPrs.GetTotal())

		for _, pr := range yourPrs.Issues {
			text += getToDoDisplayText(baseURL, pr.GetTitle(), pr.GetHTMLURL(), "")
		}
	}

	text += "##### Your Assignments\n"

	if yourAssignments.GetTotal() == 0 {
		text += "You don't have any assignments.\n"
	} else {
		text += fmt.Sprintf("You have %v assignments:\n", yourAssignments.GetTotal())

		for _, assign := range yourAssignments.Issues {
			text += getToDoDisplayText(baseURL, assign.GetTitle(), assign.GetHTMLURL(), "")
		}
	}

	return text, nil
}

func (p *Plugin) HasUnreads(info *GitHubUserInfo) bool {
	username := info.GitHubUsername
	ctx := context.Background()
	githubClient := p.githubConnect(*info.Token)
	config := p.getConfiguration()

	issues, _, err := githubClient.Search.Issues(ctx, getReviewSearchQuery(username, config.GitHubOrg), &github.SearchOptions{})
	if err != nil {
		mlog.Error(err.Error())
		return false
	}

	yourPrs, _, err := githubClient.Search.Issues(ctx, getYourPrsSearchQuery(username, config.GitHubOrg), &github.SearchOptions{})
	if err != nil {
		mlog.Error(err.Error())
		return false
	}

	yourAssignments, _, err := githubClient.Search.Issues(ctx, getYourAssigneeSearchQuery(username, config.GitHubOrg), &github.SearchOptions{})
	if err != nil {
		mlog.Error(err.Error())
		return false
	}

	relevantNotifications := false
	notifications, _, err := githubClient.Activity.ListNotifications(ctx, &github.NotificationListOptions{})
	if err != nil {
		mlog.Error(err.Error())
		return false
	}

	for _, n := range notifications {
		if n.GetReason() == notificationReasonSubscribed {
			continue
		}

		if n.GetRepository() == nil {
			p.API.LogError("Unable to get repository for notification in todo list. Skipping.")
			continue
		}

		if p.checkOrg(n.GetRepository().GetOwner().GetLogin()) != nil {
			continue
		}

		relevantNotifications = true
		break
	}

	if issues.GetTotal() == 0 && !relevantNotifications && yourPrs.GetTotal() == 0 && yourAssignments.GetTotal() == 0 {
		return false
	}

	return true
}

func (p *Plugin) checkOrg(org string) error {
	config := p.getConfiguration()

	configOrg := strings.TrimSpace(config.GitHubOrg)
	if configOrg != "" && configOrg != org {
		return fmt.Errorf("only repositories in the %v organization are supported", configOrg)
	}

	return nil
}

func (p *Plugin) isUserOrganizationMember(githubClient *github.Client, user *github.User, organization string) bool {
	if organization == "" {
		return false
	}

	isMember, _, err := githubClient.Organizations.IsMember(context.Background(), organization, *user.Login)
	if err != nil {
		mlog.Warn(err.Error())
		return false
	}

	return isMember
}

func (p *Plugin) sendRefreshEvent(userID string) {
	p.API.PublishWebSocketEvent(
		wsEventRefresh,
		nil,
		&model.WebsocketBroadcast{UserId: userID},
	)
}

func (p *Plugin) getBaseURL() string {
	config := p.getConfiguration()
	if config.EnterpriseBaseURL != "" {
		return config.EnterpriseBaseURL
	}

	return "https://github.com/"
}
