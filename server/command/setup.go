package command

import (
	"fmt"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
)

func (r *Runner) handleSetup(c *plugin.Context, args *model.CommandArgs, parameters []string) string {
	userID := args.UserId
	isSysAdmin, err := r.isAuthorizedSysAdmin(userID)
	if err != nil {
		r.pluginClient.Log.Warn("Failed to check if user is System Admin", "error", err.Error())

		return "Error checking user's permissions"
	}

	if !isSysAdmin {
		return "Only System Admins are allowed to set up the plugin."
	}

	fm := &r.serverPlugin.FlowManager

	if len(parameters) == 0 {
		err = fm.StartSetupWizard(userID, "")
	} else {
		command := parameters[0]

		switch {
		case command == "oauth":
			err = fm.StartOauthWizard(userID)
		case command == "webhook":
			err = fm.StartWebhookWizard(userID)
		case command == "announcement":
			err = fm.StartAnnouncementWizard(userID)
		default:
			return fmt.Sprintf("Unknown subcommand %v", command)
		}
	}

	if err != nil {
		return err.Error()
	}

	return ""
}
