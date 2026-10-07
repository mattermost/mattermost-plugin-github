// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-github/server/mocks"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
)

func vacationUser(t *testing.T, cs *model.CustomStatus) *model.User {
	t.Helper()
	user := &model.User{Id: MockUserID, Props: model.StringMap{}}
	if cs != nil {
		require.NoError(t, user.SetCustomStatus(cs))
	}
	return user
}

func TestIsOnVacation(t *testing.T) {
	tests := []struct {
		name     string
		status   *model.CustomStatus
		expected bool
	}{
		{name: "no custom status", status: nil, expected: false},
		{name: "other custom status", status: &model.CustomStatus{Emoji: "calendar", Text: "In a meeting"}, expected: false},
		{name: "vacation without expiry", status: &model.CustomStatus{Emoji: "palm_tree", Text: "On a vacation"}, expected: true},
		{name: "vacation not yet expired", status: &model.CustomStatus{Emoji: "palm_tree", ExpiresAt: time.Now().Add(time.Hour)}, expected: true},
		{name: "vacation expired", status: &model.CustomStatus{Emoji: "palm_tree", ExpiresAt: time.Now().Add(-time.Hour)}, expected: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, isOnVacation(vacationUser(t, tc.status)))
		})
	}
}

type vacationSyncTestEnv struct {
	p        *Plugin
	store    *mocks.MockKvStore
	api      *plugintest.API
	requests chan map[string]any
	info     *GitHubUserInfo
	key      string
}

