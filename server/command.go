package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/mattermost/mattermost-server/v5/plugin"

	"github.com/google/go-github/v25/github"
	"github.com/mattermost/mattermost-server/v5/model"
)

const commandHelp = `* |/github connect [private]| - Connect your Mattermost account to your GitHub account. 
  * |private| is optional. If used, the github bot will ask for read access to your private repositories. If these repositories send webhook events to this Mattermost server, you will be notified of changes to those repositories.
* |/github disconnect| - Disconnect your Mattermost account from your GitHub account
* |/github todo| - Get a list of unread messages and pull requests awaiting your review
* |/github subscribe list| - Will list the current channel subscriptions
* |/github subscribe owner[/repo] [features] [flags]| - Subscribe the current channel to receive notifications about opened pull requests and issues for an organization or repository
  * |features| is a comma-delimited list of one or more the following:
    * issues - includes new and closed issues
	* pulls - includes new and closed pull requests
    * pushes - includes pushes
    * creates - includes branch and tag creations
    * deletes - includes branch and tag deletions
    * issue_comments - includes new issue comments
    * pull_reviews - includes pull request reviews
	* label:"<labelname>" - must include "pulls" or "issues" in feature list when using a label
	Defaults to "pulls,issues,creates,deletes"
  * |flags| currently supported:
    * --exclude-org-member - events triggered by organization members will not be delivered (the GitHub organization config
		should be set, otherwise this flag has not effect)
* |/github unsubscribe owner/repo| - Unsubscribe the current channel from a repository
* |/github me| - Display the connected GitHub account
* |/github settings [setting] [value]| - Update your user settings
  * |setting| can be "notifications" or "reminders"
  * |value| can be "on" or "off"`

var validFeatures = map[string]bool{
	"issues":         true,
	"pulls":          true,
	"pushes":         true,
	"creates":        true,
	"deletes":        true,
	"issue_comments": true,
	"pull_reviews":   true,
}

// validateFeatures returns false when 1 or more given features
// are invalid along with a list of the invalid features.
func validateFeatures(features []string) (bool, []string) {
	valid := true
	invalidFeatures := []string{}
	hasLabel := false
	for _, f := range features {
		if _, ok := validFeatures[f]; ok {
			continue
		}
		if strings.HasPrefix(f, "label") {
			hasLabel = true
			continue
		}
		invalidFeatures = append(invalidFeatures, f)
		valid = false
	}
	if valid && hasLabel {
		// must have "pulls" or "issues" in features when using a label
		for _, f := range features {
			if f == "pulls" || f == "issues" {
				return valid, invalidFeatures
			}
		}
		valid = false
	}
	return valid, invalidFeatures
}

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          "github",
		DisplayName:      "GitHub",
		Description:      "Integration with GitHub.",
		AutoComplete:     true,
		AutoCompleteDesc: "Available commands: connect, disconnect, todo, me, settings, subscribe, unsubscribe, help",
		AutoCompleteHint: "[command]",
	}
}

func (p *Plugin) postCommandResponse(args *model.CommandArgs, text string) {
	post := &model.Post{
		UserId:    p.BotUserID,
		ChannelId: args.ChannelId,
		Message:   text,
	}
	_ = p.API.SendEphemeralPost(args.UserId, post)
}

func (p *Plugin) getGithubClient(userInfo *GitHubUserInfo) *github.Client {
	return p.githubConnect(*userInfo.Token)
}

