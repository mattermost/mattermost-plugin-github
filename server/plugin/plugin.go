package plugin

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/google/go-github/v41/github"
	"github.com/gorilla/mux"
	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-plugin-api/experimental/bot/poster"
	"github.com/mattermost/mattermost-plugin-github/server/api"
	"github.com/mattermost/mattermost-plugin-github/server/app"
	"github.com/mattermost/mattermost-plugin-github/server/command"
	"github.com/mattermost/mattermost-plugin-github/server/config"
	"github.com/mattermost/mattermost-plugin-github/server/telemetry"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/pkg/errors"

	root "github.com/mattermost/mattermost-plugin-github"
)

const (
	githubPrivateRepoKey = "_githubprivate"

	mm34646MutexKey = "mm34646_token_reset_mutex"
	mm34646DoneKey  = "mm34646_token_reset_done"

	wsEventDisconnect = "disconnect"

	wsEventRefresh = "refresh"

	WSEventRefresh = "refresh"
)

var (
	Manifest model.Manifest = root.Manifest
)

type CommandHandleFunc func(c *plugin.Context, args *model.CommandArgs, parameters []string, userInfo *app.GitHubUserInfo) string
type Plugin struct {
	plugin.MattermostPlugin
	client *pluginapi.Client

	app app.App

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	config config.Service

	handler *api.Handler

	router *mux.Router

	tracker telemetry.Tracker

	BotUserID   string
	poster      poster.Poster
	FlowManager *FlowManager

	CommandHandlers map[string]CommandHandleFunc

	// githubPermalinkRegex is used to parse github permalinks in post messages.
	githubPermalinkRegex *regexp.Regexp
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

func (p *Plugin) GetGitHubClient(ctx context.Context, userID string) (*github.Client, error) {
	userInfo, apiErr := p.GetGitHubUserInfo(userID)
	if apiErr != nil {
		return nil, apiErr
	}

	return p.GithubConnectUser(ctx, userInfo), nil
}

func (p *Plugin) setDefaultConfiguration() error {
	config := p.config.GetConfiguration()

	changed, err := config.SetDefaults(pluginapi.IsCloud(p.client.System.GetLicense()))
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
	if p.client == nil {
		p.client = pluginapi.NewClient(p.API, p.Driver)
	}

	siteURL := p.client.Configuration.GetConfig().ServiceSettings.SiteURL
	if siteURL == nil || *siteURL == "" {
		return errors.New("siteURL is not set. Please set it and restart the plugin")
	}

	err := p.setDefaultConfiguration()
	if err != nil {
		return errors.Wrap(err, "failed to set default configuration")
	}

	p.config = config.NewConfigService(p.client, p.config.GetManifest())
	pluginAPIClient := pluginapi.NewClient(p.API, p.Driver)

	if err = command.RegisterCommands(p.API.RegisterCommand, p.config.GetConfiguration()); err != nil {
		return errors.Wrapf(err, "failed register commands")
	}

	if p.config.GetConfiguration().UsePreregisteredApplication && p.chimeraURL == "" {
		return errors.New("cannot use pre-registered application if Chimera URL is not set or empty. " +
			"For now using pre-registered application is intended for Cloud instances only. " +
			"If you are running on-prem disable the setting and use a custom application, otherwise set PluginSettings.ChimeraOAuthProxyURL")
	}

	p.handler = api.NewHandler(pluginAPIClient, p.config)

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

	//TODO: Initialize p.app
	p.BotUserID = botID

	p.poster = poster.NewPoster(&p.client.Post, p.BotUserID)
	p.FlowManager = p.NewFlowManager()

	registerGitHubToUsernameMappingCallback(p.app.GetGitHubToUsernameMapping)

	go func() {
		resetErr := p.forceResetAllMM34646()
		if resetErr != nil {
			p.client.Log.Debug("failed to reset user tokens", "error", resetErr.Error())
		}
	}()
	return nil
}

func (p *Plugin) OnDeactivate() error {
	p.app.WebhookBroker.Close()
	p.app.OauthBroker.Close()
	if err := p.telemetryClient.Close(); err != nil {
		p.API.LogWarn("Telemetry client failed to close", "error", err.Error())
	}
	return nil
}

func (p *Plugin) OnInstall(c *plugin.Context, event model.OnInstallEvent) error {
	// Don't start wizard if OAuth is configured
	if p.config.GetConfiguration().IsOAuthConfigured() {
		return nil
	}

	return p.FlowManager.StartSetupWizard(event.UserId, "")
}

func (p *Plugin) OnSendDailyTelemetry() {
	p.SendDailyTelemetry()
}

func (p *Plugin) OnPluginClusterEvent(c *plugin.Context, ev model.PluginClusterEvent) {
	p.HandleClusterEvent(ev)
}

func (p *Plugin) MessageWillBePosted(c *plugin.Context, post *model.Post) (*model.Post, string) {
	// If not enabled in config, ignore.
	config := p.config.GetConfiguration()
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
	info, appErr := p.GetGitHubUserInfo(post.UserId)
	if appErr != nil {
		if appErr.ID != api.ApiErrorIDNotConnected {
			p.client.Log.Warn("Error in getting user info", "error", appErr.Message)
		}
		return nil, ""
	}
	// TODO: make this part of the Plugin struct and reuse it.
	ghClient := p.GithubConnectUser(context.Background(), info)

	replacements := p.getReplacements(msg)
	post.Message = p.makeReplacements(msg, replacements, ghClient)
	return post, ""
}

func (p *Plugin) StoreGitHubUserInfo(info *app.GitHubUserInfo) error {
	config := p.config.GetConfiguration()

	encryptedToken, err := encrypt([]byte(config.EncryptionKey), info.Token.AccessToken)
	if err != nil {
		return errors.Wrap(err, "error occurred while encrypting access token")
	}

	info.Token.AccessToken = encryptedToken

	if _, err := p.client.KV.Set(info.UserID+app.GithubTokenKey, info); err != nil {
		return errors.Wrap(err, "error occurred while trying to store user info into KV store")
	}

	return nil
}

func (p *Plugin) StoreGitHubToUserIDMapping(githubUsername, userID string) error {
	_, err := p.client.KV.Set(githubUsername+app.GithubUsernameKey, []byte(userID))
	if err != nil {
		return errors.Wrap(err, "encountered error saving github username mapping")
	}

	return nil
}

func (p *Plugin) DisconnectGitHubAccount(userID string) {
	userInfo, _ := p.app.GetGitHubUserInfo(userID)
	if userInfo == nil {
		return
	}

	if err := p.client.KV.Delete(userID + app.GithubTokenKey); err != nil {
		p.client.Log.Warn("Failed to delete github token from KV store", "userID", userID, "error", err.Error())
	}

	if err := p.client.KV.Delete(userInfo.GitHubUsername + app.GithubUsernameKey); err != nil {
		p.client.Log.Warn("Failed to delete github token from KV store", "userID", userID, "error", err.Error())
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

func (p *Plugin) sendRefreshEvent(userID string) {
	p.client.Frontend.PublishWebSocketEvent(
		wsEventRefresh,
		nil,
		&model.WebsocketBroadcast{UserId: userID},
	)
}

// getUsername returns the GitHub username for a given Mattermost user,
// if the user is connected to GitHub via this plugin.
// Otherwise it return the Mattermost username. It will be escaped via backticks.
func (p *Plugin) getUsername(mmUserID string) (string, error) {
	info, apiEr := p.GetGitHubUserInfo(mmUserID)
	if apiEr != nil {
		if apiEr.ID != api.ApiErrorIDNotConnected {
			return "", apiEr
		}

		user, appEr := p.client.User.Get(mmUserID)
		if appEr != nil {
			return "", appEr
		}

		return fmt.Sprintf("`@%s`", user.Username), nil
	}

	return "@" + info.GitHubUsername, nil
}
