package serializer

import (
	"context"

	"github.com/mattermost/mattermost-plugin-api/experimental/bot/logger"
	"golang.org/x/oauth2"
)

type Context struct {
	Ctx    context.Context
	UserID string
	Log    logger.Logger
}

type GitHubUserRequest struct {
	UserID string `json:"user_id"`
}

type GitHubUserResponse struct {
	Username string `json:"username"`
}

type ConnectedResponse struct {
	Connected           bool                   `json:"connected"`
	GitHubUsername      string                 `json:"github_username"`
	GitHubClientID      string                 `json:"github_client_id"`
	EnterpriseBaseURL   string                 `json:"enterprise_base_url,omitempty"`
	Organization        string                 `json:"organization"`
	UserSettings        *UserSettings          `json:"user_settings"`
	ClientConfiguration map[string]interface{} `json:"configuration"`
}

type UserSettings struct {
	SidebarButtons        string `json:"sidebar_buttons"`
	DailyReminder         bool   `json:"daily_reminder"`
	DailyReminderOnChange bool   `json:"daily_reminder_on_change"`
	Notifications         bool   `json:"notifications"`
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

type UserContext struct {
	Context
	GHInfo *GitHubUserInfo
}

type OAuthState struct {
	UserID         string `json:"user_id"`
	Token          string `json:"token"`
	PrivateAllowed bool   `json:"private_allowed"`
}
