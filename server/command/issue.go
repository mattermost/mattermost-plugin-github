package command

import (
	"fmt"
	"strings"

	serverplugin "github.com/mattermost/mattermost-plugin-github/server/plugin"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
)

const (
	wsEventCreateIssue = "createIssue"
)

func (r *Runner) handleIssue(_ *plugin.Context, args *model.CommandArgs, parameters []string, userInfo *serverplugin.GitHubUserInfo) string {
	if len(parameters) == 0 {
		return "Invalid issue command. Available command is 'create'."
	}

	command := parameters[0]
	parameters = parameters[1:]

	switch {
	case command == "create":
		r.openIssueCreateModal(args.UserId, args.ChannelId, strings.Join(parameters, " "))
		return ""
	default:
		return fmt.Sprintf("Unknown subcommand %v", command)
	}
}

func (r *Runner) openIssueCreateModal(userID string, channelID string, title string) {
	r.pluginClient.Frontend.PublishWebSocketEvent(
		wsEventCreateIssue,
		map[string]interface{}{
			"title":      title,
			"channel_id": channelID,
		},
		&model.WebsocketBroadcast{UserId: userID},
	)
}
