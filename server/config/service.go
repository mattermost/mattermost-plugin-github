package config

import (
	"fmt"
	"net/url"
	"path"
	"reflect"
	"sync"

	"github.com/google/go-github/github"
	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-plugin-api/experimental/telemetry"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

type Service interface {
	// GetConfiguration retrieves the active configuration under lock, making it safe to use
	// concurrently. The active configuration may change underneath the client of this method, but
	// the struct returned by this API call is considered immutable.
	GetConfiguration() *Configuration

	// UpdateConfiguration updates the config. Any parts of the config that are persisted in the plugin's
	// section in the server's config will be saved to the server.
	UpdateConfiguration(f func(*Configuration)) error

	// RegisterConfigChangeListener registers a function that will called when the config might have
	// been changed. Returns an id which can be used to unregister the listener.
	RegisterConfigChangeListener(listener func()) string

	// UnregisterConfigChangeListener unregisters the listener function identified by id.
	UnregisterConfigChangeListener(id string)

	// GetManifest gets the plugin manifest.
	GetManifest() *model.Manifest

	// Gets the OAuth Configuration
	GetOAuthConfig(bool) *oauth2.Config
}

// NewConfigService Creates a new ServiceImpl struct.
func NewConfigService(api *pluginapi.Client, manifest *model.Manifest) *Config {
	c := &Config{
		manifest: manifest,
	}
	c.client = api
	c.configuration = new(Configuration)
	c.configChangeListeners = make(map[string]func())
	c.registerChimeraURL()

	// api.LoadPluginConfiguration never returns an error, so ignore it.
	_ = api.Configuration.LoadPluginConfiguration(c.configuration)

	return c
}

type Config struct {
	api    plugin.API
	driver plugin.Driver

	client *pluginapi.Client

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *Configuration

	chimeraURL string

	// configChangeListeners will be notified when the OnConfigurationChange event has been called.
	configChangeListeners map[string]func()

	// manifest is the plugin manifest
	manifest *model.Manifest

	tracker telemetry.Tracker
}

// testOAuthServerURL is the URL for the oauthServer used for testing purposes
// It should be set through ldflags when compiling for E2E, and keep it blank otherwise
var testOAuthServerURL = ""

// WSEventConfigUpdate is the WebSocket event to update the configurations on webapp.
const WSEventConfigUpdate = "config_update"

func (c *Config) InitializeService(api plugin.API) {
	c.api = api
	c.configuration = new(Configuration)
	c.configChangeListeners = make(map[string]func())

	// api.LoadPluginConfiguration never returns an error, so ignore it.
	_ = api.LoadPluginConfiguration(c.configuration)
}

// SetConfigApi is only called by the plugin during OnActivate. After that it shouldn't be called.
func (c *Config) SetConfigApi(api plugin.API, cfg *Configuration) {
	c.api = api
	c.configuration = cfg
}

// GetConfiguration retrieves the active configuration under lock, making it safe to use
// concurrently. The active configuration may change underneath the client of this method, but
// the struct returned by this API call is considered immutable.
func (c *Config) GetConfiguration() *Configuration {
	c.configurationLock.RLock()
	defer c.configurationLock.RUnlock()

	if c.configuration == nil {
		return &Configuration{}
	}

	return c.configuration
}

// UpdateConfiguration updates the config and saves it on the server
func (c *Config) UpdateConfiguration(f func(*Configuration)) error {
	c.configurationLock.Lock()

	if c.configuration == nil {
		c.configuration = &Configuration{}
	}

	oldStorableConfig, _ := c.configuration.ToMap()
	f(c.configuration)
	newStorableConfig, _ := c.configuration.ToMap()

	// Don't hold the lock longer than necessary, especially since we're calling the api and then listeners.
	c.configurationLock.Unlock()

	if !reflect.DeepEqual(oldStorableConfig, newStorableConfig) {
		if appErr := c.api.SavePluginConfig(newStorableConfig); appErr != nil {
			return errors.New(appErr.Error())
		}
	}

	for _, f := range c.configChangeListeners {
		f()
	}

	return nil
}

func (c *Config) RegisterConfigChangeListener(listener func()) string {
	if c.configChangeListeners == nil {
		c.configChangeListeners = make(map[string]func())
	}

	id := model.NewId()
	c.configChangeListeners[id] = listener
	return id
}

func (c *Config) UnregisterConfigChangeListener(id string) {
	delete(c.configChangeListeners, id)
}

// OnConfigurationChange is invoked when configuration changes may have been made.
// This method satisfies the interface expected by the server. Embed config.Config in the plugin.
func (c *Config) OnConfigurationChange() error {
	// Have we been setup by OnActivate?
	if c.api == nil {
		return nil
	}

	if c.client == nil {
		c.client = pluginapi.NewClient(c.api, c.driver)
	}

	var configuration = new(Configuration)

	// Load the public configuration fields from the Mattermost server configuration.
	if err := c.api.LoadPluginConfiguration(configuration); err != nil {
		return errors.Wrap(err, "failed to load plugin configuration")
	}

	configuration.Sanitize()

	c.sendWebsocketEventIfNeeded(c.GetConfiguration(), configuration)

	c.setConfiguration(configuration)

	/* TODO: To migrate
	command, err := p.getCommand(configuration)
	if err != nil {
		return errors.Wrap(err, "failed to get command")
	}

	err = p.client.SlashCommand.Register(command)
	if err != nil {
		return errors.Wrap(err, "failed to register command")
	}
	*/

	// Some config changes require reloading tracking config
	if c.tracker != nil {
		c.tracker.ReloadConfig(telemetry.NewTrackerConfig(c.client.Configuration.GetConfig()))
	}

	for _, f := range c.configChangeListeners {
		f()
	}

	return nil
}

// setConfiguration replaces the active configuration under lock.
//
// Do not call setConfiguration while holding the configurationLock, as sync.Mutex is not
// reentrant. In particular, avoid using the plugin API entirely, as this may in turn trigger a
// hook back into the plugin. If that hook attempts to acquire this lock, a deadlock may occur.
//
// This method panics if setConfiguration is called with the existing configuration. This almost
// certainly means that the configuration was modified without being cloned and may result in
// an unsafe access.
func (c *Config) setConfiguration(configuration *Configuration) {
	c.configurationLock.Lock()
	defer c.configurationLock.Unlock()

	if configuration != nil && c.configuration == configuration {
		// Ignore assignment if the configuration struct is empty. Go will optimize the
		// allocation for same to point at the same memory address, breaking the check
		// above.
		if reflect.ValueOf(*configuration).NumField() == 0 {
			return
		}

		panic("setConfiguration called with the existing configuration")
	}

	c.configuration = configuration
}

func (c *Config) sendWebsocketEventIfNeeded(oldConfig, newConfig *Configuration) {
	// If the plugin just started, oldConfig is the zero value.
	// Hence, an unnecessary websocket event is sent.
	// Given that oldConfig is never nil, that case is hard to catch.
	if !reflect.DeepEqual(oldConfig.ClientConfiguration(), newConfig.ClientConfiguration()) {
		c.client.Frontend.PublishWebSocketEvent(
			WSEventConfigUpdate,
			newConfig.ClientConfiguration(),
			&model.WebsocketBroadcast{},
		)
	}
}

// GetManifest gets the plugin manifest.
func (c *Config) GetManifest() *model.Manifest {
	return c.manifest
}

func (c *Config) GetOAuthConfig(privateAllowed bool) *oauth2.Config {
	repo := github.ScopePublicRepo
	config := c.GetConfiguration()
	if config.EnablePrivateRepo && privateAllowed {
		// means that asks scope for private repositories
		repo = github.ScopeRepo
	}
	scopes := []string{string(repo), string(github.ScopeNotifications), string(github.ScopeReadOrg), string(github.ScopeAdminOrgHook)}

	if config.UsePreregisteredApplication {
		c.client.Log.Debug("Using Chimera Proxy OAuth configuration")
		return c.getOAuthConfigForChimeraApp(scopes)
	}

	baseURL := config.GetBaseURL()
	if testOAuthServerURL != "" {
		baseURL = testOAuthServerURL + "/"
	}

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

func (c *Config) getOAuthConfigForChimeraApp(scopes []string) *oauth2.Config {
	baseURL := fmt.Sprintf("%s/v1/github/%s", c.chimeraURL, chimeraGitHubAppIdentifier)
	authURL, _ := url.Parse(baseURL)
	tokenURL, _ := url.Parse(baseURL)

	authURL.Path = path.Join(authURL.Path, "oauth", "authorize")
	tokenURL.Path = path.Join(tokenURL.Path, "oauth", "token")

	redirectURL, _ := url.Parse(fmt.Sprintf("%s/plugins/github/oauth/complete", *c.api.GetConfig().ServiceSettings.SiteURL))

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

// registerChimeraURL fetches the Chimera URL from server settings or env var and sets it in the plugin object.
func (c *Config) registerChimeraURL() {
	chimeraURLSetting := c.api.GetConfig().PluginSettings.ChimeraOAuthProxyURL
	if chimeraURLSetting != nil {
		c.chimeraURL = *chimeraURLSetting
	}
}
