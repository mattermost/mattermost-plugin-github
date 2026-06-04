package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/google/go-github/v54/github"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/mattermost/mattermost/server/public/pluginapi/experimental/bot/logger"
	"github.com/mattermost/mattermost/server/public/pluginapi/experimental/bot/poster"
	"github.com/mattermost/mattermost/server/public/pluginapi/experimental/telemetry"
)

const (
	forgejoTokenKey       = "_forgejotoken"
	forgejoOauthKey       = "forgejooauthkey_"
	forgejoUsernameKey    = "_forgejousername"
	forgejoPrivateRepoKey = "_forgejoprivate"

	wsEventConnect    = "connect"
	wsEventDisconnect = "disconnect"
	// WSEventConfigUpdate is the WebSocket event to update the configurations on webapp.
	WSEventConfigUpdate = "config_update"
	wsEventRefresh      = "refresh"
	wsEventCreateIssue  = "createIssue"

	settingButtonsTeam       = "team"
	settingNotifications     = "notifications"
	settingTeamNotifications = "team-review-notifications"
	settingReminders         = "reminders"
	settingOn                = "on"
	settingOff               = "off"
	settingOnChange          = "on-change"
	settingExclude           = "exclude"

	dailySummary = "_dailySummary"
)

var (
	// testOAuthServerURL is the URL for the oauthServer used for testing purposes
	// It should be set through ldflags when compiling for E2E, and keep it blank otherwise
	testOAuthServerURL = ""
)

type KvStore interface {
	Set(key string, value any, options ...pluginapi.KVSetOption) (bool, error)
	ListKeys(page int, count int, options ...pluginapi.ListKeysOption) ([]string, error)
	Get(key string, o any) error
	Delete(key string) error
	SetAtomicWithRetries(key string, valueFunc func(oldValue []byte) (newValue interface{}, err error)) error
}

type Plugin struct {
	plugin.MattermostPlugin
	client *pluginapi.Client

	store KvStore

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *Configuration

	router *mux.Router

	telemetryClient telemetry.Client
	tracker         telemetry.Tracker

	BotUserID   string
	poster      poster.Poster
	flowManager *FlowManager

	CommandHandlers map[string]CommandHandleFunc

	// forgejoPermalinkRegex is used to parse github permalinks in post messages.
	forgejoPermalinkRegex *regexp.Regexp

	webhookBroker *WebhookBroker
	oauthBroker   *OAuthBroker

	emojiMap map[string]string
}

