package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/github"
	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
)

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          "github",
		DisplayName:      "Github",
		Description:      "Integration with Github.",
		AutoComplete:     true,
		AutoCompleteDesc: "Available commands: register, deregister, todo, me, settings, help",
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

func (p *Plugin) ExecuteCommand(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	split := strings.Split(args.Command, " ")
	command := split[0]
	//parameters := []string{}
	action := ""
	if len(split) > 1 {
		action = split[1]
	}
	/*if len(split) > 2 {
		parameters = split[2:]
	}*/

	if command != "/github" {
		return nil, nil
	}

	if action == "register" {
		resp := getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "[Click here to link your GitHub account.](http://localhost:8065/plugins/github/oauth/connect)")
		return resp, nil
	}

	ctx := context.Background()
	var githubClient *github.Client
	username := ""

	if info, err := p.getGitHubUserInfo(args.UserId); err != nil {
		text := "Unknown error."
		if err.ID == API_ERROR_ID_NOT_CONNECTED {
			text = "You must connect your account to GitHub first. Either click on the GitHub logo in the bottom left of the screen or enter `/github register`."
		}
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, text), nil
	} else {
		githubClient = githubConnect(*info.Token)
		username = info.GitHubUsername
	}

	switch action {
	case "subscribe":
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Not yet implemented."), nil
	case "deregister":
		p.disconnectGitHubAccount(args.UserId)
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Unlinked your GitHub account."), nil
	case "todo":
		result, _, err := githubClient.Search.Issues(ctx, getReviewSearchQuery(username, p.GitHubOrg), &github.SearchOptions{})
		if err != nil {
			mlog.Error(err.Error())
		}

		text := fmt.Sprintf("You have %v pull requests awaiting your review:\n", result.GetTotal())

		for _, pr := range result.Issues {
			text += fmt.Sprintf("* %v\n", pr.GetURL())
		}

		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, text), nil
	}

	return nil, nil
}
