package plugin

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/google/go-github/v31/github"
	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
)

const (
	featureIssues        = "issues"
	featurePulls         = "pulls"
	featurePushes        = "pushes"
	featureCreates       = "creates"
	featureDeletes       = "deletes"
	featureIssueComments = "issue_comments"
	featurePullReviews   = "pull_reviews"
)

var validFeatures = map[string]bool{
	featureIssues:        true,
	featurePulls:         true,
	featurePushes:        true,
	featureCreates:       true,
	featureDeletes:       true,
	featureIssueComments: true,
	featurePullReviews:   true,
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
			if f == featurePulls || f == featureIssues {
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

	owner, repo := parseOwnerAndRepo(parameters[0], p.getBaseURL())
	if repo == "" {
		if err := p.SubscribeOrg(ctx, githubClient, args.UserId, owner, args.ChannelId, features, flags); err != nil {
			return err.Error()
		}

		return fmt.Sprintf("Successfully subscribed to organization %s.", owner)
	}

	if err := p.Subscribe(ctx, githubClient, args.UserId, owner, repo, args.ChannelId, features, flags); err != nil {
		return err.Error()
	}

	msg := fmt.Sprintf("Successfully subscribed to %s.", repo)

	ghRepo, _, err := githubClient.Repositories.Get(ctx, owner, repo)
	if err != nil {
		p.API.LogWarn("Failed to fetch repository", "error", err.Error())
	} else if ghRepo != nil && ghRepo.GetPrivate() {
		msg += "\n\n**Warning:** You subscribed to a private repository. Anyone with access to this channel will be able to read the events getting posted here."
	}

	return msg
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
	githubClient := p.getGithubClient(userInfo)

	text, err := p.GetToDo(context.Background(), userInfo.GitHubUsername, githubClient)
	if err != nil {
		mlog.Error(err.Error())
		return "Encountered an error getting your to do items."
	}
	return text
}

func (p *Plugin) handleMe(_ *plugin.Context, _ *model.CommandArgs, _ []string, userInfo *GitHubUserInfo) string {
	githubClient := p.getGithubClient(userInfo)
	gitUser, _, err := githubClient.Users.Get(context.Background(), "")
	if err != nil {
		return "Encountered an error getting your GitHub profile."
	}

	text := fmt.Sprintf("You are connected to GitHub as:\n# [![image](%s =40x40)](%s) [%s](%s)", gitUser.GetAvatarURL(), gitUser.GetHTMLURL(), gitUser.GetLogin(), gitUser.GetHTMLURL())
	return text
}

func (p *Plugin) handleHelp(_ *plugin.Context, _ *model.CommandArgs, _ []string, _ *GitHubUserInfo) string {
	message, err := renderTemplate("helpText", p.getConfiguration())
	if err != nil {
		p.API.LogWarn("failed to render help template", "error", err.Error())
		return "Encountered an error posting help text."
	}

	return "###### Mattermost GitHub Plugin - Slash Command Help\n" + message
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
	command, action, parameters := parseCommand(args.Command)

	if command != "/github" {
		return &model.CommandResponse{}, nil
	}

	if action == "connect" {
		siteURL := p.API.GetConfig().ServiceSettings.SiteURL
		if siteURL == nil {
			p.postCommandResponse(args, "Encountered an error connecting to GitHub.")
			return &model.CommandResponse{}, nil
		}

		privateAllowed := false
		if len(parameters) > 0 {
			if len(parameters) != 1 || parameters[0] != "private" {
				p.postCommandResponse(args, fmt.Sprintf("Unknown command `%v`. Do you meant `/github connect private`?", args.Command))
				return &model.CommandResponse{}, nil
			}

			privateAllowed = true
		}

		qparams := ""
		if privateAllowed {
			if !p.getConfiguration().EnablePrivateRepo {
				p.postCommandResponse(args, "Private repositories are disabled. Please ask a System Admin to enabled them.")
				return &model.CommandResponse{}, nil
			}
			qparams = "?private=true"
		}

		msg := fmt.Sprintf("[Click here to link your GitHub account.](%s/plugins/github/oauth/connect%s)", *siteURL, qparams)
		p.postCommandResponse(args, msg)
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
