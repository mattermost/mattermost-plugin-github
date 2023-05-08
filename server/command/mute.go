package command

import (
	"fmt"
	"strings"

	serverplugin "github.com/mattermost/mattermost-plugin-github/server/plugin"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
)

func (r *Runner) handleMuteCommand(_ *plugin.Context, args *model.CommandArgs, parameters []string, userInfo *serverplugin.GitHubUserInfo) string {
	if len(parameters) == 0 {
		return "Invalid mute command. Available commands are 'list', 'add' and 'delete'."
	}

	command := parameters[0]

	switch {
	case command == "list":
		return r.handleMuteList(args, userInfo)
	case command == "add":
		if len(parameters) != 2 {
			return "Invalid number of parameters supplied to " + command
		}
		return r.handleMuteAdd(args, parameters[1], userInfo)
	case command == "delete":
		if len(parameters) != 2 {
			return "Invalid number of parameters supplied to " + command
		}
		return r.handleUnmute(args, parameters[1], userInfo)
	case command == "delete-all":
		return r.handleUnmuteAll(args, userInfo)
	default:
		return fmt.Sprintf("Unknown subcommand %v", command)
	}
}

func (r *Runner) getMutedUsernames(userInfo *serverplugin.GitHubUserInfo) []string {
	var mutedUsernameBytes []byte
	err := r.pluginClient.KV.Get(userInfo.UserID+"-muted-users", &mutedUsernameBytes)
	if err != nil {
		return nil
	}
	mutedUsernames := string(mutedUsernameBytes)
	var mutedUsers []string
	if len(mutedUsernames) == 0 {
		return mutedUsers
	}
	mutedUsers = strings.Split(mutedUsernames, ",")
	return mutedUsers
}

func (r *Runner) handleMuteList(args *model.CommandArgs, userInfo *serverplugin.GitHubUserInfo) string {
	mutedUsernames := r.getMutedUsernames(userInfo)
	var mutedUsers string
	for _, user := range mutedUsernames {
		mutedUsers += fmt.Sprintf("- %v\n", user)
	}
	if len(mutedUsers) == 0 {
		return "You have no muted users"
	}
	return "Your muted users:\n" + mutedUsers
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (r *Runner) handleMuteAdd(args *model.CommandArgs, username string, userInfo *serverplugin.GitHubUserInfo) string {
	mutedUsernames := r.getMutedUsernames(userInfo)
	if contains(mutedUsernames, username) {
		return username + " is already muted"
	}

	if strings.Contains(username, ",") {
		return "Invalid username provided"
	}

	var mutedUsers string
	if len(mutedUsernames) > 0 {
		// , is a character not allowed in github usernames so we can split on them
		mutedUsers = strings.Join(mutedUsernames, ",") + "," + username
	} else {
		mutedUsers = username
	}

	_, err := r.pluginClient.KV.Set(userInfo.UserID+"-muted-users", []byte(mutedUsers))
	if err != nil {
		return "Error occurred saving list of muted users"
	}

	return fmt.Sprintf("`%v`", username) + " is now muted. You'll no longer receive notifications for comments in your PRs and issues."
}

func (r *Runner) handleUnmute(args *model.CommandArgs, username string, userInfo *serverplugin.GitHubUserInfo) string {
	mutedUsernames := r.getMutedUsernames(userInfo)
	userToMute := []string{username}
	newMutedList := arrayDifference(mutedUsernames, userToMute)

	_, err := r.pluginClient.KV.Set(userInfo.UserID+"-muted-users", []byte(strings.Join(newMutedList, ",")))
	if err != nil {
		return "Error occurred unmuting users"
	}

	return fmt.Sprintf("`%v`", username) + " is no longer muted"
}

func (r *Runner) handleUnmuteAll(args *model.CommandArgs, userInfo *serverplugin.GitHubUserInfo) string {
	_, err := r.pluginClient.KV.Set(userInfo.UserID+"-muted-users", []byte(""))
	if err != nil {
		return "Error occurred unmuting users"
	}

	return "Unmuted all users"
}
