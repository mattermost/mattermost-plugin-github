package command

import (
	"context"
	"fmt"

	"github.com/mattermost/mattermost-plugin-github/server/app"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
)

func (r *Runner) handleDisconnect(_ *plugin.Context, args *model.CommandArgs, _ []string, _ *app.GitHubUserInfo) string {
	r.serverPlugin.DisconnectGitHubAccount(args.UserId)
	return "Disconnected your GitHub account."
}

func (r *Runner) handleTodo(_ *plugin.Context, _ *model.CommandArgs, _ []string, userInfo *app.GitHubUserInfo) string {
	githubClient := r.serverPlugin.GithubConnectUser(context.Background(), userInfo)

	text, err := r.serverPlugin.GetToDo(context.Background(), userInfo.GitHubUsername, githubClient)
	if err != nil {
		r.pluginClient.Log.Warn("Failed get get Todos", "error", err.Error())
		return "Encountered an error getting your to do items."
	}

	return text
}

func (r *Runner) handleMe(_ *plugin.Context, _ *model.CommandArgs, _ []string, userInfo *app.GitHubUserInfo) string {
	githubClient := r.serverPlugin.GithubConnectUser(context.Background(), userInfo)
	gitUser, _, err := githubClient.Users.Get(context.Background(), "")
	if err != nil {
		return "Encountered an error getting your GitHub profile."
	}

	text := fmt.Sprintf("You are connected to GitHub as:\n# [![image](%s =40x40)](%s) [%s](%s)", gitUser.GetAvatarURL(), gitUser.GetHTMLURL(), gitUser.GetLogin(), gitUser.GetHTMLURL())
	return text
}
