package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/plugin"

	"github.com/google/go-github/github"
	"github.com/mattermost/mattermost-server/model"
)

const COMMAND_HELP = `* |/github connect| - Connect your Mattermost account to your GitHub account
* |/github disconnect| - Disconnect your Mattermost account from your GitHub account
* |/github todo| - Get a list of unread messages and pull requests awaiting your review
* |/github subscribe owner/repo [feature]| - Subscribe the current channel to receive notifications about opened pull requests and issues for a repository
  * |feature| is a comma-delimited list of one or more the following:
    * issues
	* pulls
  * Defaults to "pulls,issues"
* |/github unsubscribe owner/repo| - Unsubscribe the current channel from a repository
* |/github me| - Display the connected GitHub account
* |/github settings| - Connect your Mattermost account to a GitHub account`

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

func getCommandResponse(responseType, text string) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: responseType,
		Text:         text,
		Username:     GITHUB_USERNAME,
		IconURL:      GITHUB_ICON_URL,
		Type:         model.POST_DEFAULT,
	}
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
		return nil, nil
	}

	if action == "connect" {
		resp := getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "[Click here to link your GitHub account.](http://localhost:8065/plugins/github/oauth/connect)")
		return resp, nil
	}

	ctx := context.Background()
	var githubClient *github.Client
	username := ""

	if info, err := p.getGitHubUserInfo(args.UserId); err != nil {
		text := "Unknown error."
		if err.ID == API_ERROR_ID_NOT_CONNECTED {
			text = "You must connect your account to GitHub first. Either click on the GitHub logo in the bottom left of the screen or enter `/github connect`."
		}
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, text), nil
	} else {
		githubClient = p.githubConnect(*info.Token)
		username = info.GitHubUsername
	}

	switch action {
	case "subscribe":
		features := "pulls,issues"

		if len(parameters) == 0 {
			return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Please specify a repository."), nil
		} else if len(parameters) > 1 {
			features = strings.Join(parameters[1:], ",")
		}

		repo := parameters[0]

		if err := p.Subscribe(context.Background(), githubClient, args.UserId, repo, args.ChannelId, features); err != nil {
			return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, err.Error()), nil
		}

		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, fmt.Sprintf("Successfully subscribed to %s.", repo)), nil
	case "unsubscribe":
		if len(parameters) == 0 {
			return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Please specify a repository."), nil
		}

		repo := parameters[0]

		if err := p.Unsubscribe(args.ChannelId, repo); err != nil {
			mlog.Error(err.Error())
			return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Encountered an error trying to unsubscribe. Please try again."), nil
		}

		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, fmt.Sprintf("Succesfully unsubscribed from %s.", repo)), nil
	case "disconnect":
		p.disconnectGitHubAccount(args.UserId)
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Disconnected your GitHub account."), nil
	case "todo":
		text, err := p.GetToDo(ctx, username, githubClient)
		if err != nil {
			mlog.Error(err.Error())
			return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Encountered an error getting your to do items."), nil
		}
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, text), nil
	case "me":
		gitUser, _, err := githubClient.Users.Get(ctx, "")
		if err != nil {
			return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Encountered an error getting your GitHub profile."), nil
		}

		text := fmt.Sprintf("You are connected to GitHub as:\n# [![image](%s =40x40) [%s](%s)](%s)", gitUser.GetAvatarURL(), gitUser.GetLogin(), gitUser.GetHTMLURL(), gitUser.GetHTMLURL())
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, text), nil
	case "help":
		text := "###### Mattermost GitHub Plugin - Slash Command Help\n" + strings.Replace(COMMAND_HELP, "|", "`", -1)

		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, text), nil
	case "settings":
		resp := getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "The settings for the GitHub Mattermost plugin can only be edited on the web or desktop apps.")
		resp.Type = "custom_git_settings"
		return resp, nil
	}

	return nil, nil
}
