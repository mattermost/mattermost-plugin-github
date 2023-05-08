package command

import (
	"github.com/mattermost/mattermost-plugin-api/experimental/command"
	"github.com/mattermost/mattermost-plugin-github/server/config"
	"github.com/mattermost/mattermost-server/v6/model"
)

func getAutocompleteData(config *config.Configuration) *model.AutocompleteData {
	if !config.IsOAuthConfigured() {
		github := model.NewAutocompleteData("github", "[command]", "Available commands: setup, about")

		setup := model.NewAutocompleteData("setup", "", "Set up the GitHub plugin")
		setup.RoleID = model.SystemAdminRoleId
		github.AddCommand(setup)

		about := command.BuildInfoAutocomplete("about")
		github.AddCommand(about)

		return github
	}

	github := model.NewAutocompleteData("github", "[command]", "Available commands: connect, disconnect, todo, subscriptions, issue, me, mute, settings, help, about")

	connect := model.NewAutocompleteData("connect", "", "Connect your Mattermost account to your GitHub account")
	if config.EnablePrivateRepo {
		if config.ConnectToPrivateByDefault {
			connect = model.NewAutocompleteData("connect", "", "Connect your Mattermost account to your GitHub account. Read access to your private repositories will be requested")
		} else {
			private := model.NewAutocompleteData("private", "(optional)", "If used, read access to your private repositories will be requested")
			connect.AddCommand(private)
		}
	}
	github.AddCommand(connect)

	disconnect := model.NewAutocompleteData("disconnect", "", "Disconnect your Mattermost account from your GitHub account")
	github.AddCommand(disconnect)

	todo := model.NewAutocompleteData("todo", "", "Get a list of unread messages and pull requests awaiting your review")
	github.AddCommand(todo)

	subscriptions := model.NewAutocompleteData("subscriptions", "[command]", "Available commands: list, add, delete")

	subscribeList := model.NewAutocompleteData("list", "", "List the current channel subscriptions")
	subscriptions.AddCommand(subscribeList)

	subscriptionsAdd := model.NewAutocompleteData("add", "[owner/repo] [features] [flags]", "Subscribe the current channel to receive notifications about opened pull requests and issues for an organization or repository. [features] and [flags] are optional arguments")
	subscriptionsAdd.AddTextArgument("Owner/repo to subscribe to", "[owner/repo]", "")
	subscriptionsAdd.AddNamedTextArgument("features", "Comma-delimited list of one or more of: issues, pulls, pulls_merged, pushes, creates, deletes, issue_creations, issue_comments, pull_reviews, label:\"<labelname>\". Defaults to pulls,issues,creates,deletes", "", `/[^,-\s]+(,[^,-\s]+)*/`, false)

	if config.GitHubOrg != "" {
		subscriptionsAdd.AddNamedStaticListArgument("exclude-org-member", "Events triggered by organization members will not be delivered (the organization config should be set, otherwise this flag has not effect)", false, []model.AutocompleteListItem{
			{
				Item:     "true",
				HelpText: "Exclude posts from members of the configured organization",
			},
			{
				Item:     "false",
				HelpText: "Include posts from members of the configured organization",
			},
		})
	}

	subscriptionsAdd.AddNamedStaticListArgument("render-style", "Determine the rendering style of various notifications.", false, []model.AutocompleteListItem{
		{
			Item:     "default",
			HelpText: "The default rendering style for all notifications (includes all information).",
		},
		{
			Item:     "skip-body",
			HelpText: "Skips the body part of various long notifications that have a body (e.g. new PRs and new issues).",
		},
		{
			Item:     "collapsed",
			HelpText: "Notifications come in a one-line format, without enlarged fonts or advanced layouts.",
		},
	})

	subscriptions.AddCommand(subscriptionsAdd)
	subscriptionsDelete := model.NewAutocompleteData("delete", "[owner/repo]", "Unsubscribe the current channel from an organization or repository")
	subscriptionsDelete.AddTextArgument("Owner/repo to unsubscribe from", "[owner/repo]", "")
	subscriptions.AddCommand(subscriptionsDelete)

	github.AddCommand(subscriptions)

	issue := model.NewAutocompleteData("issue", "[command]", "Available commands: create")

	issueCreate := model.NewAutocompleteData("create", "[title]", "Open a dialog to create a new issue in GitHub, using the title if provided")
	issueCreate.AddTextArgument("Title for the GitHub issue", "[title]", "")
	issue.AddCommand(issueCreate)

	github.AddCommand(issue)

	me := model.NewAutocompleteData("me", "", "Display the connected GitHub account")
	github.AddCommand(me)

	mute := model.NewAutocompleteData("mute", "[command]", "Available commands: list, add, delete, delete-all")

	muteAdd := model.NewAutocompleteData("add", "[github username]", "Mute notifications from the provided GitHub user")
	muteAdd.AddTextArgument("GitHub user to mute", "[username]", "")
	mute.AddCommand(muteAdd)

	muteDelete := model.NewAutocompleteData("delete", "[github username]", "Unmute notifications from the provided GitHub user")
	muteDelete.AddTextArgument("GitHub user to unmute", "[username]", "")
	mute.AddCommand(muteDelete)

	github.AddCommand(mute)

	muteDeleteAll := model.NewAutocompleteData("delete-all", "", "Unmute all muted GitHub users")
	mute.AddCommand(muteDeleteAll)

	muteList := model.NewAutocompleteData("list", "", "List muted GitHub users")
	mute.AddCommand(muteList)

	settings := model.NewAutocompleteData("settings", "[setting] [value]", "Update your user settings")

	settingNotifications := model.NewAutocompleteData("notifications", "", "Turn notifications on/off")
	settingValue := []model.AutocompleteListItem{{
		HelpText: "Turn notifications on",
		Item:     "on",
	}, {
		HelpText: "Turn notifications off",
		Item:     "off",
	}}
	settingNotifications.AddStaticListArgument("", true, settingValue)
	settings.AddCommand(settingNotifications)

	remainderNotifications := model.NewAutocompleteData("reminders", "", "Turn notifications on/off")
	settingValue = []model.AutocompleteListItem{{
		HelpText: "Turn reminders on",
		Item:     "on",
	}, {
		HelpText: "Turn reminders off",
		Item:     "off",
	}, {
		HelpText: "Turn reminders on, but only get reminders if any changes have occurred since the previous day's reminder",
		Item:     settingOnChange,
	}}
	remainderNotifications.AddStaticListArgument("", true, settingValue)
	settings.AddCommand(remainderNotifications)

	github.AddCommand(settings)

	setup := model.NewAutocompleteData("setup", "[command]", "Available commands: oauth, webhook, announcement")
	setup.RoleID = model.SystemAdminRoleId
	setup.AddCommand(model.NewAutocompleteData("oauth", "", "Set up the OAuth2 Application in GitHub"))
	setup.AddCommand(model.NewAutocompleteData("webhook", "", "Create a webhook from GitHub to Mattermost"))
	setup.AddCommand(model.NewAutocompleteData("announcement", "", "Announce to your team that they can use GitHub integration"))
	github.AddCommand(setup)

	help := model.NewAutocompleteData("help", "", "Display Slash Command help text")
	github.AddCommand(help)

	about := command.BuildInfoAutocomplete("about")
	github.AddCommand(about)

	return github
}
