package app

import (
	"time"

	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-plugin-api/experimental/bot/poster"
	"github.com/mattermost/mattermost-plugin-github/server/config"
	"golang.org/x/oauth2"
)

type App struct {
	config config.Service
	client *pluginapi.Client

	Poster poster.Poster

	BotUserID string

	WebhookBroker *WebhookBroker
	OauthBroker   *OAuthBroker
	emojiMap      map[string]string
}

const (
	GithubTokenKey         = "_githubtoken"
	ApiErrorIDNotConnected = "not_connected"
	GithubUsernameKey      = "_githubusername"

	RequestTimeout = 30 * time.Second
)

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

type APIErrorResponse struct {
	ID         string `json:"id"`
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
}

func (e *APIErrorResponse) Error() string {
	return e.Message
}
