package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/google/go-github/v41/github"
	"github.com/gorilla/mux"
	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-plugin-api/experimental/bot/poster"
	"github.com/mattermost/mattermost-plugin-api/experimental/telemetry"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	root "github.com/mattermost/mattermost-plugin-github"
)

const (
	githubTokenKey       = "_githubtoken"
	githubOauthKey       = "githuboauthkey_"
	githubUsernameKey    = "_githubusername"
	githubPrivateRepoKey = "_githubprivate"

	mm34646MutexKey = "mm34646_token_reset_mutex"
	mm34646DoneKey  = "mm34646_token_reset_done"

	wsEventConnect    = "connect"
	wsEventDisconnect = "disconnect"
	// WSEventConfigUpdate is the WebSocket event to update the configurations on webapp.
	WSEventConfigUpdate = "config_update"
	wsEventRefresh      = "refresh"
	wsEventCreateIssue  = "createIssue"

	WSEventRefresh = "refresh"

	settingButtonsTeam   = "team"
	settingNotifications = "notifications"
	settingReminders     = "reminders"
	settingOn            = "on"
	settingOff           = "off"
	settingOnChange      = "on-change"

	notificationReasonSubscribed = "subscribed"
	dailySummary                 = "_dailySummary"

	chimeraGitHubAppIdentifier = "plugin-github"
)

var (
	Manifest model.Manifest = root.Manifest
)

type Plugin struct {
	plugin.MattermostPlugin
	client *pluginapi.Client

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *Configuration

	chimeraURL string

	router *mux.Router

	telemetryClient telemetry.Client
	tracker         telemetry.Tracker

	BotUserID   string
	poster      poster.Poster
	flowManager *FlowManager

	CommandHandlers map[string]CommandHandleFunc

	// githubPermalinkRegex is used to parse github permalinks in post messages.
	githubPermalinkRegex *regexp.Regexp

	webhookBroker *WebhookBroker
	oauthBroker   *OAuthBroker

	emojiMap map[string]string
}

// NewPlugin returns an instance of a Plugin.
func NewPlugin() *Plugin {
	p := &Plugin{
		githubPermalinkRegex: regexp.MustCompile(`https?://(?P<haswww>www\.)?github\.com/(?P<user>[\w-]+)/(?P<repo>[\w-.]+)/blob/(?P<commit>\w+)/(?P<path>[\w-/.]+)#(?P<line>[\w-]+)?`),
	}

	p.CommandHandlers = map[string]CommandHandleFunc{
		"subscriptions": p.handleSubscriptions,
		"subscribe":     p.handleSubscribe,
		"unsubscribe":   p.handleUnsubscribe,
		"disconnect":    p.handleDisconnect,
		"todo":          p.handleTodo,
		"mute":          p.handleMuteCommand,
		"me":            p.handleMe,
		"help":          p.handleHelp,
		"":              p.handleHelp,
		"settings":      p.handleSettings,
		"issue":         p.handleIssue,
	}

	p.createGithubEmojiMap()
	return p
}

func (p *Plugin) createGithubEmojiMap() {
	baseGithubEmojiMap := map[string]string{
		"+1":         "+1",
		"-1":         "-1",
		"thumbsup":   "+1",
		"thumbsdown": "-1",
		"laughing":   "laugh",
		"confused":   "confused",
		"heart":      "heart",
		"tada":       "hooray",
		"rocket":     "rocket",
		"eyes":       "eyes",
	}

	p.emojiMap = map[string]string{}
	for systemEmoji := range model.SystemEmojis {
		for mmBase, ghBase := range baseGithubEmojiMap {
			if strings.HasPrefix(systemEmoji, mmBase) {
				p.emojiMap[systemEmoji] = ghBase
			}
		}
	}
}

func (p *Plugin) GetGitHubClient(ctx context.Context, userID string) (*github.Client, error) {
	userInfo, apiErr := p.getGitHubUserInfo(userID)
	if apiErr != nil {
		return nil, apiErr
	}

	return p.githubConnectUser(ctx, userInfo), nil
}

