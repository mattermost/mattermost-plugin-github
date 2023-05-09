package app

import (
	"context"
	"strings"

	"github.com/google/go-github/github"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

func (a *App) createGithubEmojiMap() {
	baseGithubEmojiMap := map[string]string{
		"+1":         "+1",
		"-1":         "-1",
		"thumbsup":   "+1",
		"thumbsdown": "-1",
		"laughing":   "laugh",
		"confused":   "confused",
		"heart":      "heart",
		"tada":       "hooray",
		"rocket":     "rocket",
		"eyes":       "eyes",
	}

	a.emojiMap = map[string]string{}
	for systemEmoji := range model.SystemEmojis {
		for mmBase, ghBase := range baseGithubEmojiMap {
			if strings.HasPrefix(systemEmoji, mmBase) {
				a.emojiMap[systemEmoji] = ghBase
			}
		}
	}
}

func (a *App) getPostPropsForReaction(reaction *model.Reaction) (org, repo string, id float64, objectType string, ok bool) {
	post, err := a.client.Post.GetPost(reaction.PostId)
	if err != nil {
		a.client.Log.Debug("Error fetching post for reaction", "error", err.Error())
		return org, repo, id, objectType, false
	}

	// Getting the Github repository from notification post props
	repo, ok = post.GetProp(postPropGithubRepo).(string)
	if !ok || repo == "" {
		return org, repo, id, objectType, false
	}

	orgRepo := strings.Split(repo, "/")
	if len(orgRepo) != 2 {
		a.client.Log.Debug("Invalid organization repository")
		return org, repo, id, objectType, false
	}

	org, repo = orgRepo[0], orgRepo[1]

	// Getting the Github object id from notification post props
	id, ok = post.GetProp(postPropGithubObjectID).(float64)
	if !ok || id == 0 {
		return org, repo, id, objectType, false
	}

	// Getting the Github object type from notification post props
	objectType, ok = post.GetProp(postPropGithubObjectType).(string)
	if !ok || objectType == "" {
		return org, repo, id, objectType, false
	}

	return org, repo, id, objectType, true
}

func (a *App) ReactionHasBeenAdded(c *plugin.Context, reaction *model.Reaction) {
	githubEmoji := a.emojiMap[reaction.EmojiName]
	if githubEmoji == "" {
		a.client.Log.Warn("Emoji is not supported by Github", "Emoji", reaction.EmojiName)
		return
	}

	owner, repo, id, objectType, ok := a.getPostPropsForReaction(reaction)
	if !ok {
		return
	}

	info, appErr := a.GetGitHubUserInfo(reaction.UserId)
	if appErr != nil {
		if appErr.ID != ApiErrorIDNotConnected {
			a.client.Log.Debug("Error in getting user info", "error", appErr.Error())
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), RequestTimeout)
	defer cancel()
	ghClient := a.GithubConnectUser(ctx, info)
	switch objectType {
	case githubObjectTypeIssueComment:
		if _, _, err := ghClient.Reactions.CreateIssueCommentReaction(context.Background(), owner, repo, int64(id), githubEmoji); err != nil {
			a.client.Log.Debug("Error occurred while creating issue comment reaction", "error", err.Error())
			return
		}
	case githubObjectTypeIssue:
		if _, _, err := ghClient.Reactions.CreateIssueReaction(context.Background(), owner, repo, int(id), githubEmoji); err != nil {
			a.client.Log.Debug("Error occurred while creating issue reaction", "error", err.Error())
			return
		}
	case githubObjectTypePRReviewComment:
		if _, _, err := ghClient.Reactions.CreatePullRequestCommentReaction(context.Background(), owner, repo, int64(id), githubEmoji); err != nil {
			a.client.Log.Debug("Error occurred while creating PR review comment reaction", "error", err.Error())
			return
		}
	default:
		return
	}
}

func (a *App) ReactionHasBeenRemoved(c *plugin.Context, reaction *model.Reaction) {
	githubEmoji := a.emojiMap[reaction.EmojiName]
	if githubEmoji == "" {
		a.client.Log.Warn("Emoji is not supported by Github", "Emoji", reaction.EmojiName)
		return
	}

	owner, repo, id, objectType, ok := a.getPostPropsForReaction(reaction)
	if !ok {
		return
	}

	info, appErr := a.GetGitHubUserInfo(reaction.UserId)
	if appErr != nil {
		if appErr.ID != ApiErrorIDNotConnected {
			a.client.Log.Debug("Error in getting user info", "error", appErr.Error())
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), RequestTimeout)
	defer cancel()
	ghClient := a.GithubConnectUser(ctx, info)
	switch objectType {
	case githubObjectTypeIssueComment:
		reactions, _, err := ghClient.Reactions.ListIssueCommentReactions(context.Background(), owner, repo, int64(id), &github.ListOptions{})
		if err != nil {
			a.client.Log.Debug("Error getting issue comment reaction list", "error", err.Error())
			return
		}

		for _, reactionObj := range reactions {
			if info.UserID == reaction.UserId && a.emojiMap[reaction.EmojiName] == reactionObj.GetContent() {
				if _, err = ghClient.Reactions.DeleteIssueCommentReaction(context.Background(), owner, repo, int64(id), reactionObj.GetID()); err != nil {
					a.client.Log.Debug("Error occurred while removing issue comment reaction", "error", err.Error())
				}
				return
			}
		}
	case githubObjectTypeIssue:
		reactions, _, err := ghClient.Reactions.ListIssueReactions(context.Background(), owner, repo, int(id), &github.ListOptions{})
		if err != nil {
			a.client.Log.Debug("Error getting issue reaction list", "error", err.Error())
			return
		}

		for _, reactionObj := range reactions {
			if info.UserID == reaction.UserId && a.emojiMap[reaction.EmojiName] == reactionObj.GetContent() {
				if _, err = ghClient.Reactions.DeleteIssueReaction(context.Background(), owner, repo, int(id), reactionObj.GetID()); err != nil {
					a.client.Log.Debug("Error occurred while removing issue reaction", "error", err.Error())
				}
				return
			}
		}
	case githubObjectTypePRReviewComment:
		reactions, _, err := ghClient.Reactions.ListPullRequestCommentReactions(context.Background(), owner, repo, int64(id), &github.ListOptions{})
		if err != nil {
			a.client.Log.Debug("Error getting PR review comment reaction list", "error", err.Error())
			return
		}

		for _, reactionObj := range reactions {
			if info.UserID == reaction.UserId && a.emojiMap[reaction.EmojiName] == reactionObj.GetContent() {
				if _, err = ghClient.Reactions.DeletePullRequestCommentReaction(context.Background(), owner, repo, int64(id), reactionObj.GetID()); err != nil {
					a.client.Log.Debug("Error occurred while removing PR review comment reaction", "error", err.Error())
				}
				return
			}
		}
	default:
		return
	}
}
