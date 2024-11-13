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
				EncryptionKey: "abcd",
			},
		},
		{
			description: "valid configuration: custom OAuth app",
			config: &Configuration{
				ForgejoOAuthClientID:     "client-id",
				ForgejoOAuthClientSecret: "client-secret",
				EncryptionKey:            "abcd",
			},
		},
		{
			description: "invalid configuration: custom OAuth app without credentials",
			config: &Configuration{
				EncryptionKey: "abcd",
			},
			errMsg: "must have a forgejo oauth client id",
		},
		{
			description: "invalid configuration: Forgejo URL with pre-registered app",
			config: &Configuration{
				BaseURL:       "https://my-company.forgejo.com",
				EncryptionKey: "abcd",
			},
			errMsg: "cannot use pre-registered application with Forgejo",
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
		config      *Configuration

		shouldChange bool
		outputCheck  func(*testing.T, *Configuration)
		errMsg       string
	}{
		{
			description: "noop",
			config: &Configuration{
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
			config: &Configuration{
				EncryptionKey: "",
			},
			shouldChange: true,
			outputCheck: func(t *testing.T, c *Configuration) {
				assert.Len(t, c.EncryptionKey, 32)
			},
		}, {
			description: "set webhook key",
			config: &Configuration{
				WebhookSecret: "",
			},
			shouldChange: true,
			outputCheck: func(t *testing.T, c *Configuration) {
				assert.Len(t, c.WebhookSecret, 32)
			},
		}, {
			description: "set webhook and encryption key",
			config: &Configuration{
				EncryptionKey: "",
				WebhookSecret: "",
			},
			shouldChange: true,
			outputCheck: func(t *testing.T, c *Configuration) {
				assert.Len(t, c.EncryptionKey, 32)
				assert.Len(t, c.WebhookSecret, 32)
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			changed, err := testCase.config.setDefaults(testCase.isCloud)

			assert.Equal(t, testCase.shouldChange, changed)
			testCase.outputCheck(t, testCase.config)

			if testCase.errMsg != "" {
				require.Error(t, err)
				assert.Equal(t, testCase.errMsg, err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetOrganizations(t *testing.T) {
	tcs := []struct {
		Organizations   string
		ExpectedOrgList []string
	}{
		{
			Organizations:   "org-1,org-2",
			ExpectedOrgList: []string{"org-1", "org-2"},
		},
		{
			Organizations:   "org-1,org-2,",
			ExpectedOrgList: []string{"org-1", "org-2"},
		},
		{
			Organizations:   "org-1,     org-2    ",
			ExpectedOrgList: []string{"org-1", "org-2"},
		},
	}

	for _, tc := range tcs {
		config := Configuration{
			ForgejoOrg: tc.Organizations,
		}
		orgList := config.getOrganizations()
		assert.Equal(t, tc.ExpectedOrgList, orgList)
	}
}
