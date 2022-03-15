package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsValid(t *testing.T) {
	for _, testCase := range []struct {
		description string
		config      *Configuration
		errMsg      string
	}{
		{
			description: "valid configuration: pre-registered app",
			config: &Configuration{
				EncryptionKey:               "abcd",
				UsePreregisteredApplication: true,
			},
		},
		{
			description: "valid configuration: custom OAuth app",
			config: &Configuration{
				GitHubOAuthClientID:         "client-id",
				GitHubOAuthClientSecret:     "client-secret",
				EncryptionKey:               "abcd",
				UsePreregisteredApplication: false,
			},
		},
		{
			description: "invalid configuration: custom OAuth app without credentials",
			config: &Configuration{
				EncryptionKey:               "abcd",
				UsePreregisteredApplication: false,
			},
			errMsg: "must have a github oauth client id",
		},
		{
			description: "invalid configuration: GitHub Enterprise URL with pre-registered app",
			config: &Configuration{
				EnterpriseBaseURL:           "https://my-company.github.com",
				UsePreregisteredApplication: true,
				EncryptionKey:               "abcd",
			},
			errMsg: "cannot use pre-registered application with GitHub enterprise",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			err := testCase.config.IsValid()
			if testCase.errMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSetDefaults(t *testing.T) {
	for _, testCase := range []struct {
		description string
		isCloud     bool
		config      Configuration

		shouldChange bool
		outputCheck  func(*testing.T, *Configuration)
		errMsg       string
	}{
		{
			description: "noop",
			config: Configuration{
				EncryptionKey: "abcd",
				WebhookSecret: "efgh",
			},
			shouldChange: false,
			outputCheck: func(t *testing.T, c *Configuration) {
				assert.Equal(t, "abcd", c.EncryptionKey)
				assert.Equal(t, "efgh", c.WebhookSecret)
			},
		}, {
			description: "set encryption key",
			config: Configuration{
				EncryptionKey: "",
			},
			shouldChange: true,
			outputCheck: func(t *testing.T, c *Configuration) {
				assert.Len(t, c.EncryptionKey, 32)
			},
		}, {
			description: "set webhook key",
			config: Configuration{
				WebhookSecret: "",
			},
			shouldChange: true,
			outputCheck: func(t *testing.T, c *Configuration) {
				assert.Len(t, c.WebhookSecret, 32)
			},
		}, {
			description: "set webhook and encryption key",
			config: Configuration{
				EncryptionKey: "",
				WebhookSecret: "",
			},
			shouldChange: true,
			outputCheck: func(t *testing.T, c *Configuration) {
				assert.Len(t, c.EncryptionKey, 32)
				assert.Len(t, c.WebhookSecret, 32)
			},
		}, {
			description: "Should not set UsePreregisteredApplication in on-prem",
			isCloud:     false,
			config: Configuration{
				EncryptionKey:               "abcd",
				WebhookSecret:               "efgh",
				UsePreregisteredApplication: false,
			},
			shouldChange: false,
			outputCheck: func(t *testing.T, c *Configuration) {
				assert.Equal(t, "abcd", c.EncryptionKey)
				assert.Equal(t, "efgh", c.WebhookSecret)
			},
		}, {
			description: "Should set UsePreregisteredApplication in cloud if no OAuth secret is configured",
			isCloud:     true,
			config: Configuration{
				EncryptionKey:               "abcd",
				WebhookSecret:               "efgh",
				UsePreregisteredApplication: false,
			},
			shouldChange: true,
			outputCheck: func(t *testing.T, c *Configuration) {
				assert.Equal(t, "abcd", c.EncryptionKey)
				assert.Equal(t, "efgh", c.WebhookSecret)

				assert.True(t, c.UsePreregisteredApplication)
			},
		}, {
			description: "Should set not UsePreregisteredApplication in cloud if OAuth secret is configured",
			isCloud:     true,
			config: Configuration{
				EncryptionKey:               "abcd",
				WebhookSecret:               "efgh",
				UsePreregisteredApplication: false,
				GitHubOAuthClientID:         "some id",
				GitHubOAuthClientSecret:     "some secret",
			},
			shouldChange: false,
			outputCheck: func(t *testing.T, c *Configuration) {
				assert.Equal(t, "abcd", c.EncryptionKey)
				assert.Equal(t, "efgh", c.WebhookSecret)

				assert.False(t, c.UsePreregisteredApplication)
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			changed, err := testCase.config.setDefaults(testCase.isCloud)

			assert.Equal(t, testCase.shouldChange, changed)
			testCase.outputCheck(t, &testCase.config)

			if testCase.errMsg != "" {
				require.Error(t, err)
				assert.Equal(t, testCase.errMsg, err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
