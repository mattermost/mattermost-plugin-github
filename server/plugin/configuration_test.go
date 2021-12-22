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
