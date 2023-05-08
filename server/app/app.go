package app

import "golang.org/x/oauth2"

type App struct {
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

type UserSettings struct {
	SidebarButtons        string `json:"sidebar_buttons"`
	DailyReminder         bool   `json:"daily_reminder"`
	DailyReminderOnChange bool   `json:"daily_reminder_on_change"`
	Notifications         bool   `json:"notifications"`
}
