package command

import (
	"fmt"
	"strings"
	"unicode"

	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-plugin-api/experimental/command"
	"github.com/mattermost/mattermost-plugin-github/server/api"
	"github.com/mattermost/mattermost-plugin-github/server/config"
	serverplugin "github.com/mattermost/mattermost-plugin-github/server/plugin"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/pkg/errors"
)

// Register is a function that allows the runner to register commands with the mattermost server.
type Register func(*model.Command) error

// RegisterCommands should be called by the plugin to register all necessary commands
func RegisterCommands(registerFunc Register, config *config.Configuration) error {
	return registerFunc(getCommand(config))
}

func getCommand(config *config.Configuration) *model.Command {
	return &model.Command{
		Trigger:          "github",
		AutoComplete:     true,
		AutoCompleteDesc: "Available commands: connect, disconnect, todo, subscriptions, issue, me, mute, settings, help, about",
		AutoCompleteHint: "[command]",
		AutocompleteData: getAutocompleteData(config),
	}
}

// Runner handles commands.
type Runner struct {
	context      *plugin.Context
	args         *model.CommandArgs
	pluginClient *pluginapi.Client
	serverPlugin serverplugin.Plugin
	// poster             bot.Poster
	// playbookRunService app.PlaybookRunService
	// playbookService    app.PlaybookService
	configService config.Service
	// userInfoStore     app.UserInfoStore
	// userInfoTelemetry app.UserInfoTelemetry
	// permissions       *app.PermissionsService
}

// NewCommandRunner creates a command runner.
func NewCommandRunner(ctx *plugin.Context,
	args *model.CommandArgs,
	api *pluginapi.Client,
	configService config.Service,
) *Runner {
	return &Runner{
		context:       ctx,
		args:          args,
		pluginClient:  api,
		configService: configService,
	}
}

func (r *Runner) isValid() error {
	if r.context == nil || r.args == nil || r.pluginClient == nil {
		return errors.New("invalid arguments to command.Runner")
	}
	return nil
}

func (r *Runner) postCommandResponse(args *model.CommandArgs, text string) {
	post := &model.Post{
		UserId:    r.serverPlugin.BotUserID,
		ChannelId: args.ChannelId,
		RootId:    args.RootId,
		Message:   text,
	}
	r.pluginClient.Post.SendEphemeralPost(args.UserId, post)
}

// Returns the elements in a, that are not in b
func arrayDifference(a, b []string) []string {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}
	var diff []string
	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}

func (r *Runner) handleHelp(_ *plugin.Context, _ *model.CommandArgs, _ []string, _ *serverplugin.GitHubUserInfo) string {
	message, err := renderTemplate("helpText", r.configService.GetConfiguration())
	if err != nil {
		r.pluginClient.Log.Warn("Failed to render help template", "error", err.Error())
		return "Encountered an error posting help text."
	}

	return "###### Mattermost GitHub Plugin - Slash Command Help\n" + message
}

func (r *Runner) isAuthorizedSysAdmin(userID string) (bool, error) {
	user, err := r.pluginClient.User.Get(userID)
	if err != nil {
		return false, err
	}
	if !strings.Contains(user.Roles, "system_admin") {
		return false, nil
	}
	return true, nil
}

