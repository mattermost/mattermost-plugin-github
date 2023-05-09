package command

import (
	"github.com/mattermost/mattermost-plugin-github/server/app"
	serverplugin "github.com/mattermost/mattermost-plugin-github/server/plugin"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
)

const (
	settingNotifications = "notifications"
	settingReminders     = "reminders"
	settingOn            = "on"
	settingOff           = "off"
	settingOnChange      = "on-change"
)

func (r *Runner) handleSettings(_ *plugin.Context, _ *model.CommandArgs, parameters []string, userInfo *serverplugin.GitHubUserInfo) string {
	if len(parameters) < 2 {
		return "Please specify both a setting and value. Use `/github help` for more usage information."
	}

	setting := parameters[0]
	settingValue := parameters[1]

	switch setting {
	case settingNotifications:
		switch settingValue {
		case settingOn:
			userInfo.Settings.Notifications = true
		case settingOff:
			userInfo.Settings.Notifications = false
		default:
			return "Invalid value. Accepted values are: \"on\" or \"off\"."
		}
	case settingReminders:
		switch settingValue {
		case settingOn:
			userInfo.Settings.DailyReminder = true
			userInfo.Settings.DailyReminderOnChange = false
		case settingOff:
			userInfo.Settings.DailyReminder = false
			userInfo.Settings.DailyReminderOnChange = false
		case settingOnChange:
			userInfo.Settings.DailyReminder = true
			userInfo.Settings.DailyReminderOnChange = true
		default:
			return "Invalid value. Accepted values are: \"on\" or \"off\" or \"on-change\" ."
		}
	default:
		return "Unknown setting " + setting
	}

	if setting == settingNotifications {
		if userInfo.Settings.Notifications {
			err := r.serverPlugin.StoreGitHubToUserIDMapping(userInfo.GitHubUsername, userInfo.UserID)
			if err != nil {
				r.pluginClient.Log.Warn("Failed to store GitHub to userID mapping",
					"userID", userInfo.UserID,
					"GitHub username", userInfo.GitHubUsername,
					"error", err.Error())
			}
		} else {
			err := r.pluginClient.KV.Delete(userInfo.GitHubUsername + app.GithubUsernameKey)
			if err != nil {
				r.pluginClient.Log.Warn("Failed to delete GitHub to userID mapping",
					"userID", userInfo.UserID,
					"GitHub username", userInfo.GitHubUsername,
					"error", err.Error())
			}
		}
	}

	err := r.serverPlugin.StoreGitHubUserInfo(userInfo)
	if err != nil {
		r.pluginClient.Log.Warn("Failed to store github user info", "error", err.Error())
		return "Failed to store settings"
	}

	return "Settings updated."
}
