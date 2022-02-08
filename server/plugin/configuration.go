package plugin

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/mattermost/mattermost-plugin-api/experimental/telemetry"
	"github.com/pkg/errors"
)

// configuration captures the plugin's external configuration as exposed in the Mattermost server
// configuration, as well as values computed from the configuration. Any public fields will be
// deserialized from the Mattermost server configuration in OnConfigurationChange.
//
// As plugins are inherently concurrent (hooks being called asynchronously), and the plugin
// configuration can change at any time, access to the configuration must be synchronized. The
// strategy used in this plugin is to guard a pointer to the configuration, and clone the entire
// struct whenever it changes. You may replace this with whatever strategy you choose.
//
// If you add non-reference types to your configuration struct, be sure to rewrite Clone as a deep
// copy appropriate for your types.
type Configuration struct {
	GitHubOrg                   string
	GitHubOAuthClientID         string
	GitHubOAuthClientSecret     string
	WebhookSecret               string
	EnableLeftSidebar           bool
	EnablePrivateRepo           bool
	ConnectToPrivateByDefault   bool
	EncryptionKey               string
	EnterpriseBaseURL           string
	EnterpriseUploadURL         string
	EnableCodePreview           string
	EnableWebhookEventLogging   bool
	UsePreregisteredApplication bool
}

func (c *Configuration) ToMap() (map[string]interface{}, error) {
	var out map[string]interface{}
	data, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (c *Configuration) setDefaults(isCloud bool) (bool, error) {
	changed := false

	if c.EncryptionKey == "" {
		secret, err := generateSecret()
		if err != nil {
			return false, err
		}

		c.EncryptionKey = secret
		changed = true
	}

	if c.WebhookSecret == "" {
		secret, err := generateSecret()
		if err != nil {
			return false, err
		}

		c.WebhookSecret = secret
		changed = true
	}

	if isCloud && !c.UsePreregisteredApplication && !c.IsOAuthConfigured() {
		c.UsePreregisteredApplication = true
		changed = true
	}

	return changed, nil
}

func (c *Configuration) getBaseURL() string {
	if c.EnterpriseBaseURL != "" {
		return c.EnterpriseBaseURL + "/"
	}

	return "https://github.com/"
}

func (c *Configuration) sanitize() {
	// Ensure EnterpriseBaseURL and EnterpriseUploadURL end with a slash
	c.EnterpriseBaseURL = strings.TrimRight(c.EnterpriseBaseURL, "/")
	c.EnterpriseUploadURL = strings.TrimRight(c.EnterpriseUploadURL, "/")

	// Trim spaces around org and OAuth credentials
	c.GitHubOrg = strings.TrimSpace(c.GitHubOrg)
	c.GitHubOAuthClientID = strings.TrimSpace(c.GitHubOAuthClientID)
	c.GitHubOAuthClientSecret = strings.TrimSpace(c.GitHubOAuthClientSecret)
}

func (c *Configuration) IsOAuthConfigured() bool {
	return (c.GitHubOAuthClientID != "" && c.GitHubOAuthClientSecret != "") ||
		c.UsePreregisteredApplication
}

// IsSASS return if SASS GitHub at https://github.com is used
func (c *Configuration) IsSASS() bool {
	return c.EnterpriseBaseURL == "" && c.EnterpriseUploadURL == ""
}

// Clone shallow copies the configuration. Your implementation may require a deep copy if
// your configuration has reference types.
func (c *Configuration) Clone() *Configuration {
	var clone = *c
	return &clone
}

// IsValid checks if all needed fields are set.
func (c *Configuration) IsValid() error {
	if !c.UsePreregisteredApplication {
		if c.GitHubOAuthClientID == "" {
			return errors.New("must have a github oauth client id")
		}
		if c.GitHubOAuthClientSecret == "" {
			return errors.New("must have a github oauth client secret")
		}
	}

	if c.UsePreregisteredApplication && c.EnterpriseBaseURL != "" {
		return errors.New("cannot use pre-registered application with GitHub enterprise")
	}

	if c.EncryptionKey == "" {
		return errors.New("must have an encryption key")
	}

	return nil
}

// getConfiguration retrieves the active configuration under lock, making it safe to use
// concurrently. The active configuration may change underneath the client of this method, but
// the struct returned by this API call is considered immutable.
func (p *Plugin) getConfiguration() *Configuration {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()

	if p.configuration == nil {
		return &Configuration{}
	}

	return p.configuration
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
func (p *Plugin) setConfiguration(configuration *Configuration) {
	p.configurationLock.Lock()
	defer p.configurationLock.Unlock()

	if configuration != nil && p.configuration == configuration {
		// Ignore assignment if the configuration struct is empty. Go will optimize the
		// allocation for same to point at the same memory address, breaking the check
		// above.
		if reflect.ValueOf(*configuration).NumField() == 0 {
			return
		}

		panic("setConfiguration called with the existing configuration")
	}

	p.configuration = configuration
}

// OnConfigurationChange is invoked when configuration changes may have been made.
func (p *Plugin) OnConfigurationChange() error {
	var configuration = new(Configuration)

	// Load the public configuration fields from the Mattermost server configuration.
	if err := p.API.LoadPluginConfiguration(configuration); err != nil {
		return errors.Wrap(err, "failed to load plugin configuration")
	}

	configuration.sanitize()

	p.setConfiguration(configuration)

	command, err := p.getCommand(configuration)
	if err != nil {
		return errors.Wrap(err, "failed to get command")
	}

	err = p.API.RegisterCommand(command)
	if err != nil {
		return errors.Wrap(err, "failed to register command")
	}

	enableDiagnostics := false
	if config := p.API.GetConfig(); config != nil {
		if configValue := config.LogSettings.EnableDiagnostics; configValue != nil {
			enableDiagnostics = *configValue
		}
	}

	p.tracker = telemetry.NewTracker(p.telemetryClient, p.API.GetDiagnosticId(), p.API.GetServerVersion(), Manifest.Id, Manifest.Version, "github", enableDiagnostics)

	return nil
}

func generateSecret() (string, error) {
	b := make([]byte, 256)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	s := base64.RawStdEncoding.EncodeToString(b)

	s = s[:32]

	return s, nil
}
