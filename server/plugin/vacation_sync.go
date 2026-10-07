// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"context"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

const (
	vacationSyncJobKey       = "github_vacation_sync"
	vacationSyncKeyPrefix    = "vacation_sync_"
	vacationSyncInterval     = time.Minute
	vacationSyncUserTimeout  = 30 * time.Second
	vacationCustomStatusIcon = "palm_tree"
	vacationGitHubEmoji      = ":palm_tree:"
	vacationGitHubMessage    = "On a vacation"
)

// vacationSyncState is stored per opted-in user under vacationSyncKeyPrefix+userID.
type vacationSyncState struct {
	// Applied is true while the plugin has set the user's GitHub status to Busy.
	Applied bool `json:"applied"`
	// Notified is true once the user was told that the sync failed, to avoid repeating the DM every run.
	Notified bool `json:"notified"`
}

func isOnVacation(user *model.User) bool {
	cs := user.GetCustomStatus()
	if cs == nil || cs.Emoji != vacationCustomStatusIcon {
		return false
	}
	return cs.ExpiresAt.IsZero() || cs.ExpiresAt.After(time.Now())
}

func (p *Plugin) syncVacationStatuses() {
	var userIDs []string
	for page := 0; ; page++ {
		keys, err := p.store.ListKeys(page, keysPerPage, pluginapi.WithPrefix(vacationSyncKeyPrefix))
		if err != nil {
			p.client.Log.Warn("Failed to list users for vacation status sync", "page", page, "error", err.Error())
			break
		}
		for _, key := range keys {
			userIDs = append(userIDs, strings.TrimPrefix(key, vacationSyncKeyPrefix))
		}
		if len(keys) < keysPerPage {
			break
		}
	}

	for _, userID := range userIDs {
		ctx, cancel := context.WithTimeout(context.Background(), vacationSyncUserTimeout)
		p.syncUserVacationStatus(ctx, userID)
		cancel()
	}
}

func (p *Plugin) syncUserVacationStatus(ctx context.Context, userID string) {
	key := vacationSyncKeyPrefix + userID

	info, apiErr := p.getGitHubUserInfo(userID)
	if apiErr != nil {
		if apiErr.ID == apiErrorIDNotConnected {
			p.deleteVacationSyncState(key)
		}
		return
	}
	if info.Settings == nil || !info.Settings.SyncVacationStatus {
		p.deleteVacationSyncState(key)
		return
	}

	var state vacationSyncState
	if err := p.store.Get(key, &state); err != nil {
		p.client.Log.Warn("Failed to get vacation sync state", "userID", userID, "error", err.Error())
		return
	}

	user, err := p.client.User.Get(userID)
	if err != nil {
		p.client.Log.Warn("Failed to get user for vacation status sync", "userID", userID, "error", err.Error())
		return
	}

	next := state
	onVacation := isOnVacation(user)
	switch {
	case onVacation && !state.Applied:
		err = p.graphQLConnect(info).SetBusyStatus(ctx, vacationGitHubMessage, vacationGitHubEmoji)
		if err != nil {
			p.client.Log.Warn("Failed to set GitHub status to busy", "userID", userID, "error", err.Error())
			if !state.Notified {
				p.CreateBotDMPost(userID,
					"Failed to set your GitHub status to Busy while you are on vacation. Your GitHub connection may be missing the `user` permission. Please reconnect your account using `/github disconnect` followed by `/github connect`.",
					"custom_git_vacation_sync")
				next.Notified = true
			}
		} else {
			next.Applied = true
			next.Notified = false
		}
	case !onVacation && state.Applied:
		if err = p.graphQLConnect(info).ClearStatus(ctx); err != nil {
			p.client.Log.Warn("Failed to clear GitHub busy status", "userID", userID, "error", err.Error())
		} else {
			next.Applied = false
		}
	case !onVacation:
		next.Notified = false
	}

	if next == state {
		return
	}
	if _, err = p.store.Set(key, next); err != nil {
		p.client.Log.Warn("Failed to store vacation sync state", "userID", userID, "error", err.Error())
	}
}

func (p *Plugin) deleteVacationSyncState(key string) {
	if err := p.store.Delete(key); err != nil {
		p.client.Log.Warn("Failed to delete vacation sync state", "key", key, "error", err.Error())
	}
}