func (r *Runner) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	cmd, action, parameters := parseCommand(args.Command)

	if cmd != "/github" {
		return &model.CommandResponse{}, nil
	}

	if action == "about" {
		text, err := command.BuildInfo(*r.configService.GetManifest())
		if err != nil {
			text = errors.Wrap(err, "failed to get build info").Error()
		}
		r.postCommandResponse(args, text)
		return &model.CommandResponse{}, nil
	}

	if action == "setup" {
		message := r.handleSetup(c, args, parameters)
		if message != "" {
			r.postCommandResponse(args, message)
		}
		return &model.CommandResponse{}, nil
	}

	config := r.configService.GetConfiguration()

	if validationErr := config.IsValid(); validationErr != nil {
		isSysAdmin, err := r.isAuthorizedSysAdmin(args.UserId)
		var text string
		switch {
		case err != nil:
			text = "Error checking user's permissions"
			r.pluginClient.Log.Warn(text, "error", err.Error())
		case isSysAdmin:
			text = fmt.Sprintf("Before using this plugin, you'll need to configure it by running `/github setup`: %s", validationErr.Error())
		default:
			text = "Please contact your system administrator to correctly configure the GitHub plugin."
		}

		r.postCommandResponse(args, text)
		return &model.CommandResponse{}, nil
	}

	if action == "connect" {
		siteURL := r.pluginClient.Configuration.GetConfig().ServiceSettings.SiteURL
		if siteURL == nil {
			r.postCommandResponse(args, "Encountered an error connecting to GitHub.")
			return &model.CommandResponse{}, nil
		}

		privateAllowed := r.configService.GetConfiguration().ConnectToPrivateByDefault
		if len(parameters) > 0 {
			if privateAllowed {
				r.postCommandResponse(args, fmt.Sprintf("Unknown command `%v`. Do you meant `/github connect`?", args.Command))
				return &model.CommandResponse{}, nil
			}

			if len(parameters) != 1 || parameters[0] != "private" {
				r.postCommandResponse(args, fmt.Sprintf("Unknown command `%v`. Do you meant `/github connect private`?", args.Command))
				return &model.CommandResponse{}, nil
			}

			privateAllowed = true
		}

		qparams := ""
		if privateAllowed {
			if !r.configService.GetConfiguration().EnablePrivateRepo {
				r.postCommandResponse(args, "Private repositories are disabled. Please ask a System Admin to enabled them.")
				return &model.CommandResponse{}, nil
			}
			qparams = "?private=true"
		}

		msg := fmt.Sprintf("[Click here to link your GitHub account.](%s/plugins/%s/oauth/connect%s)", *siteURL, r.configService.GetManifest().Id, qparams)
		r.postCommandResponse(args, msg)
		return &model.CommandResponse{}, nil
	}

	info, apiErr := r.serverPlugin.GetGitHubUserInfo(args.UserId)
	if apiErr != nil {
		text := "Unknown error."
		if apiErr.ID == api.ApiErrorIDNotConnected {
			text = "You must connect your account to GitHub first. Either click on the GitHub logo in the bottom left of the screen or enter `/github connect`."
		}
		r.postCommandResponse(args, text)
		return &model.CommandResponse{}, nil
	}

	if f, ok := r.serverPlugin.CommandHandlers[action]; ok {
		message := f(c, args, parameters, info)
		if message != "" {
			r.postCommandResponse(args, message)
		}
		return &model.CommandResponse{}, nil
	}

	r.postCommandResponse(args, fmt.Sprintf("Unknown action %v", action))
	return &model.CommandResponse{}, nil
}

// parseCommand parses the entire command input string and retrieves the command, action and parameters
func parseCommand(input string) (command, action string, parameters []string) {
	split := make([]string, 0)
	current := ""
	inQuotes := false

	for _, char := range input {
		if unicode.IsSpace(char) {
			// keep whitespaces that are inside double qoutes
			if inQuotes {
				current += " "
				continue
			}

			// ignore successive whitespaces that are outside of double quotes
			if len(current) == 0 && !inQuotes {
				continue
			}

			// append the current word to the list & move on to the next word/expression
			split = append(split, current)
			current = ""
			continue
		}

		// append the current character to the current word
		current += string(char)

		if char == '"' {
			inQuotes = !inQuotes
		}
	}

	// append the last word/expression to the list
	if len(current) > 0 {
		split = append(split, current)
	}

	command = split[0]

	if len(split) > 1 {
		action = split[1]
	}

	if len(split) > 2 {
		parameters = split[2:]
	}

	return command, action, parameters
}

func SliceContainsString(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}
