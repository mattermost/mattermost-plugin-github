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

const COMMAND_HELP = `* |/github connect| - Connect your Mattermost account to your GitHub account
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
    * --exclude-org-member - events triggered by organization members will not be delivered (the Github organization config
		should be set, otherwise this flag has not effect)
* |/github unsubscribe owner/repo| - Unsubscribe the current channel from a repository
* |/github me| - Display the connected GitHub account
* |/github settings [setting] [value]| - Update your user settings
  * |setting| can be "notifications" or "reminders"
  * |value| can be "on" or "off"`

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          "github",
		DisplayName:      "Github",
		Description:      "Integration with Github.",
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

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	split := strings.Fields(args.Command)
	command := split[0]
	parameters := []string{}
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

		p.postCommandResponse(args, fmt.Sprintf("[Click here to link your GitHub account.](%s/plugins/github/oauth/connect)", *config.ServiceSettings.SiteURL))
		return &model.CommandResponse{}, nil
	}

	ctx := context.Background()
	var githubClient *github.Client

	info, apiErr := p.getGitHubUserInfo(args.UserId)
	if apiErr != nil {
		text := "Unknown error."
		if apiErr.ID == API_ERROR_ID_NOT_CONNECTED {
			text = "You must connect your account to GitHub first. Either click on the GitHub logo in the bottom left of the screen or enter `/github connect`."
		}
		p.postCommandResponse(args, text)
		return &model.CommandResponse{}, nil
	}

	githubClient = p.githubConnect(*info.Token)

	switch action {
	case "subscribe":
		config := p.getConfiguration()
		features := "pulls,issues,creates,deletes"
		flags := SubscriptionFlags{}

		txt := ""
		if len(parameters) == 0 {
			p.postCommandResponse(args, "Please specify a repository or 'list' command.")
			return &model.CommandResponse{}, nil
		} else if len(parameters) == 1 && parameters[0] == "list" {
			subs, err := p.GetSubscriptionsByChannel(args.ChannelId)
			if err != nil {
				p.postCommandResponse(args, err.Error())
				return &model.CommandResponse{}, nil
			}

			if len(subs) == 0 {
				txt = "Currently there are no subscriptions in this channel"
			} else {
				txt = "### Subscriptions in this channel\n"
			}
			for _, sub := range subs {
				txt += fmt.Sprintf("* `%s` - %s\n", strings.Trim(sub.Repository, "/"), sub.Features)
			}
			p.postCommandResponse(args, txt)
			return &model.CommandResponse{}, nil
		} else if len(parameters) > 1 {
			optionList := []string{}

			for _, element := range parameters[1:] {
				if isFlag(element) {
					flags.AddFlag(parseFlag(element))
				} else {
					optionList = append(optionList, element)
				}
			}

			if len(optionList) > 1 {
				p.postCommandResponse(args, "Just one list of features is allowed")
				return &model.CommandResponse{}, nil
			} else if len(optionList) == 1 {
				features = optionList[0]
			}
		}

		_, owner, repo := parseOwnerAndRepo(parameters[0], config.EnterpriseBaseURL)
		if repo == "" {
			if err := p.SubscribeOrg(context.Background(), githubClient, args.UserId, owner, args.ChannelId, features, flags); err != nil {
				p.postCommandResponse(args, err.Error())
				return &model.CommandResponse{}, nil
			}

			p.postCommandResponse(args, fmt.Sprintf("Successfully subscribed to organization %s.", owner))
			return &model.CommandResponse{}, nil
		}

		if err := p.Subscribe(context.Background(), githubClient, args.UserId, owner, repo, args.ChannelId, features, flags); err != nil {
			p.postCommandResponse(args, err.Error())
			return &model.CommandResponse{}, nil
		}

		p.postCommandResponse(args, fmt.Sprintf("Successfully subscribed to %s.", repo))
		return &model.CommandResponse{}, nil
	case "unsubscribe":
		if len(parameters) == 0 {
			p.postCommandResponse(args, "Please specify a repository.")
			return &model.CommandResponse{}, nil
		}

		repo := parameters[0]

		if err := p.Unsubscribe(args.ChannelId, repo); err != nil {
			mlog.Error(err.Error())
			p.postCommandResponse(args, "Encountered an error trying to unsubscribe. Please try again.")
			return &model.CommandResponse{}, nil
		}

		p.postCommandResponse(args, fmt.Sprintf("Succesfully unsubscribed from %s.", repo))
		return &model.CommandResponse{}, nil
	case "disconnect":
		p.disconnectGitHubAccount(args.UserId)
		p.postCommandResponse(args, "Disconnected your GitHub account.")
		return &model.CommandResponse{}, nil
	case "todo":
		text, err := p.GetToDo(ctx, info.GitHubUsername, githubClient)
		if err != nil {
			mlog.Error(err.Error())
			p.postCommandResponse(args, "Encountered an error getting your to do items.")
			return &model.CommandResponse{}, nil
		}
		p.postCommandResponse(args, text)
		return &model.CommandResponse{}, nil
	case "me":
		gitUser, _, err := githubClient.Users.Get(ctx, "")
		if err != nil {
			p.postCommandResponse(args, "Encountered an error getting your GitHub profile.")
			return &model.CommandResponse{}, nil
		}

		text := fmt.Sprintf("You are connected to GitHub as:\n# [![image](%s =40x40)](%s) [%s](%s)", gitUser.GetAvatarURL(), gitUser.GetHTMLURL(), gitUser.GetLogin(), gitUser.GetHTMLURL())
		p.postCommandResponse(args, text)
		return &model.CommandResponse{}, nil
	case "help":
		text := "###### Mattermost GitHub Plugin - Slash Command Help\n" + strings.Replace(COMMAND_HELP, "|", "`", -1)
		p.postCommandResponse(args, text)
		return &model.CommandResponse{}, nil
	case "":
		text := "###### Mattermost GitHub Plugin - Slash Command Help\n" + strings.Replace(COMMAND_HELP, "|", "`", -1)
		p.postCommandResponse(args, text)
		return &model.CommandResponse{}, nil
	case "settings":
		if len(parameters) < 2 {
			p.postCommandResponse(args, "Please specify both a setting and value. Use `/github help` for more usage information.")
			return &model.CommandResponse{}, nil
		}

		setting := parameters[0]
		if setting != SETTING_NOTIFICATIONS && setting != SETTING_REMINDERS {
			p.postCommandResponse(args, "Unknown setting.")
			return &model.CommandResponse{}, nil
		}

		strValue := parameters[1]
		value := false
		if strValue == SETTING_ON {
			value = true
		} else if strValue != SETTING_OFF {
			p.postCommandResponse(args, "Invalid value. Accepted values are: \"on\" or \"off\".")
			return &model.CommandResponse{}, nil
		}

		if setting == SETTING_NOTIFICATIONS {
			if value {
				p.storeGitHubToUserIDMapping(info.GitHubUsername, info.UserID)
			} else {
				p.API.KVDelete(info.GitHubUsername + GITHUB_USERNAME_KEY)
			}

			info.Settings.Notifications = value
		} else if setting == SETTING_REMINDERS {
			info.Settings.DailyReminder = value
		}

		p.storeGitHubUserInfo(info)

		p.postCommandResponse(args, "Settings updated.")
		return &model.CommandResponse{}, nil
	}

	p.postCommandResponse(args, fmt.Sprintf("Unknown action %v", action))

	return &model.CommandResponse{}, nil
}
