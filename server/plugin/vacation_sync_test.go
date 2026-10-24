// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
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
			assert.Equal(t, tc.expected, isOnVacation(tc.status))
		})
	}
}

type vacationSyncTestEnv struct {
	p        *Plugin
	store    *mocks.MockKvStore
	api      *plugintest.API
	requests chan map[string]any
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

func recentStatusPreference(t *testing.T, statuses ...model.CustomStatus) model.Preference {
	t.Helper()
	value, err := json.Marshal(statuses)
	require.NoError(t, err)
	return model.Preference{
		UserId:   MockUserID,
		Category: model.PreferenceCategoryCustomStatus,
		Name:     model.PreferenceNameRecentCustomStatuses,
		Value:    string(value),
	}
}

func TestPreferencesHaveChanged(t *testing.T) {
	vacation := model.CustomStatus{Emoji: "palm_tree", Text: "On a vacation"}
	meeting := model.CustomStatus{Emoji: "calendar", Text: "In a meeting"}

	t.Run("sets Busy when the newest status is vacation", func(t *testing.T) {
		env := setupVacationSyncTest(t, http.StatusOK, true)
		env.expectState(t, vacationSyncState{})
		env.store.EXPECT().Set(env.key, vacationSyncState{Applied: true}).Return(true, nil)

		env.p.PreferencesHaveChanged(nil, []model.Preference{recentStatusPreference(t, vacation, meeting)})

		input := receiveMutationInput(t, env.requests)
		assert.Equal(t, true, input["limitedAvailability"])
		assert.Equal(t, "On a vacation", input["message"])
		assert.Equal(t, ":palm_tree:", input["emoji"])
	})

	t.Run("clears Busy when a different status is set after vacation", func(t *testing.T) {
		env := setupVacationSyncTest(t, http.StatusOK, true)
		env.expectState(t, vacationSyncState{Applied: true})
		env.store.EXPECT().Set(env.key, vacationSyncState{}).Return(true, nil)

		env.p.PreferencesHaveChanged(nil, []model.Preference{recentStatusPreference(t, meeting, vacation)})

		input := receiveMutationInput(t, env.requests)
		assert.Equal(t, false, input["limitedAvailability"])
		assert.NotContains(t, input, "message")
	})

	t.Run("does not touch GitHub when Busy was not applied by the plugin", func(t *testing.T) {
		env := setupVacationSyncTest(t, http.StatusOK, true)
		env.expectState(t, vacationSyncState{})

		env.p.PreferencesHaveChanged(nil, []model.Preference{recentStatusPreference(t, meeting)})

		assert.Empty(t, env.requests)
	})

	t.Run("does nothing when already applied and vacation is set again", func(t *testing.T) {
		env := setupVacationSyncTest(t, http.StatusOK, true)
		env.expectState(t, vacationSyncState{Applied: true})

		env.p.PreferencesHaveChanged(nil, []model.Preference{recentStatusPreference(t, vacation)})

		assert.Empty(t, env.requests)
	})

	t.Run("ignores users who have not enabled the setting", func(t *testing.T) {
		env := setupVacationSyncTest(t, http.StatusOK, false)

		env.p.PreferencesHaveChanged(nil, []model.Preference{recentStatusPreference(t, vacation)})

		assert.Empty(t, env.requests)
	})

	t.Run("ignores unrelated preferences", func(t *testing.T) {
		env := setupVacationSyncTest(t, http.StatusOK, true)

		env.p.PreferencesHaveChanged(nil, []model.Preference{
			{UserId: MockUserID, Category: model.PreferenceCategoryTheme, Name: model.PreferenceNameRecentCustomStatuses, Value: "[]"},
			{UserId: MockUserID, Category: model.PreferenceCategoryCustomStatus, Name: "other", Value: "[]"},
		})

		assert.Empty(t, env.requests)
	})

	t.Run("notifies the user when GitHub rejects the update", func(t *testing.T) {
		env := setupVacationSyncTest(t, http.StatusForbidden, true)
		env.expectState(t, vacationSyncState{})
		env.api.On("LogWarn", "Failed to set GitHub status to busy", "userID", MockUserID, "error", mock.Anything).Once()
		env.api.On("GetDirectChannel", MockUserID, MockBotID).Return(&model.Channel{Id: MockChannelID}, nil)
		env.api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
			return post.ChannelId == MockChannelID && post.UserId == MockBotID
		})).Return(&model.Post{}, nil)

		env.p.PreferencesHaveChanged(nil, []model.Preference{recentStatusPreference(t, vacation)})

		receiveMutationInput(t, env.requests)
		env.api.AssertExpectations(t)
	})
}