// NewPlugin returns an instance of a Plugin.
func NewPlugin() *Plugin {
	p := &Plugin{
		forgejoPermalinkRegex: regexp.MustCompile(`https?://(?P<haswww>www\.)?forgejo\.pyn\.ru/(?P<user>[\w-]+)/(?P<repo>[\w-.]+)/blob/(?P<commit>[\w-]+)/(?P<path>[\w-/.]+)#(?P<line>[\w-]+)?`),
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

func (p *Plugin) ensurePluginAPIClient() {
	if p.client == nil {
		p.client = pluginapi.NewClient(p.API, p.Driver)
		p.store = &p.client.KV
	}
}

func (p *Plugin) GetGitHubClient(ctx context.Context, userID string) (*github.Client, error) {
	userInfo, apiErr := p.getGitHubUserInfo(userID)
	if apiErr != nil {
		return nil, apiErr
	}

	return p.githubConnectUser(ctx, userInfo), nil
}

func (p *Plugin) githubConnectUser(_ context.Context, info *ForgejoUserInfo) *github.Client {
	tok := *info.Token
	return p.githubConnectToken(tok)
}

func (p *Plugin) forgejoConnect(info *ForgejoUserInfo) *http.Client {
	config, err := p.getOAuthConfig()
	if err != nil {
		p.client.Log.Error("Failed to create OAuth config", "error", err.Error())
		return nil
	}
	tok := *info.Token
	return config.Client(context.Background(), &tok)
}

func (p *Plugin) githubConnectToken(token oauth2.Token) *github.Client {
	config := p.getConfiguration()

	client, err := GetGitHubClient(token, config)
	if err != nil {
		p.client.Log.Warn("Failed to create Forgejo client", "error", err.Error())
		return nil
	}

	return client
}

func GetGitHubClient(token oauth2.Token, config *Configuration) (*github.Client, error) {
	oauthConfig, err := getOauthConfig(config)
	if err != nil {
		return nil, err
	}
	tc := oauthConfig.Client(context.Background(), &token)
	return getGitHubClient(tc, config)
}

func getGitHubClient(authenticatedClient *http.Client, config *Configuration) (*github.Client, error) {
	if config.BaseURL == "" || config.UploadURL == "" {
		return github.NewClient(authenticatedClient), nil
	}
	baseURL, err := url.JoinPath(config.BaseURL, "api", "v1")
	if err != nil {
		return nil, err
	}

	uploadURL, err := url.JoinPath(config.UploadURL, "api", "v1")
	if err != nil {
		return nil, err
	}

	client, err := github.NewEnterpriseClient(baseURL, uploadURL, authenticatedClient)
	if err != nil {
		return nil, err
	}

	client.BaseURL.Path = "/api/v1/"

	return client, nil
}

func (p *Plugin) setDefaultConfiguration() error {
	config := p.getConfiguration()

	changed, err := config.setDefaults(pluginapi.IsCloud(p.client.System.GetLicense()))
	if err != nil {
		return err
	}

	if changed {
		configMap, err := config.ToMap()
		if err != nil {
			return err
		}

		err = p.client.Configuration.SavePluginConfig(configMap)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Plugin) OnActivate() error {
	p.ensurePluginAPIClient()

	_, err := getSiteURL(p.client)
	if err != nil {
		return err
	}

	err = p.setDefaultConfiguration()
	if err != nil {
		return errors.Wrap(err, "failed to set default configuration")
	}

	p.initializeAPI()
	p.initializeTelemetry()

	p.webhookBroker = NewWebhookBroker(p.sendGitHubPingEvent)
	p.oauthBroker = NewOAuthBroker(p.sendOAuthCompleteEvent)

	botID, err := p.client.Bot.EnsureBot(&model.Bot{
		OwnerId:     Manifest.Id, // Workaround to support older server version affected by https://github.com/mattermost/mattermost-server/pull/21560
		Username:    "forgejo",
		DisplayName: "Forgejo",
		Description: "Created by the Forgejo plugin.",
	}, pluginapi.ProfileImagePath(filepath.Join("assets", "profile.png")))
	if err != nil {
		return errors.Wrap(err, "failed to ensure forgejo bot")
	}
	p.BotUserID = botID

	p.poster = poster.NewPoster(&p.client.Post, p.BotUserID)
	flowManager, err := p.NewFlowManager()
	if err != nil {
		return errors.Wrap(err, "failed to create flow manager")
	}
	p.flowManager = flowManager

	registerForgejoToUsernameMappingCallback(p.getGitHubToUsernameMapping)
	return nil
}

func (p *Plugin) OnDeactivate() error {
	p.webhookBroker.Close()
	p.oauthBroker.Close()
	if err := p.telemetryClient.Close(); err != nil {
		p.client.Log.Warn("Telemetry client failed to close", "error", err.Error())
	}
	return nil
}

func (p *Plugin) getPostPropsForReaction(reaction *model.Reaction) (org, repo string, id float64, objectType string, ok bool) {
	post, err := p.client.Post.GetPost(reaction.PostId)
	if err != nil {
		p.client.Log.Debug("Error fetching post for reaction", "error", err.Error())
		return org, repo, id, objectType, false
	}

	// Getting the Forgejo repository from notification post props
	repo, ok = post.GetProp(postPropForgejoRepo).(string)
	if !ok || repo == "" {
		return org, repo, id, objectType, false
	}

	orgRepo := strings.Split(repo, "/")
	if len(orgRepo) != 2 {
		p.client.Log.Debug("Invalid organization repository")
		return org, repo, id, objectType, false
	}

	org, repo = orgRepo[0], orgRepo[1]

	// Getting the Forgejo object id from notification post props
	id, ok = post.GetProp(postPropForgejoObjectID).(float64)
	if !ok || id == 0 {
		return org, repo, id, objectType, false
	}

	// Getting the Forgejo object type from notification post props
	objectType, ok = post.GetProp(postPropForgejoObjectType).(string)
	if !ok || objectType == "" {
		return org, repo, id, objectType, false
	}

	return org, repo, id, objectType, true
}

func (p *Plugin) ReactionHasBeenAdded(c *plugin.Context, reaction *model.Reaction) {
	githubEmoji := p.emojiMap[reaction.EmojiName]
	if githubEmoji == "" {
		return
	}

	owner, repo, id, objectType, ok := p.getPostPropsForReaction(reaction)
	if !ok {
		return
	}

	info, appErr := p.getGitHubUserInfo(reaction.UserId)
	if appErr != nil {
		if appErr.ID != apiErrorIDNotConnected {
			p.client.Log.Debug("Error in getting user info", "error", appErr.Error())
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	ghClient := p.githubConnectUser(ctx, info)
	switch objectType {
	case forgejoObjectTypeIssueComment:
		if _, _, err := ghClient.Reactions.CreateIssueCommentReaction(context.Background(), owner, repo, int64(id), githubEmoji); err != nil {
			p.client.Log.Debug("Error occurred while creating issue comment reaction", "error", err.Error())
			return
		}
	case forgejoObjectTypeIssue:
		if _, _, err := ghClient.Reactions.CreateIssueReaction(context.Background(), owner, repo, int(id), githubEmoji); err != nil {
			p.client.Log.Debug("Error occurred while creating issue reaction", "error", err.Error())
			return
		}
	case forgejoObjectTypePRReviewComment:
		if _, _, err := ghClient.Reactions.CreatePullRequestCommentReaction(context.Background(), owner, repo, int64(id), githubEmoji); err != nil {
			p.client.Log.Debug("Error occurred while creating PR review comment reaction", "error", err.Error())
			return
		}
	default:
		return
	}
}

func (p *Plugin) ReactionHasBeenRemoved(c *plugin.Context, reaction *model.Reaction) {
	githubEmoji := p.emojiMap[reaction.EmojiName]
	if githubEmoji == "" {
		return
	}

	owner, repo, id, objectType, ok := p.getPostPropsForReaction(reaction)
	if !ok {
		return
	}

	info, appErr := p.getGitHubUserInfo(reaction.UserId)
	if appErr != nil {
		if appErr.ID != apiErrorIDNotConnected {
			p.client.Log.Debug("Error in getting user info", "error", appErr.Error())
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	ghClient := p.githubConnectUser(ctx, info)
	switch objectType {
	case forgejoObjectTypeIssueComment:
		reactions, _, err := ghClient.Reactions.ListIssueCommentReactions(context.Background(), owner, repo, int64(id), &github.ListOptions{})
		if err != nil {
			p.client.Log.Debug("Error getting issue comment reaction list", "error", err.Error())
			return
		}

		for _, reactionObj := range reactions {
			if info.UserID == reaction.UserId && p.emojiMap[reaction.EmojiName] == reactionObj.GetContent() {
				if _, err = ghClient.Reactions.DeleteIssueCommentReaction(context.Background(), owner, repo, int64(id), reactionObj.GetID()); err != nil {
					p.client.Log.Debug("Error occurred while removing issue comment reaction", "error", err.Error())
				}
				return
			}
		}
	case forgejoObjectTypeIssue:
		reactions, _, err := ghClient.Reactions.ListIssueReactions(context.Background(), owner, repo, int(id), &github.ListOptions{})
		if err != nil {
			p.client.Log.Debug("Error getting issue reaction list", "error", err.Error())
			return
		}

		for _, reactionObj := range reactions {
			if info.UserID == reaction.UserId && p.emojiMap[reaction.EmojiName] == reactionObj.GetContent() {
				if _, err = ghClient.Reactions.DeleteIssueReaction(context.Background(), owner, repo, int(id), reactionObj.GetID()); err != nil {
					p.client.Log.Debug("Error occurred while removing issue reaction", "error", err.Error())
				}
				return
			}
		}
	case forgejoObjectTypePRReviewComment:
		reactions, _, err := ghClient.Reactions.ListPullRequestCommentReactions(context.Background(), owner, repo, int64(id), &github.ListOptions{})
		if err != nil {
			p.client.Log.Debug("Error getting PR review comment reaction list", "error", err.Error())
			return
		}

		for _, reactionObj := range reactions {
			if info.UserID == reaction.UserId && p.emojiMap[reaction.EmojiName] == reactionObj.GetContent() {
				if _, err = ghClient.Reactions.DeletePullRequestCommentReaction(context.Background(), owner, repo, int64(id), reactionObj.GetID()); err != nil {
					p.client.Log.Debug("Error occurred while removing PR review comment reaction", "error", err.Error())
				}
				return
			}
		}
	default:
		return
	}
}

func (p *Plugin) OnInstall(c *plugin.Context, event model.OnInstallEvent) error {
	conf := p.getConfiguration()

	// Don't start wizard if OAuth is configured
	if conf.IsOAuthConfigured() {
		p.client.Log.Debug("OAuth is configured, skipping setup wizard",
			"ForgejoOAuthClientID", lastN(conf.ForgejoOAuthClientID, 4),
			"ForgejoOAuthClientSecret", lastN(conf.ForgejoOAuthClientSecret, 4))
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

func (p *Plugin) MessageWillBePosted(c *plugin.Context, post *model.Post) (*model.Post, string) {
	// If not enabled in config, ignore.
	config := p.getConfiguration()
	if config.EnableCodePreview == "disable" {
		return nil, ""
	}

	if post.UserId == "" {
		return nil, ""
	}

	shouldProcessMessage, err := p.client.Post.ShouldProcessMessage(post)
	if err != nil {
		p.client.Log.Warn("Error while checking if the message should be processed", "error", err.Error())
		return nil, ""
	}

	if !shouldProcessMessage {
		return nil, ""
	}

	msg := post.Message
	info, appErr := p.getGitHubUserInfo(post.UserId)
	if appErr != nil {
		if appErr.ID != apiErrorIDNotConnected {
			p.client.Log.Warn("Error in getting user info", "error", appErr.Message)
		}
		return nil, ""
	}
	// TODO: make this part of the Plugin struct and reuse it.
	ghClient := p.githubConnectUser(context.Background(), info)

	replacements := p.getReplacements(msg)
	post.Message = p.makeReplacements(msg, replacements, ghClient)
	return post, ""
}

func (p *Plugin) getOAuthConfig() (*oauth2.Config, error) {
	oauthConfig, err := getOauthConfig(p.getConfiguration())
	if err != nil {
		return nil, errors.Wrap(err, "failed to create OAuth config")
	}
	redirectURL, err := buildPluginURL(p.client, "oauth", "complete")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create PluginURL")
	}
	oauthConfig.RedirectURL = redirectURL
	return oauthConfig, nil
}

func getOauthConfig(config *Configuration) (*oauth2.Config, error) {
	baseURL := config.getBaseURL()
	if testOAuthServerURL != "" {
		baseURL = testOAuthServerURL + "/"
	}

	authURL, err := url.JoinPath(baseURL, "login", "oauth", "authorize")
	if err != nil {
		return nil, errors.Wrap(err, "failed to build AuthURL")
	}
	tokenURL, err := url.JoinPath(baseURL, "login", "oauth", "access_token")
	if err != nil {
		return nil, errors.Wrap(err, "failed to build TokenURL")
	}

	return &oauth2.Config{
		ClientID:     config.ForgejoOAuthClientID,
		ClientSecret: config.ForgejoOAuthClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:   authURL,
			TokenURL:  tokenURL,
			AuthStyle: oauth2.AuthStyleInHeader,
		},
	}, nil
}

type ForgejoUserInfo struct {
	UserID              string
	Token               *oauth2.Token
	ForgejoUsername     string
	LastToDoPostAt      int64
	Settings            *UserSettings
	AllowedPrivateRepos bool
}

type UserSettings struct {
	SidebarButtons                 string   `json:"sidebar_buttons"`
	DailyReminder                  bool     `json:"daily_reminder"`
	DailyReminderOnChange          bool     `json:"daily_reminder_on_change"`
	Notifications                  bool     `json:"notifications"`
	DisableTeamNotifications       bool     `json:"disable_team_review_notifications"`
	ExcludeTeamReviewNotifications []string `json:"exclude_team_review_notifications"`
}

func (p *Plugin) storeGitHubUserInfo(info *ForgejoUserInfo) error {
	config := p.getConfiguration()

	encryptedAccessToken, accessErr := encrypt([]byte(config.EncryptionKey), info.Token.AccessToken)
	if accessErr != nil {
		return errors.Wrap(accessErr, "error occurred while encrypting access token")
	}
	encryptedRefreshToken, refreshErr := encrypt([]byte(config.EncryptionKey), info.Token.RefreshToken)
	if refreshErr != nil {
		return errors.Wrap(refreshErr, "error occurred while encrypting refresh token")
	}

	info.Token.AccessToken = encryptedAccessToken
	info.Token.RefreshToken = encryptedRefreshToken

	if _, err := p.store.Set(info.UserID+forgejoTokenKey, info); err != nil {
		return errors.Wrap(err, "error occurred while trying to store user info into KV store")
	}

	return nil
}

func (p *Plugin) getGitHubUserInfo(userID string) (*ForgejoUserInfo, *APIErrorResponse) {
	config := p.getConfiguration()

	var userInfo *ForgejoUserInfo
	err := p.store.Get(userID+forgejoTokenKey, &userInfo)
	if err != nil {
		return nil, &APIErrorResponse{ID: "", Message: "Unable to get user info.", StatusCode: http.StatusInternalServerError}
	}
	if userInfo == nil {
		return nil, &APIErrorResponse{ID: apiErrorIDNotConnected, Message: "Must connect user account to Forgejo first.", StatusCode: http.StatusBadRequest}
	}

	unencryptedAccessToken, accessErr := decrypt([]byte(config.EncryptionKey), userInfo.Token.AccessToken)
	if accessErr != nil {
		p.client.Log.Warn("Failed to decrypt access token", "error", accessErr.Error())
		return nil, &APIErrorResponse{ID: "", Message: "Unable to decrypt access token.", StatusCode: http.StatusInternalServerError}
	}

	unencryptedRefreshToken, refreshErr := decrypt([]byte(config.EncryptionKey), userInfo.Token.RefreshToken)
	if refreshErr != nil {
		p.client.Log.Warn("Failed to decrypt refresh token", "error", refreshErr.Error())
		return nil, &APIErrorResponse{ID: "", Message: "Unable to decrypt refresh token.", StatusCode: http.StatusInternalServerError}
	}

	userInfo.Token.AccessToken = unencryptedAccessToken
	userInfo.Token.RefreshToken = unencryptedRefreshToken

	return userInfo, nil
}

func (p *Plugin) storeGitHubToUserIDMapping(githubUsername, userID string) error {
	_, err := p.store.Set(githubUsername+forgejoUsernameKey, []byte(userID))
	if err != nil {
		return errors.Wrap(err, "encountered error saving forgejo username mapping")
	}

	return nil
}

func (p *Plugin) getForgejoToUserIDMapping(forgejoUsername string) string {
	var data []byte
	err := p.store.Get(forgejoUsername+forgejoUsernameKey, &data)
	if err != nil {
		p.client.Log.Warn("Error occurred while getting the user ID from KV store using the Forgejo username", "error", err.Error())
		return ""
	}

	return string(data)
}

// getGitHubToUsernameMapping maps a GitHub username to the corresponding Mattermost username, if any.
func (p *Plugin) getGitHubToUsernameMapping(githubUsername string) string {
	user, _ := p.client.User.Get(p.getForgejoToUserIDMapping(githubUsername))
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

	if err := p.store.Delete(userID + forgejoTokenKey); err != nil {
		p.client.Log.Warn("Failed to delete forgejo token from KV store", "userID", userID, "error", err.Error())
	}

	if err := p.store.Delete(userInfo.ForgejoUsername + forgejoUsernameKey); err != nil {
		p.client.Log.Warn("Failed to delete forgejo token from KV store", "userID", userID, "error", err.Error())
	}

	user, err := p.client.User.Get(userID)
	if err != nil {
		p.client.Log.Warn("Failed to get user props", "userID", userID, "error", err.Error())
	} else {
		_, ok := user.Props["git_user"]
		if ok {
			delete(user.Props, "git_user")
			err := p.client.User.Update(user)
			if err != nil {
				p.client.Log.Warn("Failed to get update user props", "userID", userID, "error", err.Error())
			}
		}
	}

	p.client.Frontend.PublishWebSocketEvent(
		wsEventDisconnect,
		nil,
		&model.WebsocketBroadcast{UserId: userID},
	)
}

func (p *Plugin) openIssueCreateModal(userID string, channelID string, title string) {
	p.client.Frontend.PublishWebSocketEvent(
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
	channel, err := p.client.Channel.GetDirect(userID, p.BotUserID)
	if err != nil {
		p.client.Log.Warn("Couldn't get bot's DM channel", "userID", userID, "error", err.Error())
		return
	}

	post := &model.Post{
		UserId:    p.BotUserID,
		ChannelId: channel.Id,
		Message:   message,
		Type:      postType,
	}

	if err = p.client.Post.CreatePost(post); err != nil {
		p.client.Log.Warn("Failed to create DM post", "userID", userID, "post", post, "error", err.Error())
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
	_, err := p.store.Set(userID+dailySummary, []byte(summaryText))
	if err != nil {
		return err
	}

	return nil
}

func (p *Plugin) GetDailySummaryText(userID string) (string, error) {
	var summaryByte []byte
	err := p.store.Get(userID+dailySummary, &summaryByte)
	if err != nil {
		return "", err
	}

	return string(summaryByte), nil
}

func (p *Plugin) PostToDo(info *ForgejoUserInfo, userID string) error {
	text, err := p.GetToDo(info)
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

func (p *Plugin) GetToDo(info *ForgejoUserInfo) (string, error) {
	config := p.getConfiguration()
	orgList := config.getOrganizations()
	baseURL := config.getBaseURL()

	forgejoClient := p.forgejoConnect(info)

	var resultReview, resultAssignee, resultOpenPR []*github.Issue
	for _, org := range orgList {
		resultReviewData := makeForgejoRequest[[]FIssue](p, forgejoClient, p.createRequestURL(baseURL, org, "review_requested"))
		resultOpenPRData := makeForgejoRequest[[]FIssue](p, forgejoClient, p.createRequestURL(baseURL, org, "created"))
		resultAssigneeData := makeForgejoRequest[[]FIssue](p, forgejoClient, p.createRequestURL(baseURL, org, "assigned"))

		resultReview = fillGhIssue(resultReviewData, baseURL, resultReview)
		resultOpenPR = fillGhIssue(resultOpenPRData, baseURL, resultOpenPR)
		resultAssignee = fillGhIssue(resultAssigneeData, baseURL, resultAssignee)
	}
	notifications := makeForgejoRequest[[]FNotification](p, forgejoClient, fmt.Sprintf("%sapi/v1/notifications", baseURL))

	text := "##### Unread Messages\n"

	notificationCount := 0
	notificationContent := ""
	for _, n := range notifications {
		if n.Repository == nil {
			p.client.Log.Warn("Unable to get repository for notification in todo list. Skipping.")
			continue
		}

		if p.checkOrg(*n.Repository.Owner.Login) != nil {
			continue
		}

		notificationSubject := *n.Subject
		notificationType := *notificationSubject.Type
		switch notificationType {
		case "RepositoryVulnerabilityAlert":
			message := fmt.Sprintf("[Vulnerability Alert for %v](%v)", n.Repository.FullName, fixGithubNotificationSubjectURL(*n.Subject.URL, ""))
			notificationContent += fmt.Sprintf("* %v\n", message)
		default:
			issueURL := *n.Subject.URL
			issueNumIndex := strings.LastIndex(issueURL, "/")
			issueNum := issueURL[issueNumIndex+1:]
			subjectURL := *n.Subject.URL
			if *n.Subject.LatestCommentURL != "" {
				subjectURL = *n.Subject.LatestCommentURL
			}

			notificationTitle := *notificationSubject.Title
			notificationURL := fixGithubNotificationSubjectURL(subjectURL, issueNum)
			notificationContent += getToDoDisplayText(baseURL, notificationTitle, notificationURL, notificationType, n.Repository)
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

	if len(resultReview) == 0 {
		text += "You don't have any pull requests awaiting your review.\n"
	} else {
		text += fmt.Sprintf("You have %v pull requests awaiting your review:\n", len(resultReview))

		for _, pr := range resultReview {
			text += getToDoDisplayText(baseURL, pr.GetTitle(), pr.GetHTMLURL(), "", nil)
		}
	}

	text += "##### Your Open Pull Requests\n"

	if len(resultOpenPR) == 0 {
		text += "You don't have any open pull requests.\n"
	} else {
		text += fmt.Sprintf("You have %v open pull requests:\n", len(resultOpenPR))

		for _, pr := range resultOpenPR {
			text += getToDoDisplayText(baseURL, pr.GetTitle(), pr.GetHTMLURL(), "", nil)
		}
	}

	text += "##### Your Assignments\n"

	if len(resultAssignee) == 0 {
		text += "You don't have any assignments.\n"
	} else {
		text += fmt.Sprintf("You have %v assignments:\n", len(resultAssignee))

		for _, assign := range resultAssignee {
			text += getToDoDisplayText(baseURL, assign.GetTitle(), assign.GetHTMLURL(), "", nil)
		}
	}

	return text, nil
}

func makeForgejoRequest[T any](p *Plugin, forgejoClient *http.Client, requestURL string) T {
	var result T
	response, err := forgejoClient.Get(requestURL)
	if err != nil {
		p.client.Log.Error("Failed Forgejo request", "url", requestURL, "error", err.Error())
		return result
	}
	defer response.Body.Close()

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		p.client.Log.Error("Error decoding Plugin JSON body", err.Error())
	}
	return result
}

func (p *Plugin) HasUnreads(info *ForgejoUserInfo) bool {
	config := p.getConfiguration()
	orgList := config.getOrganizations()
	baseURL := config.getBaseURL()

	forgejoClient := p.forgejoConnect(info)

	var resultReview, resultAssignee, resultOpenPR []*github.Issue
	for _, org := range orgList {
		resultReviewData := makeForgejoRequest[[]FIssue](p, forgejoClient, p.createRequestURL(baseURL, org, "review_requested"))
		resultOpenPRData := makeForgejoRequest[[]FIssue](p, forgejoClient, p.createRequestURL(baseURL, org, "created"))
		resultAssigneeData := makeForgejoRequest[[]FIssue](p, forgejoClient, p.createRequestURL(baseURL, org, "assigned"))

		resultReview = fillGhIssue(resultReviewData, baseURL, resultReview)
		resultOpenPR = fillGhIssue(resultOpenPRData, baseURL, resultOpenPR)
		resultAssignee = fillGhIssue(resultAssigneeData, baseURL, resultAssignee)
	}

	relevantNotifications := false
	notifications := makeForgejoRequest[[]FNotification](p, forgejoClient, fmt.Sprintf("%sapi/v1/notifications", baseURL))

	for _, n := range notifications {
		if n.Repository == nil {
			p.client.Log.Warn("Unable to get repository for notification in todo list. Skipping.")
			continue
		}

		if p.checkOrg(*n.Repository.Owner.Login) != nil {
			continue
		}

		relevantNotifications = true
		break
	}

	if len(resultReview) == 0 && !relevantNotifications && len(resultOpenPR) == 0 && len(resultAssignee) == 0 {
		return false
	}

	return true
}

func (p *Plugin) checkOrg(org string) error {
	config := p.getConfiguration()

	orgList := config.getOrganizations()
	if len(orgList) == 0 {
		return nil
	}

	for _, configOrg := range orgList {
		if configOrg == strings.ToLower(org) {
			return nil
		}
	}

	return errors.Errorf("only repositories in the %v organization(s) are supported", config.ForgejoOrg)
}

func (p *Plugin) isUserOrganizationMember(githubClient *github.Client, username string, organization string) bool {
	if organization == "" {
		return false
	}

	isMember, _, err := githubClient.Organizations.IsMember(context.Background(), organization, username)
	if err != nil {
		p.client.Log.Warn("Failed to check if user is org member", "Forgejo username", username, "error", err.Error())
		return false
	}

	return isMember
}

func (p *Plugin) isOrganizationLocked() bool {
	config := p.getConfiguration()
	configOrg := strings.TrimSpace(config.ForgejoOrg)

	return configOrg != ""
}

func (p *Plugin) sendRefreshEvent(userID string) {
	eventLogger := logger.New(p.API).With(logger.LogContext{
		"userid": userID,
	})

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)

	context := &Context{
		Ctx:    ctx,
		UserID: userID,
		Log:    eventLogger,
	}

	defer cancel()

	info, apiErr := p.getGitHubUserInfo(context.UserID)
	if apiErr != nil {
		p.client.Log.Warn("Failed to get forgejo user info", "error", apiErr.Error())
		return
	}

	userContext := &UserContext{
		Context: *context,
		GHInfo:  info,
	}

	sidebarContent, err := p.getSidebarData(userContext)
	if err != nil {
		p.client.Log.Warn("Failed to get the sidebar data", "error", err.Error())
		return
	}

	contentMap, err := sidebarContent.toMap()
	if err != nil {
		p.client.Log.Warn("Failed to convert sidebar content to map", "error", err.Error())
		return
	}

	p.client.Frontend.PublishWebSocketEvent(
		wsEventRefresh,
		contentMap,
		&model.WebsocketBroadcast{UserId: userID},
	)
}

func (s *SidebarContent) toMap() (map[string]interface{}, error) {
	var m map[string]interface{}
	bytes, err := json.Marshal(&s)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(bytes, &m); err != nil {
		return nil, err
	}

	return m, nil
}

// getUsername returns the Forgejo username for a given Mattermost user,
// if the user is connected to Forgejo via this plugin.
// Otherwise it return the Mattermost username. It will be escaped via backticks.
func (p *Plugin) getUsername(mmUserID string) (string, error) {
	info, apiEr := p.getGitHubUserInfo(mmUserID)
	if apiEr != nil {
		if apiEr.ID != apiErrorIDNotConnected {
			return "", apiEr
		}

		user, appEr := p.client.User.Get(mmUserID)
		if appEr != nil {
			return "", appEr
		}

		return fmt.Sprintf("`@%s`", user.Username), nil
	}

	return "@" + info.ForgejoUsername, nil
}