func (p *Plugin) githubConnectUser(ctx context.Context, info *GitHubUserInfo) *github.Client {
	tok := *info.Token
	return p.githubConnectToken(tok)
}

func (p *Plugin) githubConnectToken(token oauth2.Token) *github.Client {
	config := p.getConfiguration()

	client, err := GetGitHubClient(token, config)
	if err != nil {
		p.API.LogWarn("Failed to create GitHub client", "error", err.Error())
		return nil
	}

	return client
}

func GetGitHubClient(token oauth2.Token, config *Configuration) (*github.Client, error) {
	ts := oauth2.StaticTokenSource(&token)
	tc := oauth2.NewClient(context.Background(), ts)

	return getGitHubClient(tc, config)
}

func getGitHubClient(authenticatedClient *http.Client, config *Configuration) (*github.Client, error) {
	if config.EnterpriseBaseURL == "" || config.EnterpriseUploadURL == "" {
		return github.NewClient(authenticatedClient), nil
	}

	baseURL, _ := url.Parse(config.EnterpriseBaseURL)
	baseURL.Path = path.Join(baseURL.Path, "api", "v3")

	uploadURL, _ := url.Parse(config.EnterpriseUploadURL)
	uploadURL.Path = path.Join(uploadURL.Path, "api", "v3")

	client, err := github.NewEnterpriseClient(baseURL.String(), uploadURL.String(), authenticatedClient)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (p *Plugin) setDefaultConfiguration() error {
	config := p.getConfiguration()

	changed, err := config.setDefaults(pluginapi.IsCloud(p.API.GetLicense()))
	if err != nil {
		return err
	}

	if changed {
		configMap, err := config.ToMap()
		if err != nil {
			return err
		}

		appErr := p.API.SavePluginConfig(configMap)
		if appErr != nil {
			return appErr
		}
	}

	return nil
}

func (p *Plugin) OnActivate() error {
	p.client = pluginapi.NewClient(p.API, p.Driver)

	siteURL := p.API.GetConfig().ServiceSettings.SiteURL
	if siteURL == nil || *siteURL == "" {
		return errors.New("siteURL is not set. Please set it and restart the plugin")
	}

	err := p.setDefaultConfiguration()
	if err != nil {
		return errors.Wrap(err, "failed to set default configuration")
	}

	p.registerChimeraURL()
	if p.getConfiguration().UsePreregisteredApplication && p.chimeraURL == "" {
		return errors.New("cannot use pre-registered application if Chimera URL is not set or empty. " +
			"For now using pre-registered application is intended for Cloud instances only. " +
			"If you are running on-prem disable the setting and use a custom application, otherwise set PluginSettings.ChimeraOAuthProxyURL")
	}

	p.initializeAPI()
	p.initializeTelemetry()

	p.webhookBroker = NewWebhookBroker(p.sendGitHubPingEvent)
	p.oauthBroker = NewOAuthBroker(p.sendOAuthCompleteEvent)

	botID, err := p.client.Bot.EnsureBot(&model.Bot{
		OwnerId:     Manifest.Id, // Workaround to support older server version affected by https://github.com/mattermost/mattermost-server/pull/21560
		Username:    "github",
		DisplayName: "GitHub",
		Description: "Created by the GitHub plugin.",
	}, pluginapi.ProfileImagePath(filepath.Join("assets", "profile.png")))
	if err != nil {
		return errors.Wrap(err, "failed to ensure github bot")
	}
	p.BotUserID = botID

	p.poster = poster.NewPoster(&p.client.Post, p.BotUserID)
	p.flowManager = p.NewFlowManager()

	registerGitHubToUsernameMappingCallback(p.getGitHubToUsernameMapping)

	go func() {
		resetErr := p.forceResetAllMM34646()
		if resetErr != nil {
			p.API.LogDebug("failed to reset user tokens", "error", resetErr.Error())
		}
	}()
	return nil
}

func (p *Plugin) OnDeactivate() error {
	p.webhookBroker.Close()
	p.oauthBroker.Close()
	if err := p.telemetryClient.Close(); err != nil {
		p.API.LogWarn("Telemetry client failed to close", "error", err.Error())
	}
	return nil
}

func (p *Plugin) getPostPropsForReaction(reaction *model.Reaction) (org, repo string, id float64, objectType string, ok bool) {
	post, err := p.client.Post.GetPost(reaction.PostId)
	if err != nil {
		p.API.LogDebug("Error fetching post for reaction", "error", err.Error())
		return org, repo, id, objectType, false
	}

	// Getting the Github repository from notification post props
	repo, ok = post.GetProp(postPropGithubRepo).(string)
	if !ok || repo == "" {
		return org, repo, id, objectType, false
	}

	orgRepo := strings.Split(repo, "/")
	if len(orgRepo) != 2 {
		p.API.LogDebug("Invalid organization repository")
		return org, repo, id, objectType, false
	}

	org, repo = orgRepo[0], orgRepo[1]

	// Getting the Github object id from notification post props
	id, ok = post.GetProp(postPropGithubObjectID).(float64)
	if !ok || id == 0 {
		return org, repo, id, objectType, false
	}

	// Getting the Github object type from notification post props
	objectType, ok = post.GetProp(postPropGithubObjectType).(string)
	if !ok || objectType == "" {
		return org, repo, id, objectType, false
	}

	return org, repo, id, objectType, true
}

func (p *Plugin) ReactionHasBeenAdded(c *plugin.Context, reaction *model.Reaction) {
	githubEmoji := p.emojiMap[reaction.EmojiName]
	if githubEmoji == "" {
		p.API.LogWarn("Emoji is not supported by Github", "Emoji", reaction.EmojiName)
		return
	}

	owner, repo, id, objectType, ok := p.getPostPropsForReaction(reaction)
	if !ok {
		return
	}

	info, appErr := p.getGitHubUserInfo(reaction.UserId)
	if appErr != nil {
		if appErr.ID != apiErrorIDNotConnected {
			p.API.LogDebug("Error in getting user info", "error", appErr.Error())
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	ghClient := p.githubConnectUser(ctx, info)
	switch objectType {
	case githubObjectTypeIssueComment:
		if _, _, err := ghClient.Reactions.CreateIssueCommentReaction(context.Background(), owner, repo, int64(id), githubEmoji); err != nil {
			p.API.LogDebug("Error occurred while creating issue comment reaction", "error", err.Error())
			return
		}
	case githubObjectTypeIssue:
		if _, _, err := ghClient.Reactions.CreateIssueReaction(context.Background(), owner, repo, int(id), githubEmoji); err != nil {
			p.API.LogDebug("Error occurred while creating issue reaction", "error", err.Error())
			return
		}
	case githubObjectTypePRReviewComment:
		if _, _, err := ghClient.Reactions.CreatePullRequestCommentReaction(context.Background(), owner, repo, int64(id), githubEmoji); err != nil {
			p.API.LogDebug("Error occurred while creating PR review comment reaction", "error", err.Error())
			return
		}
	default:
		return
	}
}

func (p *Plugin) ReactionHasBeenRemoved(c *plugin.Context, reaction *model.Reaction) {
	githubEmoji := p.emojiMap[reaction.EmojiName]
	if githubEmoji == "" {
		p.API.LogWarn("Emoji is not supported by Github", "Emoji", reaction.EmojiName)
		return
	}

	owner, repo, id, objectType, ok := p.getPostPropsForReaction(reaction)
	if !ok {
		return
	}

	info, appErr := p.getGitHubUserInfo(reaction.UserId)
	if appErr != nil {
		if appErr.ID != apiErrorIDNotConnected {
			p.API.LogDebug("Error in getting user info", "error", appErr.Error())
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	ghClient := p.githubConnectUser(ctx, info)
	switch objectType {
	case githubObjectTypeIssueComment:
		reactions, _, err := ghClient.Reactions.ListIssueCommentReactions(context.Background(), owner, repo, int64(id), &github.ListOptions{})
		if err != nil {
			p.API.LogDebug("Error getting issue comment reaction list", "error", err.Error())
			return
		}

		for _, reactionObj := range reactions {
			if info.UserID == reaction.UserId && p.emojiMap[reaction.EmojiName] == reactionObj.GetContent() {
				if _, err = ghClient.Reactions.DeleteIssueCommentReaction(context.Background(), owner, repo, int64(id), reactionObj.GetID()); err != nil {
					p.API.LogDebug("Error occurred while removing issue comment reaction", "error", err.Error())
				}
				return
			}
		}
	case githubObjectTypeIssue:
		reactions, _, err := ghClient.Reactions.ListIssueReactions(context.Background(), owner, repo, int(id), &github.ListOptions{})
		if err != nil {
			p.API.LogDebug("Error getting issue reaction list", "error", err.Error())
			return
		}

		for _, reactionObj := range reactions {
			if info.UserID == reaction.UserId && p.emojiMap[reaction.EmojiName] == reactionObj.GetContent() {
				if _, err = ghClient.Reactions.DeleteIssueReaction(context.Background(), owner, repo, int(id), reactionObj.GetID()); err != nil {
					p.API.LogDebug("Error occurred while removing issue reaction", "error", err.Error())
				}
				return
			}
		}
	case githubObjectTypePRReviewComment:
		reactions, _, err := ghClient.Reactions.ListPullRequestCommentReactions(context.Background(), owner, repo, int64(id), &github.ListOptions{})
		if err != nil {
			p.API.LogDebug("Error getting PR review comment reaction list", "error", err.Error())
			return
		}

		for _, reactionObj := range reactions {
			if info.UserID == reaction.UserId && p.emojiMap[reaction.EmojiName] == reactionObj.GetContent() {
				if _, err = ghClient.Reactions.DeletePullRequestCommentReaction(context.Background(), owner, repo, int64(id), reactionObj.GetID()); err != nil {
					p.API.LogDebug("Error occurred while removing PR review comment reaction", "error", err.Error())
				}
				return
			}
		}
	default:
		return
	}
}

func (p *Plugin) OnInstall(c *plugin.Context, event model.OnInstallEvent) error {
	// Don't start wizard if OAuth is configured
	if p.getConfiguration().IsOAuthConfigured() {
		return nil
	}

	return p.flowManager.StartSetupWizard(event.UserId, "")
}

func (p *Plugin) OnSendDailyTelemetry() {
	p.SendDailyTelemetry()
}

func (p *Plugin) OnPluginClusterEvent(c *plugin.Context, ev model.PluginClusterEvent) {
	p.HandleClusterEvent(ev)
}

// registerChimeraURL fetches the Chimera URL from server settings or env var and sets it in the plugin object.
func (p *Plugin) registerChimeraURL() {
	chimeraURLSetting := p.API.GetConfig().PluginSettings.ChimeraOAuthProxyURL
	if chimeraURLSetting != nil {
		p.chimeraURL = *chimeraURLSetting
	}
}

func (p *Plugin) MessageWillBePosted(c *plugin.Context, post *model.Post) (*model.Post, string) {
	// If not enabled in config, ignore.
	config := p.getConfiguration()
	if config.EnableCodePreview == "disable" {
		return nil, ""
	}

	if post.UserId == "" {
		return nil, ""
	}

	client := pluginapi.NewClient(p.API, p.Driver)

	shouldProcessMessage, err := client.Post.ShouldProcessMessage(post)
	if err != nil {
		p.API.LogWarn("Error while checking if the message should be processed", "error", err.Error())
		return nil, ""
	}

	if !shouldProcessMessage {
		return nil, ""
	}

	msg := post.Message
	info, appErr := p.getGitHubUserInfo(post.UserId)
	if appErr != nil {
		if appErr.ID != apiErrorIDNotConnected {
			p.API.LogWarn("Error in getting user info", "error", appErr.Message)
		}
		return nil, ""
	}
	// TODO: make this part of the Plugin struct and reuse it.
	ghClient := p.githubConnectUser(context.Background(), info)

	replacements := p.getReplacements(msg)
	post.Message = p.makeReplacements(msg, replacements, ghClient)
	return post, ""
}

func (p *Plugin) getOAuthConfig(privateAllowed bool) *oauth2.Config {
	config := p.getConfiguration()

	repo := github.ScopePublicRepo
	if config.EnablePrivateRepo && privateAllowed {
		// means that asks scope for private repositories
		repo = github.ScopeRepo
	}
	scopes := []string{string(repo), string(github.ScopeNotifications), string(github.ScopeReadOrg), string(github.ScopeAdminOrgHook)}

	if config.UsePreregisteredApplication {
		p.API.LogDebug("Using Chimera Proxy OAuth configuration")
		return p.getOAuthConfigForChimeraApp(scopes)
	}

	baseURL := config.getBaseURL()
	authURL, _ := url.Parse(baseURL)
	tokenURL, _ := url.Parse(baseURL)

	authURL.Path = path.Join(authURL.Path, "login", "oauth", "authorize")
	tokenURL.Path = path.Join(tokenURL.Path, "login", "oauth", "access_token")

	return &oauth2.Config{
		ClientID:     config.GitHubOAuthClientID,
		ClientSecret: config.GitHubOAuthClientSecret,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:   authURL.String(),
			TokenURL:  tokenURL.String(),
			AuthStyle: oauth2.AuthStyleInHeader,
		},
	}
}

func (p *Plugin) getOAuthConfigForChimeraApp(scopes []string) *oauth2.Config {
	baseURL := fmt.Sprintf("%s/v1/github/%s", p.chimeraURL, chimeraGitHubAppIdentifier)
	authURL, _ := url.Parse(baseURL)
	tokenURL, _ := url.Parse(baseURL)

	authURL.Path = path.Join(authURL.Path, "oauth", "authorize")
	tokenURL.Path = path.Join(tokenURL.Path, "oauth", "token")

	redirectURL, _ := url.Parse(fmt.Sprintf("%s/plugins/github/oauth/complete", *p.API.GetConfig().ServiceSettings.SiteURL))

	return &oauth2.Config{
		ClientID:     "placeholder",
		ClientSecret: "placeholder",
		Scopes:       scopes,
		RedirectURL:  redirectURL.String(),
		Endpoint: oauth2.Endpoint{
			AuthURL:   authURL.String(),
			TokenURL:  tokenURL.String(),
			AuthStyle: oauth2.AuthStyleInHeader,
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

	// MM34646ResetTokenDone is set for a user whose token has been reset for MM-34646.
	MM34646ResetTokenDone bool
}

type UserSettings struct {
	SidebarButtons        string `json:"sidebar_buttons"`
	DailyReminder         bool   `json:"daily_reminder"`
	DailyReminderOnChange bool   `json:"daily_reminder_on_change"`
	Notifications         bool   `json:"notifications"`
}

func (p *Plugin) storeGitHubUserInfo(info *GitHubUserInfo) error {
	config := p.getConfiguration()

	encryptedToken, err := encrypt([]byte(config.EncryptionKey), info.Token.AccessToken)
	if err != nil {
		return errors.Wrap(err, "error occurred while encrypting access token")
	}

	info.Token.AccessToken = encryptedToken

	jsonInfo, err := json.Marshal(info)
	if err != nil {
		return errors.Wrap(err, "error while converting user info to json")
	}

	if err := p.API.KVSet(info.UserID+githubTokenKey, jsonInfo); err != nil {
		return errors.Wrap(err, "error occurred while trying to store user info into KV store")
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
		p.API.LogWarn("Failed to decrypt access token", "error", err.Error())
		return nil, &APIErrorResponse{ID: "", Message: "Unable to decrypt access token.", StatusCode: http.StatusInternalServerError}
	}

	userInfo.Token.AccessToken = unencryptedToken

	return &userInfo, nil
}

func (p *Plugin) storeGitHubToUserIDMapping(githubUsername, userID string) error {
	if err := p.API.KVSet(githubUsername+githubUsernameKey, []byte(userID)); err != nil {
		return errors.New("encountered error saving github username mapping")
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

	if appErr := p.API.KVDelete(userID + githubTokenKey); appErr != nil {
		p.API.LogWarn("Failed to delete github token from KV store", "userID", userID, "error", appErr.Error())
	}

	if appErr := p.API.KVDelete(userInfo.GitHubUsername + githubUsernameKey); appErr != nil {
		p.API.LogWarn("Failed to delete github token from KV store", "userID", userID, "error", appErr.Error())
	}

	user, appErr := p.API.GetUser(userID)
	if appErr != nil {
		p.API.LogWarn("Failed to get user props", "userID", userID, "error", appErr.Error())
	} else {
		_, ok := user.Props["git_user"]
		if ok {
			delete(user.Props, "git_user")
			_, appErr := p.API.UpdateUser(user)
			if appErr != nil {
				p.API.LogWarn("Failed to get update user props", "userID", userID, "error", appErr.Error())
			}
		}
	}

	p.API.PublishWebSocketEvent(
		wsEventDisconnect,
		nil,
		&model.WebsocketBroadcast{UserId: userID},
	)
}

func (p *Plugin) openIssueCreateModal(userID string, channelID string, title string) {
	p.API.PublishWebSocketEvent(
		wsEventCreateIssue,
		map[string]interface{}{
			"title":      title,
			"channel_id": channelID,
		},
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
		p.API.LogWarn("Failed to create DM post", "userID", userID, "post", post, "error", err.Error())
		return
	}
}

func (p *Plugin) CheckIfDuplicateDailySummary(userID, text string) (bool, error) {
	previousSummary, err := p.GetDailySummaryText(userID)
	if err != nil {
		return false, err
	}
	if previousSummary == text {
		return true, nil
	}

	return false, nil
}

func (p *Plugin) StoreDailySummaryText(userID, summaryText string) error {
	if err := p.API.KVSet(userID+dailySummary, []byte(summaryText)); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) GetDailySummaryText(userID string) (string, error) {
	summaryByte, err := p.API.KVGet(userID + dailySummary)
	if err != nil {
		return "", err
	}

	return string(summaryByte), nil
}

func (p *Plugin) PostToDo(info *GitHubUserInfo, userID string) error {
	ctx := context.Background()
	text, err := p.GetToDo(ctx, info.GitHubUsername, p.githubConnectUser(ctx, info))
	if err != nil {
		return err
	}

	if info.Settings.DailyReminderOnChange {
		isSameSummary, err := p.CheckIfDuplicateDailySummary(userID, text)
		if err != nil {
			return err
		}
		if isSameSummary {
			return nil
		}
		err = p.StoreDailySummaryText(userID, text)
		if err != nil {
			return err
		}
	}
	p.CreateBotDMPost(info.UserID, text, "custom_git_todo")
	return nil
}

func (p *Plugin) GetToDo(ctx context.Context, username string, githubClient *github.Client) (string, error) {
	config := p.getConfiguration()
	baseURL := config.getBaseURL()

	issueResults, _, err := githubClient.Search.Issues(ctx, getReviewSearchQuery(username, config.GitHubOrg), &github.SearchOptions{})
	if err != nil {
		return "", errors.Wrap(err, "Error occurred while searching for reviews")
	}

	notifications, _, err := githubClient.Activity.ListNotifications(ctx, &github.NotificationListOptions{})
	if err != nil {
		return "", errors.Wrap(err, "error occurred while listing notifications")
	}

	yourPrs, _, err := githubClient.Search.Issues(ctx, getYourPrsSearchQuery(username, config.GitHubOrg), &github.SearchOptions{})
	if err != nil {
		return "", errors.Wrap(err, "error occurred while searching for PRs")
	}

	yourAssignments, _, err := githubClient.Search.Issues(ctx, getYourAssigneeSearchQuery(username, config.GitHubOrg), &github.SearchOptions{})
	if err != nil {
		return "", errors.Wrap(err, "error occurred while searching for assignments")
	}

	text := "##### Unread Messages\n"

	notificationCount := 0
	notificationContent := ""
	for _, n := range notifications {
		if n.GetReason() == notificationReasonSubscribed {
			continue
		}

		if n.GetRepository() == nil {
			p.API.LogWarn("Unable to get repository for notification in todo list. Skipping.")
			continue
		}

		if p.checkOrg(n.GetRepository().GetOwner().GetLogin()) != nil {
			continue
		}

		notificationSubject := n.GetSubject()
		notificationType := notificationSubject.GetType()
		switch notificationType {
		case "RepositoryVulnerabilityAlert":
			message := fmt.Sprintf("[Vulnerability Alert for %v](%v)", n.GetRepository().GetFullName(), fixGithubNotificationSubjectURL(n.GetSubject().GetURL(), ""))
			notificationContent += fmt.Sprintf("* %v\n", message)
		default:
			issueURL := n.GetSubject().GetURL()
			issueNumIndex := strings.LastIndex(issueURL, "/")
			issueNum := issueURL[issueNumIndex+1:]
			subjectURL := n.GetSubject().GetURL()
			if n.GetSubject().GetLatestCommentURL() != "" {
				subjectURL = n.GetSubject().GetLatestCommentURL()
			}

			notificationTitle := notificationSubject.GetTitle()
			notificationURL := fixGithubNotificationSubjectURL(subjectURL, issueNum)
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
	githubClient := p.githubConnectUser(ctx, info)
	config := p.getConfiguration()

	query := getReviewSearchQuery(username, config.GitHubOrg)
	issues, _, err := githubClient.Search.Issues(ctx, query, &github.SearchOptions{})
	if err != nil {
		p.API.LogWarn("Failed to search for review", "query", query, "error", err.Error())
		return false
	}

	query = getYourPrsSearchQuery(username, config.GitHubOrg)
	yourPrs, _, err := githubClient.Search.Issues(ctx, query, &github.SearchOptions{})
	if err != nil {
		p.API.LogWarn("Failed to search for PRs", "query", query, "error", "error", err.Error())
		return false
	}

	query = getYourAssigneeSearchQuery(username, config.GitHubOrg)
	yourAssignments, _, err := githubClient.Search.Issues(ctx, query, &github.SearchOptions{})
	if err != nil {
		p.API.LogWarn("Failed to search for assignments", "query", query, "error", "error", err.Error())
		return false
	}

	relevantNotifications := false
	notifications, _, err := githubClient.Activity.ListNotifications(ctx, &github.NotificationListOptions{})
	if err != nil {
		p.API.LogWarn("Failed to list notifications", "error", err.Error())
		return false
	}

	for _, n := range notifications {
		if n.GetReason() == notificationReasonSubscribed {
			continue
		}

		if n.GetRepository() == nil {
			p.API.LogWarn("Unable to get repository for notification in todo list. Skipping.")
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
	if configOrg != "" && configOrg != org && strings.ToLower(configOrg) != org {
		return errors.Errorf("only repositories in the %v organization are supported", configOrg)
	}

	return nil
}

func (p *Plugin) isUserOrganizationMember(githubClient *github.Client, user *github.User, organization string) bool {
	if organization == "" {
		return false
	}

	isMember, _, err := githubClient.Organizations.IsMember(context.Background(), organization, *user.Login)
	if err != nil {
		p.API.LogWarn("Failled to check if user is org member", "GitHub username", *user.Login, "error", err.Error())
		return false
	}

	return isMember
}

func (p *Plugin) isOrganizationLocked() bool {
	config := p.getConfiguration()
	configOrg := strings.TrimSpace(config.GitHubOrg)

	return configOrg != ""
}

func (p *Plugin) sendRefreshEvent(userID string) {
	p.API.PublishWebSocketEvent(
		wsEventRefresh,
		nil,
		&model.WebsocketBroadcast{UserId: userID},
	)
}

// getUsername returns the GitHub username for a given Mattermost user,
// if the user is connected to GitHub via this plugin.
// Otherwise it return the Mattermost username. It will be escaped via backticks.
func (p *Plugin) getUsername(mmUserID string) (string, error) {
	info, apiEr := p.getGitHubUserInfo(mmUserID)
	if apiEr != nil {
		if apiEr.ID != apiErrorIDNotConnected {
			return "", apiEr
		}

		user, appEr := p.API.GetUser(mmUserID)
		if appEr != nil {
			return "", appEr
		}

		return fmt.Sprintf("`@%s`", user.Username), nil
	}

	return "@" + info.GitHubUsername, nil
}