func TestHandleSettingsVacationSync(t *testing.T) {
	newInfo := func(enabled bool) *GitHubUserInfo {
		return &GitHubUserInfo{UserID: MockUserID, GitHubUsername: MockUsername, Token: &oauth2.Token{AccessToken: MockAccessToken}, Settings: &UserSettings{SyncVacationStatus: enabled}}
	}

	t.Run("on while already on vacation sets Busy", func(t *testing.T) {
		env := setupVacationSyncTest(t, http.StatusOK, false)
		info := newInfo(false)
		env.expectUser(vacationUser(t, &model.CustomStatus{Emoji: "palm_tree", Text: "On a vacation"}))
		env.expectState(t, vacationSyncState{})
		env.store.EXPECT().Set(env.key, vacationSyncState{Applied: true}).Return(true, nil)
		env.store.EXPECT().Set(MockUserID+githubTokenKey, gomock.Any()).Return(true, nil)

		result := env.p.handleSettings(nil, nil, []string{settingVacationSync, settingOn}, info)

		assert.Equal(t, "Settings updated.", result)
		assert.True(t, info.Settings.SyncVacationStatus)
		input := receiveMutationInput(t, env.requests)
		assert.Equal(t, true, input["limitedAvailability"])
	})

	t.Run("on while not on vacation does not touch GitHub", func(t *testing.T) {
		env := setupVacationSyncTest(t, http.StatusOK, false)
		info := newInfo(false)
		env.expectUser(vacationUser(t, nil))
		env.expectState(t, vacationSyncState{})
		env.store.EXPECT().Set(MockUserID+githubTokenKey, gomock.Any()).Return(true, nil)

		result := env.p.handleSettings(nil, nil, []string{settingVacationSync, settingOn}, info)

		assert.Equal(t, "Settings updated.", result)
		assert.Empty(t, env.requests)
	})

	t.Run("off while Busy is applied clears the GitHub status", func(t *testing.T) {
		env := setupVacationSyncTest(t, http.StatusOK, true)
		info := newInfo(true)
		env.expectState(t, vacationSyncState{Applied: true})
		env.store.EXPECT().Set(env.key, vacationSyncState{}).Return(true, nil)
		env.store.EXPECT().Set(MockUserID+githubTokenKey, gomock.Any()).Return(true, nil)

		result := env.p.handleSettings(nil, nil, []string{settingVacationSync, settingOff}, info)

		assert.Equal(t, "Settings updated.", result)
		assert.False(t, info.Settings.SyncVacationStatus)
		input := receiveMutationInput(t, env.requests)
		assert.Equal(t, false, input["limitedAvailability"])
	})

	t.Run("off without an applied status does not touch GitHub", func(t *testing.T) {
		env := setupVacationSyncTest(t, http.StatusOK, true)
		info := newInfo(true)
		env.expectState(t, vacationSyncState{})
		env.store.EXPECT().Set(MockUserID+githubTokenKey, gomock.Any()).Return(true, nil)

		result := env.p.handleSettings(nil, nil, []string{settingVacationSync, settingOff}, info)

		assert.Equal(t, "Settings updated.", result)
		assert.Empty(t, env.requests)
	})

	t.Run("invalid value", func(t *testing.T) {
		env := setupVacationSyncTest(t, http.StatusOK, false)

		result := env.p.handleSettings(nil, nil, []string{settingVacationSync, "maybe"}, newInfo(false))

		assert.Equal(t, "Invalid value. Accepted values are: \"on\" or \"off\".", result)
	})
}