func (p *Plugin) handleSubscribe(_ *plugin.Context, args *model.CommandArgs, parameters []string, userInfo *GitHubUserInfo) string {
	config := p.getConfiguration()
	features := "pulls,issues,creates,deletes"
	flags := SubscriptionFlags{}

	txt := ""
	switch {
	case len(parameters) == 0:
		return "Please specify a repository or 'list' command."
	case len(parameters) == 1 && parameters[0] == "list":
		subs, err := p.GetSubscriptionsByChannel(args.ChannelId)
		if err != nil {
			return err.Error()
		}

		if len(subs) == 0 {
			txt = "Currently there are no subscriptions in this channel"
		} else {
			txt = "### Subscriptions in this channel\n"
		}
		for _, sub := range subs {
			subFlags := sub.Flags.String()
			txt += fmt.Sprintf("* `%s` - %s", strings.Trim(sub.Repository, "/"), sub.Features)
			if subFlags != "" {
				txt += fmt.Sprintf(" %s", subFlags)
			}
			txt += "\n"
		}
		return txt
	case len(parameters) > 1:
		var optionList []string

		for _, element := range parameters[1:] {
			if isFlag(element) {
				flags.AddFlag(parseFlag(element))
			} else {
				optionList = append(optionList, element)
			}
		}

		if len(optionList) > 1 {
			return "Just one list of features is allowed"
		} else if len(optionList) == 1 {
			features = optionList[0]
			fs := strings.Split(features, ",")
			ok, ifs := validateFeatures(fs)
			if !ok {
				msg := fmt.Sprintf("Invalid feature(s) provided: %s", strings.Join(ifs, ","))
				if len(ifs) == 0 {
					msg = "Feature list must have \"pulls\" or \"issues\" when using a label."
				}
				return msg
			}
		}
	}

	ctx := context.Background()
	githubClient := p.getGithubClient(userInfo)

	owner, repo := parseOwnerAndRepo(parameters[0], config.EnterpriseBaseURL)
	if repo == "" {
		if err := p.SubscribeOrg(ctx, githubClient, args.UserId, owner, args.ChannelId, features, flags); err != nil {
			return err.Error()
		}

		return fmt.Sprintf("Successfully subscribed to organization %s.", owner)
	}

	if err := p.Subscribe(ctx, githubClient, args.UserId, owner, repo, args.ChannelId, features, flags); err != nil {
		return err.Error()
	}

	return fmt.Sprintf("Successfully subscribed to %s.", repo)
}
func (p *Plugin) handleUnsubscribe(_ *plugin.Context, args *model.CommandArgs, parameters []string, _ *GitHubUserInfo) string {
	if len(parameters) == 0 {
		return "Please specify a repository."
	}

	repo := parameters[0]

	if err := p.Unsubscribe(args.ChannelId, repo); err != nil {
		mlog.Error(err.Error())
		return "Encountered an error trying to unsubscribe. Please try again."
	}

	return fmt.Sprintf("Successfully unsubscribed from %s.", repo)
}
func (p *Plugin) handleDisconnect(_ *plugin.Context, args *model.CommandArgs, _ []string, _ *GitHubUserInfo) string {
	p.disconnectGitHubAccount(args.UserId)
	return "Disconnected your GitHub account."
}
func (p *Plugin) handleTodo(_ *plugin.Context, _ *model.CommandArgs, _ []string, userInfo *GitHubUserInfo) string {
	ctx := context.Background()
	githubClient := p.getGithubClient(userInfo)

	text, err := p.GetToDo(ctx, userInfo.GitHubUsername, githubClient)
	if err != nil {
		mlog.Error(err.Error())
		return "Encountered an error getting your to do items."
	}
	return text
}
func (p *Plugin) handleMe(_ *plugin.Context, _ *model.CommandArgs, _ []string, userInfo *GitHubUserInfo) string {
	ctx := context.Background()
	githubClient := p.getGithubClient(userInfo)
	gitUser, _, err := githubClient.Users.Get(ctx, "")
	if err != nil {
		return "Encountered an error getting your GitHub profile."
	}

	text := fmt.Sprintf("You are connected to GitHub as:\n# [![image](%s =40x40)](%s) [%s](%s)", gitUser.GetAvatarURL(), gitUser.GetHTMLURL(), gitUser.GetLogin(), gitUser.GetHTMLURL())
	return text
}
func (p *Plugin) handleHelp(_ *plugin.Context, _ *model.CommandArgs, _ []string, _ *GitHubUserInfo) string {
	text := "###### Mattermost GitHub Plugin - Slash Command Help\n" + strings.Replace(commandHelp, "|", "`", -1)
	return text
}
func (p *Plugin) handleEmpty(_ *plugin.Context, _ *model.CommandArgs, _ []string, _ *GitHubUserInfo) string {
	text := "###### Mattermost GitHub Plugin - Slash Command Help\n" + strings.Replace(commandHelp, "|", "`", -1)
	return text
}
func (p *Plugin) handleSettings(_ *plugin.Context, _ *model.CommandArgs, parameters []string, userInfo *GitHubUserInfo) string {
	if len(parameters) < 2 {
		return "Please specify both a setting and value. Use `/github help` for more usage information."
	}

	setting := parameters[0]
	if setting != settingNotifications && setting != settingReminders {
		return "Unknown setting."
	}

	strValue := parameters[1]
	value := false
	if strValue == settingOn {
		value = true
	} else if strValue != settingOff {
		return "Invalid value. Accepted values are: \"on\" or \"off\"."
	}

	if setting == settingNotifications {
		if value {
			err := p.storeGitHubToUserIDMapping(userInfo.GitHubUsername, userInfo.UserID)
			if err != nil {
				mlog.Error(err.Error())
			}
		} else {
			err := p.API.KVDelete(userInfo.GitHubUsername + githubUsernameKey)
			if err != nil {
				mlog.Error(err.Error())
			}
		}

		userInfo.Settings.Notifications = value
	} else if setting == settingReminders {
		userInfo.Settings.DailyReminder = value
	}

	err := p.storeGitHubUserInfo(userInfo)
	if err != nil {
		mlog.Error(err.Error())
		return "Failed to store settings"
	}

	return "Settings updated."
}

type CommandHandleFunc func(c *plugin.Context, args *model.CommandArgs, parameters []string, userInfo *GitHubUserInfo) string

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	split := strings.Fields(args.Command)
	command := split[0]
	var parameters []string
	action := ""
	if len(split) > 1 {
		action = split[1]
	}
	if len(split) > 2 {
		parameters = split[2:]
	}

	if command != "/github" {
		return &model.CommandResponse{}, nil
	}

	if action == "connect" {
		config := p.API.GetConfig()
		if config.ServiceSettings.SiteURL == nil {
			p.postCommandResponse(args, "Encountered an error connecting to GitHub.")
			return &model.CommandResponse{}, nil
		}

		qparams := ""
		if len(parameters) == 1 && parameters[0] == "private" {
			qparams = "?private=true"
		}

		p.postCommandResponse(args, fmt.Sprintf("[Click here to link your GitHub account.](%s/plugins/github/oauth/connect%s)", *config.ServiceSettings.SiteURL, qparams))
		return &model.CommandResponse{}, nil
	}

	info, apiErr := p.getGitHubUserInfo(args.UserId)
	if apiErr != nil {
		text := "Unknown error."
		if apiErr.ID == apiErrorIDNotConnected {
			text = "You must connect your account to GitHub first. Either click on the GitHub logo in the bottom left of the screen or enter `/github connect`."
		}
		p.postCommandResponse(args, text)
		return &model.CommandResponse{}, nil
	}

	if f, ok := p.CommandHandlers[action]; ok {
		message := f(c, args, parameters, info)
		p.postCommandResponse(args, message)
		return &model.CommandResponse{}, nil
	}

	p.postCommandResponse(args, fmt.Sprintf("Unknown action %v", action))
	return &model.CommandResponse{}, nil
}
