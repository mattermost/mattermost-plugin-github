package config

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
)

const (
	chimeraGitHubAppIdentifier = "plugin-github"
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
	GitHubOrg                   string `json:"githuborg"`
	GitHubOAuthClientID         string `json:"githuboauthclientid"`
	GitHubOAuthClientSecret     string `json:"githuboauthclientsecret"`
	WebhookSecret               string `json:"webhooksecret"`
	EnableLeftSidebar           bool   `json:"enableleftsidebar"`
	EnablePrivateRepo           bool   `json:"enableprivaterepo"`
	ConnectToPrivateByDefault   bool   `json:"connecttoprivatebydefault"`
	EncryptionKey               string `json:"encryptionkey"`
	EnterpriseBaseURL           string `json:"enterprisebaseurl"`
	EnterpriseUploadURL         string `json:"enterpriseuploadurl"`
	EnableCodePreview           string `json:"enablecodepreview"`
	EnableWebhookEventLogging   bool   `json:"enablewebhookeventlogging"`
	UsePreregisteredApplication bool   `json:"usepreregisteredapplication"`
	BotUserID                   string `json:"bot_user_id"`
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

func (c *Configuration) SetDefaults(isCloud bool) (bool, error) {
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

func (c *Configuration) GetBaseURL() string {
	if c.EnterpriseBaseURL != "" {
		return c.EnterpriseBaseURL + "/"
	}

	return "https://github.com/"
}

func (c *Configuration) Sanitize() {
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

// IsSASS return if SASS GitHub at https://github.com is used.
func (c *Configuration) IsSASS() bool {
	return c.EnterpriseBaseURL == "" && c.EnterpriseUploadURL == ""
}

func (c *Configuration) ClientConfiguration() map[string]interface{} {
	return map[string]interface{}{
		"left_sidebar_enabled": c.EnableLeftSidebar,
	}
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
