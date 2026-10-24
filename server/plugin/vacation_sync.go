// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

const (
	vacationSyncKeyPrefix    = "vacation_sync_"
	vacationSyncTimeout      = 30 * time.Second
	vacationCustomStatusIcon = "palm_tree"
	vacationGitHubEmoji      = ":palm_tree:"
	vacationGitHubMessage    = "On a vacation"
)

// vacationSyncState is stored per user so a GitHub status the plugin did not set is never cleared.
type vacationSyncState struct {
	Applied bool `json:"applied"`
}

// PreferencesHaveChanged reacts to custom status changes: the server saves the "recent custom
// statuses" preference whenever a user sets a custom status, and the newest one is first.
func (p *Plugin) PreferencesHaveChanged(_ *plugin.Context, preferences []model.Preference) {
	for _, preference := range preferences {
		if preference.Category != model.PreferenceCategoryCustomStatus || preference.Name != model.PreferenceNameRecentCustomStatuses {
			continue
		}

		var recent model.RecentCustomStatuses
		if err := json.Unmarshal([]byte(preference.Value), &recent); err != nil {
			p.client.Log.Warn("Failed to parse recent custom statuses", "userID", preference.UserId, "error", err.Error())
			continue
		}
		if len(recent) == 0 {
			continue
		}

		info, apiErr := p.getGitHubUserInfo(preference.UserId)
		if apiErr != nil || info.Settings == nil || !info.Settings.SyncVacationStatus {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), vacationSyncTimeout)
		p.applyVacationStatus(ctx, info, isOnVacation(&recent[0]))
		cancel()
	}
}

func isOnVacation(cs *model.CustomStatus) bool {
	if cs == nil || cs.Emoji != vacationCustomStatusIcon {
		return false
	}
	return cs.ExpiresAt.IsZero() || cs.ExpiresAt.After(time.Now())
}

// applyVacationStatus sets the user's GitHub status to Busy when they go on vacation and clears it
// when they come back, but only if the plugin was the one that set it.
func (p *Plugin) applyVacationStatus(ctx context.Context, info *GitHubUserInfo, onVacation bool) {
	key := vacationSyncKeyPrefix + info.UserID

	var state vacationSyncState
	if err := p.store.Get(key, &state); err != nil {
		p.client.Log.Warn("Failed to get vacation sync state", "userID", info.UserID, "error", err.Error())
		return
	}

	switch {
	case onVacation && !state.Applied:
		if err := p.graphQLConnect(info).SetBusyStatus(ctx, vacationGitHubMessage, vacationGitHubEmoji); err != nil {
			p.client.Log.Warn("Failed to set GitHub status to busy", "userID", info.UserID, "error", err.Error())
			p.CreateBotDMPost(info.UserID,
				"Failed to set your GitHub status to Busy while you are on vacation. Your GitHub connection may be missing the `user` permission. Please reconnect your account using `/github disconnect` followed by `/github connect`.",
				"custom_git_vacation_sync")
			return
		}
		state.Applied = true
	case !onVacation && state.Applied:
		if err := p.graphQLConnect(info).ClearStatus(ctx); err != nil {
			p.client.Log.Warn("Failed to clear GitHub busy status", "userID", info.UserID, "error", err.Error())
			return
		}
		state.Applied = false
	default:
		return
	}

	if _, err := p.store.Set(key, state); err != nil {
		p.client.Log.Warn("Failed to store vacation sync state", "userID", info.UserID, "error", err.Error())
	}
}
