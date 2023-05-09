package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-plugin-github/server/app"
	serverplugin "github.com/mattermost/mattermost-plugin-github/server/plugin"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
)

const (
	flagFeatures = "features"
)

func (r *Runner) handleSubscriptions(c *plugin.Context, args *model.CommandArgs, parameters []string, userInfo *serverplugin.GitHubUserInfo) string {
	if len(parameters) == 0 {
		return "Invalid subscribe command. Available commands are 'list', 'add' and 'delete'."
	}

	command := parameters[0]
	parameters = parameters[1:]

	switch {
	case command == "list":
		return r.handleSubscriptionsList(c, args, parameters, userInfo)
	case command == "add":
		return r.handleSubscribesAdd(c, args, parameters, userInfo)
	case command == "delete":
		return r.handleUnsubscribe(c, args, parameters, userInfo)
	default:
		return fmt.Sprintf("Unknown subcommand %v", command)
	}
}

func (r *Runner) handleSubscriptionsList(_ *plugin.Context, args *model.CommandArgs, parameters []string, _ *serverplugin.GitHubUserInfo) string {
	txt := ""
	subs, err := r.serverPlugin.GetSubscriptionsByChannel(args.ChannelId)
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
}

func (r *Runner) handleSubscribesAdd(_ *plugin.Context, args *model.CommandArgs, parameters []string, userInfo *serverplugin.GitHubUserInfo) string {
	if len(parameters) == 0 {
		return "Please specify a repository."
	}

	config := r.configService.GetConfiguration()

	features := "pulls,issues,creates,deletes"
	flags := r.serverPlugin.SubscriptionFlags{}

	if len(parameters) > 1 {
		flagParams := parameters[1:]

		if len(flagParams)%2 != 0 {
			return "Please use the correct format for flags: --<name> <value>"
		}
		for i := 0; i < len(flagParams); i += 2 {
			flag := flagParams[i]
			value := flagParams[i+1]

			if !isFlag(flag) {
				return "Please use the correct format for flags: --<name> <value>"
			}
			parsedFlag := parseFlag(flag)

			if parsedFlag == flagFeatures {
				features = value
				continue
			}
			if err := flags.AddFlag(parsedFlag, value); err != nil {
				return fmt.Sprintf("Unsupported value for flag %s", flag)
			}
		}

		fs := strings.Split(features, ",")
		if SliceContainsString(fs, featureIssues) && SliceContainsString(fs, featureIssueCreation) {
			return "Feature list cannot contain both issue and issue_creations"
		}
		if SliceContainsString(fs, app.FeaturePulls) && SliceContainsString(fs, featurePullsMerged) {
			return "Feature list cannot contain both pulls and pulls_merged"
		}
		ok, ifs := r.app.ValidateFeatures(fs)
		if !ok {
			msg := fmt.Sprintf("Invalid feature(s) provided: %s", strings.Join(ifs, ","))
			if len(ifs) == 0 {
				msg = "Feature list must have \"pulls\" or \"issues\" when using a label."
			}
			return msg
		}
	}

	ctx := context.Background()
	githubClient := r.serverPlugin.GithubConnectUser(ctx, userInfo)

	owner, repo := r.serverPlugin.parseOwnerAndRepo(parameters[0], config.GetBaseURL())
	if repo == "" {
		if err := r.serverPlugin.SubscribeOrg(ctx, githubClient, args.UserId, owner, args.ChannelId, features, flags); err != nil {
			return err.Error()
		}

		return fmt.Sprintf("Successfully subscribed to organization %s.", owner)
	}

	if err := r.serverPlugin.Subscribe(ctx, githubClient, args.UserId, owner, repo, args.ChannelId, features, flags); err != nil {
		return err.Error()
	}
	repoLink := config.GetBaseURL() + owner + "/" + repo

	msg := fmt.Sprintf("Successfully subscribed to [%s](%s).", repo, repoLink)

	ghRepo, _, err := githubClient.Repositories.Get(ctx, owner, repo)
	if err != nil {
		r.pluginClient.Log.Warn("Failed to fetch repository", "error", err.Error())
	} else if ghRepo != nil && ghRepo.GetPrivate() {
		msg += "\n\n**Warning:** You subscribed to a private repository. Anyone with access to this channel will be able to read the events getting posted here."
	}

	return msg
}

func (r *Runner) handleSubscribe(c *plugin.Context, args *model.CommandArgs, parameters []string, userInfo *serverplugin.GitHubUserInfo) string {
	switch {
	case len(parameters) == 0:
		return "Please specify a repository or 'list' command."
	case len(parameters) == 1 && parameters[0] == "list":
		return r.handleSubscriptionsList(c, args, parameters[1:], userInfo)
	default:
		return r.handleSubscribesAdd(c, args, parameters, userInfo)
	}
}

func (r *Runner) handleUnsubscribe(_ *plugin.Context, args *model.CommandArgs, parameters []string, _ *serverplugin.GitHubUserInfo) string {
	if len(parameters) == 0 {
		return "Please specify a repository."
	}

	repo := parameters[0]

	if err := r.serverPlugin.Unsubscribe(args.ChannelId, repo); err != nil {
		r.pluginClient.Log.Warn("Failed to unsubscribe", "repo", repo, "error", err.Error())
		return "Encountered an error trying to unsubscribe. Please try again."
	}

	return fmt.Sprintf("Successfully unsubscribed from %s.", repo)
}

func isFlag(text string) bool {
	return strings.HasPrefix(text, "--")
}

func parseFlag(flag string) string {
	return strings.TrimPrefix(flag, "--")
}