func setupVacationSyncTest(t *testing.T, graphQLStatus int, syncEnabled bool) *vacationSyncTestEnv {
	t.Helper()

	requests := make(chan map[string]any, 10)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		requests <- payload
		w.WriteHeader(graphQLStatus)
		_, err = w.Write([]byte(`{"data":{"changeUserStatus":{"status":{"message":"ok"}}}}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	ctrl := gomock.NewController(t)
	store := mocks.NewMockKvStore(ctrl)
	api := &plugintest.API{}
	p := getPluginTest(api, store)

	encryptionKey := "dummyEncryptKey1"
	p.setConfiguration(&Configuration{EncryptionKey: encryptionKey, EnterpriseBaseURL: server.URL})
	encryptedToken, err := encrypt([]byte(encryptionKey), MockAccessToken)
	require.NoError(t, err)

	stored, err := json.Marshal(&GitHubUserInfo{
		UserID:         MockUserID,
		GitHubUsername: MockUsername,
		Token:          &oauth2.Token{AccessToken: encryptedToken},
		Settings:       &UserSettings{SyncVacationStatus: syncEnabled},
	})
	require.NoError(t, err)
	store.EXPECT().Get(MockUserID+githubTokenKey, gomock.Any()).DoAndReturn(func(_ string, out any) error {
		return json.Unmarshal(stored, out)
	}).AnyTimes()

	return &vacationSyncTestEnv{
		p:        p,
		store:    store,
		api:      api,
		requests: requests,
		key:      vacationSyncKeyPrefix + MockUserID,
	}
}

func (e *vacationSyncTestEnv) expectState(t *testing.T, state vacationSyncState) {
	t.Helper()
	e.store.EXPECT().Get(e.key, gomock.Any()).DoAndReturn(func(_ string, out any) error {
		b, err := json.Marshal(state)
		require.NoError(t, err)
		return json.Unmarshal(b, out)
	})
}

func (e *vacationSyncTestEnv) expectUser(user *model.User) {
	e.api.On("GetUser", MockUserID).Return(user, nil)
}

func receiveMutationInput(t *testing.T, requests chan map[string]any) map[string]any {
	t.Helper()
	select {
	case payload := <-requests:
		variables, ok := payload["variables"].(map[string]any)
		require.True(t, ok)
		input, ok := variables["input"].(map[string]any)
		require.True(t, ok)
		return input
	case <-time.After(time.Second):
		require.FailNow(t, "expected a GraphQL request")
		return nil
	}
}

func TestSyncUserVacationStatus(t *testing.T) {
	vacationStatus := &model.CustomStatus{Emoji: "palm_tree", Text: "On a vacation"}

	t.Run("sets Busy when user goes on vacation", func(t *testing.T) {
		env := setupVacationSyncTest(t, http.StatusOK, true)
		env.expectState(t, vacationSyncState{})
		env.expectUser(vacationUser(t, vacationStatus))
		env.store.EXPECT().Set(env.key, vacationSyncState{Applied: true}).Return(true, nil)

		env.p.syncUserVacationStatus(context.Background(), MockUserID)

		input := receiveMutationInput(t, env.requests)
		assert.Equal(t, true, input["limitedAvailability"])
		assert.Equal(t, "On a vacation", input["message"])
		assert.Equal(t, ":palm_tree:", input["emoji"])
	})

	t.Run("clears status when user returns from vacation", func(t *testing.T) {
		env := setupVacationSyncTest(t, http.StatusOK, true)
		env.expectState(t, vacationSyncState{Applied: true})
		env.expectUser(vacationUser(t, nil))
		env.store.EXPECT().Set(env.key, vacationSyncState{}).Return(true, nil)

		env.p.syncUserVacationStatus(context.Background(), MockUserID)

		input := receiveMutationInput(t, env.requests)
		assert.Equal(t, false, input["limitedAvailability"])
		assert.NotContains(t, input, "message")
	})

	t.Run("does nothing when already applied and still on vacation", func(t *testing.T) {
		env := setupVacationSyncTest(t, http.StatusOK, true)
		env.expectState(t, vacationSyncState{Applied: true})
		env.expectUser(vacationUser(t, vacationStatus))

		env.p.syncUserVacationStatus(context.Background(), MockUserID)

		assert.Empty(t, env.requests)
	})

	t.Run("does not touch GitHub status when never on vacation", func(t *testing.T) {
		env := setupVacationSyncTest(t, http.StatusOK, true)
		env.expectState(t, vacationSyncState{})
		env.expectUser(vacationUser(t, nil))

		env.p.syncUserVacationStatus(context.Background(), MockUserID)

		assert.Empty(t, env.requests)
	})

	t.Run("removes marker when setting is disabled", func(t *testing.T) {
		env := setupVacationSyncTest(t, http.StatusOK, false)
		env.store.EXPECT().Delete(env.key).Return(nil)

		env.p.syncUserVacationStatus(context.Background(), MockUserID)

		assert.Empty(t, env.requests)
	})

	t.Run("notifies the user once when GitHub rejects the update", func(t *testing.T) {
		env := setupVacationSyncTest(t, http.StatusForbidden, true)
		env.expectState(t, vacationSyncState{})
		env.expectUser(vacationUser(t, vacationStatus))
		env.store.EXPECT().Set(env.key, vacationSyncState{Notified: true}).Return(true, nil)
		env.api.On("LogWarn", "Failed to set GitHub status to busy", "userID", MockUserID, "error", mock.Anything).Once()
		env.api.On("GetDirectChannel", MockUserID, MockBotID).Return(&model.Channel{Id: MockChannelID}, nil)
		env.api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
			return post.ChannelId == MockChannelID && post.UserId == MockBotID
		})).Return(&model.Post{}, nil)

		env.p.syncUserVacationStatus(context.Background(), MockUserID)

		receiveMutationInput(t, env.requests)
		env.api.AssertExpectations(t)
	})

	t.Run("does not notify again after a previous failure", func(t *testing.T) {
		env := setupVacationSyncTest(t, http.StatusForbidden, true)
		env.expectState(t, vacationSyncState{Notified: true})
		env.expectUser(vacationUser(t, vacationStatus))
		env.api.On("LogWarn", "Failed to set GitHub status to busy", "userID", MockUserID, "error", mock.Anything).Once()

		env.p.syncUserVacationStatus(context.Background(), MockUserID)

		receiveMutationInput(t, env.requests)
		env.api.AssertNotCalled(t, "CreatePost", mock.Anything)
	})
}

func TestHandleSettingsVacationSync(t *testing.T) {
	t.Run("on stores the sync marker", func(t *testing.T) {
		env := setupVacationSyncTest(t, http.StatusOK, false)
		info := &GitHubUserInfo{UserID: MockUserID, GitHubUsername: MockUsername, Token: &oauth2.Token{AccessToken: MockAccessToken}, Settings: &UserSettings{}}
		env.store.EXPECT().Set(env.key, vacationSyncState{}).Return(true, nil)
		env.store.EXPECT().Set(MockUserID+githubTokenKey, gomock.Any()).Return(true, nil)

		result := env.p.handleSettings(nil, nil, []string{settingVacationSync, settingOn}, info)

		assert.Equal(t, "Settings updated.", result)
		assert.True(t, info.Settings.SyncVacationStatus)
	})

	t.Run("off while Busy is applied clears GitHub status and the marker", func(t *testing.T) {
		env := setupVacationSyncTest(t, http.StatusOK, true)
		info := &GitHubUserInfo{UserID: MockUserID, GitHubUsername: MockUsername, Token: &oauth2.Token{AccessToken: MockAccessToken}, Settings: &UserSettings{SyncVacationStatus: true}}
		env.expectState(t, vacationSyncState{Applied: true})
		env.store.EXPECT().Delete(env.key).Return(nil)
		env.store.EXPECT().Set(MockUserID+githubTokenKey, gomock.Any()).Return(true, nil)

		result := env.p.handleSettings(nil, nil, []string{settingVacationSync, settingOff}, info)

		assert.Equal(t, "Settings updated.", result)
		assert.False(t, info.Settings.SyncVacationStatus)
		input := receiveMutationInput(t, env.requests)
		assert.Equal(t, false, input["limitedAvailability"])
	})

	t.Run("off without an applied status only removes the marker", func(t *testing.T) {
		env := setupVacationSyncTest(t, http.StatusOK, true)
		info := &GitHubUserInfo{UserID: MockUserID, GitHubUsername: MockUsername, Token: &oauth2.Token{AccessToken: MockAccessToken}, Settings: &UserSettings{SyncVacationStatus: true}}
		env.expectState(t, vacationSyncState{})
		env.store.EXPECT().Delete(env.key).Return(nil)
		env.store.EXPECT().Set(MockUserID+githubTokenKey, gomock.Any()).Return(true, nil)

		result := env.p.handleSettings(nil, nil, []string{settingVacationSync, settingOff}, info)

		assert.Equal(t, "Settings updated.", result)
		assert.Empty(t, env.requests)
	})

	t.Run("invalid value", func(t *testing.T) {
		env := setupVacationSyncTest(t, http.StatusOK, false)
		info := &GitHubUserInfo{UserID: MockUserID, Settings: &UserSettings{}}

		result := env.p.handleSettings(nil, nil, []string{settingVacationSync, "maybe"}, info)

		assert.Equal(t, "Invalid value. Accepted values are: \"on\" or \"off\".", result)
	})
}
