// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/golang/mock/gomock"
	"github.com/google/go-github/v54/github"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/mattermost/mattermost-plugin-github/server/mocks"
)

const (
	testOldKey = "old_encryption_key_32chX" // 24 bytes = valid AES-192 key
	testNewKey = "new_encryption_key_32chX" // 24 bytes = valid AES-192 key
)

func setupRotationTest(t *testing.T) (*Plugin, *plugintest.API, *mocks.MockKvStore, *gomock.Controller) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockKvStore := mocks.NewMockKvStore(ctrl)

	api := &plugintest.API{}
	p := NewPlugin()
	p.store = mockKvStore
	p.BotUserID = MockBotID
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)

	p.setConfiguration(&Configuration{EncryptionKey: testNewKey})

	api.On("KVSetWithOptions", mock.Anything, mock.Anything, mock.Anything).Return(true, nil).Maybe()
	api.On("LogError", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogAuditRec", mock.Anything).Maybe()

	return p, api, mockKvStore, ctrl
}

func TestReEncryptUserData_HappyPath(t *testing.T) {
	p, api, mockKvStore, ctrl := setupRotationTest(t)
	defer ctrl.Finish()

	encryptedToken, err := encrypt([]byte(testOldKey), MockAccessToken)
	require.NoError(t, err)

	userInfo := &GitHubUserInfo{
		UserID:         "user1",
		GitHubUsername: "ghuser1",
		Token:          &oauth2.Token{AccessToken: encryptedToken},
		Settings:       &UserSettings{},
	}
	userInfoBytes, err := json.Marshal(userInfo)
	require.NoError(t, err)

	mockKvStore.EXPECT().ListKeys(0, keysPerPage, gomock.Any()).Return([]string{"user1" + githubTokenKey}, nil)

	mockKvStore.EXPECT().Get("user1"+githubTokenKey, gomock.Any()).DoAndReturn(
		func(key string, out any) error {
			return json.Unmarshal(userInfoBytes, out)
		},
	)

	mockKvStore.EXPECT().Set("user1"+githubTokenKey, gomock.Any()).DoAndReturn(
		func(key string, value any, opts ...pluginapi.KVSetOption) (bool, error) {
			storedInfo, ok := value.(*GitHubUserInfo)
			require.True(t, ok, "expected *GitHubUserInfo")
			decrypted, decErr := decrypt([]byte(testNewKey), storedInfo.Token.AccessToken)
			require.NoError(t, decErr)
			require.Equal(t, MockAccessToken, decrypted)
			return true, nil
		},
	)

	api.On("LogInfo", "Encryption key changed, re-encrypting user tokens",
		"user_count", "1").Times(1)

	p.reEncryptUserData(testNewKey, testOldKey)

	api.AssertExpectations(t)
}

func TestReEncryptUserData_DecryptFailure(t *testing.T) {
	p, api, mockKvStore, ctrl := setupRotationTest(t)
	defer ctrl.Finish()

	userInfo := &GitHubUserInfo{
		UserID:         "user1",
		GitHubUsername: "ghuser1",
		Token:          &oauth2.Token{AccessToken: "not-valid-base64-ciphertext!@#$"},
		Settings:       &UserSettings{},
	}
	userInfoBytes, err := json.Marshal(userInfo)
	require.NoError(t, err)

	mockKvStore.EXPECT().ListKeys(0, keysPerPage, gomock.Any()).Return([]string{"user1" + githubTokenKey}, nil)

	mockKvStore.EXPECT().Get("user1"+githubTokenKey, gomock.Any()).DoAndReturn(
		func(key string, out any) error {
			return json.Unmarshal(userInfoBytes, out)
		},
	)

	// forceDisconnectUser expectations
	mockKvStore.EXPECT().Delete("user1" + githubTokenKey).Return(nil)
	mockKvStore.EXPECT().Delete("user1" + githubPrivateRepoKey).Return(nil)
	mockKvStore.EXPECT().Delete("ghuser1" + githubUsernameKey).Return(nil)

	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("GetUser", "user1").Return(&model.User{
		Id:    "user1",
		Props: model.StringMap{},
	}, nil)
	api.On("GetDirectChannel", "user1", MockBotID).Return(&model.Channel{Id: "dmchannel"}, nil)
	api.On("CreatePost", mock.Anything).Return(&model.Post{}, nil)
	api.On("PublishWebSocketEvent", wsEventDisconnect, map[string]any(nil),
		&model.WebsocketBroadcast{UserId: "user1"}).Times(1)

	p.reEncryptUserData(testNewKey, testOldKey)

	api.AssertExpectations(t)
}

func TestReEncryptUserData_StoreFailure(t *testing.T) {
	p, api, mockKvStore, ctrl := setupRotationTest(t)
	defer ctrl.Finish()

	encryptedToken, err := encrypt([]byte(testOldKey), MockAccessToken)
	require.NoError(t, err)

	userInfo := &GitHubUserInfo{
		UserID:         "user1",
		GitHubUsername: "ghuser1",
		Token:          &oauth2.Token{AccessToken: encryptedToken},
		Settings:       &UserSettings{},
	}
	userInfoBytes, err := json.Marshal(userInfo)
	require.NoError(t, err)

	mockKvStore.EXPECT().ListKeys(0, keysPerPage, gomock.Any()).Return([]string{"user1" + githubTokenKey}, nil)

	mockKvStore.EXPECT().Get("user1"+githubTokenKey, gomock.Any()).DoAndReturn(
		func(key string, out any) error {
			return json.Unmarshal(userInfoBytes, out)
		},
	)

	// storeGitHubUserInfo fails
	mockKvStore.EXPECT().Set("user1"+githubTokenKey, gomock.Any()).Return(false, errors.New("KV store write error"))

	// forceDisconnectUser expectations
	mockKvStore.EXPECT().Delete("user1" + githubTokenKey).Return(nil)
	mockKvStore.EXPECT().Delete("user1" + githubPrivateRepoKey).Return(nil)
	mockKvStore.EXPECT().Delete("ghuser1" + githubUsernameKey).Return(nil)

	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("GetUser", "user1").Return(&model.User{
		Id:    "user1",
		Props: model.StringMap{},
	}, nil)
	api.On("GetDirectChannel", "user1", MockBotID).Return(&model.Channel{Id: "dmchannel"}, nil)
	api.On("CreatePost", mock.Anything).Return(&model.Post{}, nil)
	api.On("PublishWebSocketEvent", wsEventDisconnect, map[string]any(nil),
		&model.WebsocketBroadcast{UserId: "user1"}).Times(1)

	p.reEncryptUserData(testNewKey, testOldKey)

	api.AssertExpectations(t)
}

func TestReEncryptUserData_NoConnectedUsers(t *testing.T) {
	p, _, mockKvStore, ctrl := setupRotationTest(t)
	defer ctrl.Finish()

	mockKvStore.EXPECT().ListKeys(0, keysPerPage, gomock.Any()).Return([]string{}, nil)

	p.reEncryptUserData(testNewKey, testOldKey)
}

func TestReEncryptUserData_ListKeysError(t *testing.T) {
	p, api, mockKvStore, ctrl := setupRotationTest(t)
	defer ctrl.Finish()

	mockKvStore.EXPECT().ListKeys(0, keysPerPage, gomock.Any()).Return(nil, errors.New("KV list error"))

	api.On("LogWarn", "Encryption key changed but failed to list user keys for re-encryption, proceeding with keys collected so far",
		"page", "0", "keys_collected", "0", "error", "KV list error").Times(1)

	p.reEncryptUserData(testNewKey, testOldKey)

	api.AssertExpectations(t)
}

func TestReEncryptUserData_MultipleUsers(t *testing.T) {
	p, api, mockKvStore, ctrl := setupRotationTest(t)
	defer ctrl.Finish()

	enc1, err := encrypt([]byte(testOldKey), "token_user1")
	require.NoError(t, err)
	enc2, err := encrypt([]byte(testOldKey), "token_user2")
	require.NoError(t, err)

	user1 := &GitHubUserInfo{
		UserID:         "user1",
		GitHubUsername: "ghuser1",
		Token:          &oauth2.Token{AccessToken: enc1},
		Settings:       &UserSettings{},
	}
	user2 := &GitHubUserInfo{
		UserID:         "user2",
		GitHubUsername: "ghuser2",
		Token:          &oauth2.Token{AccessToken: enc2},
		Settings:       &UserSettings{},
	}
	u1bytes, _ := json.Marshal(user1)
	u2bytes, _ := json.Marshal(user2)

	mockKvStore.EXPECT().ListKeys(0, keysPerPage, gomock.Any()).Return(
		[]string{"user1" + githubTokenKey, "user2" + githubTokenKey}, nil)

	mockKvStore.EXPECT().Get("user1"+githubTokenKey, gomock.Any()).DoAndReturn(
		func(key string, out any) error { return json.Unmarshal(u1bytes, out) },
	)
	mockKvStore.EXPECT().Get("user2"+githubTokenKey, gomock.Any()).DoAndReturn(
		func(key string, out any) error { return json.Unmarshal(u2bytes, out) },
	)

	mockKvStore.EXPECT().Set("user1"+githubTokenKey, gomock.Any()).Return(true, nil)
	mockKvStore.EXPECT().Set("user2"+githubTokenKey, gomock.Any()).Return(true, nil)

	api.On("LogInfo", "Encryption key changed, re-encrypting user tokens",
		"user_count", "2").Times(1)

	p.reEncryptUserData(testNewKey, testOldKey)

	api.AssertExpectations(t)
}

func TestReEncryptUserData_AlreadyMigratedToken(t *testing.T) {
	p, api, mockKvStore, ctrl := setupRotationTest(t)
	defer ctrl.Finish()

	encryptedWithNewKey, err := encrypt([]byte(testNewKey), MockAccessToken)
	require.NoError(t, err)

	userInfo := &GitHubUserInfo{
		UserID:         "user1",
		GitHubUsername: "ghuser1",
		Token:          &oauth2.Token{AccessToken: encryptedWithNewKey},
		Settings:       &UserSettings{},
	}
	userInfoBytes, err := json.Marshal(userInfo)
	require.NoError(t, err)

	mockKvStore.EXPECT().ListKeys(0, keysPerPage, gomock.Any()).Return([]string{"user1" + githubTokenKey}, nil)

	mockKvStore.EXPECT().Get("user1"+githubTokenKey, gomock.Any()).DoAndReturn(
		func(key string, out any) error {
			return json.Unmarshal(userInfoBytes, out)
		},
	)

	api.On("LogInfo", "Encryption key changed, re-encrypting user tokens",
		"user_count", "1").Times(1)

	p.reEncryptUserData(testNewKey, testOldKey)

	api.AssertExpectations(t)
}

func TestForceDisconnectUser_CleansUpAndNotifies(t *testing.T) {
	p, api, mockKvStore, ctrl := setupRotationTest(t)
	defer ctrl.Finish()

	mockKvStore.EXPECT().Delete("user1" + githubTokenKey).Return(nil)
	mockKvStore.EXPECT().Delete("user1" + githubPrivateRepoKey).Return(nil)
	mockKvStore.EXPECT().Delete("ghuser1" + githubUsernameKey).Return(nil)

	api.On("GetUser", "user1").Return(&model.User{
		Id:    "user1",
		Props: model.StringMap{"git_user": "ghuser1"},
	}, nil)
	api.On("UpdateUser", mock.MatchedBy(func(u *model.User) bool {
		_, hasGitUser := u.Props["git_user"]
		return u.Id == "user1" && !hasGitUser
	})).Return(&model.User{Id: "user1", Props: model.StringMap{}}, nil)
	api.On("PublishWebSocketEvent", wsEventDisconnect, map[string]any(nil),
		&model.WebsocketBroadcast{UserId: "user1"}).Times(1)
	api.On("GetDirectChannel", "user1", MockBotID).Return(&model.Channel{Id: "dmchannel"}, nil)
	api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
		return post.UserId == MockBotID &&
			post.ChannelId == "dmchannel" &&
			post.Type == "custom_git_disconnect"
	})).Return(&model.Post{}, nil)

	p.forceDisconnectUser("user1", "ghuser1")

	api.AssertExpectations(t)
}

func TestForceDisconnectUser_NoGitHubUsername_FallbackFromProps(t *testing.T) {
	p, api, mockKvStore, ctrl := setupRotationTest(t)
	defer ctrl.Finish()

	mockKvStore.EXPECT().Delete("user1" + githubTokenKey).Return(nil)
	mockKvStore.EXPECT().Delete("user1" + githubPrivateRepoKey).Return(nil)
	// Username recovered from user props, so the mapping delete should happen
	mockKvStore.EXPECT().Delete("ghuser1" + githubUsernameKey).Return(nil)

	api.On("GetUser", "user1").Return(&model.User{
		Id:    "user1",
		Props: model.StringMap{"git_user": "ghuser1"},
	}, nil)
	api.On("UpdateUser", mock.Anything).Return(&model.User{Id: "user1", Props: model.StringMap{}}, nil)
	api.On("PublishWebSocketEvent", wsEventDisconnect, map[string]any(nil),
		&model.WebsocketBroadcast{UserId: "user1"}).Times(1)
	api.On("GetDirectChannel", "user1", MockBotID).Return(&model.Channel{Id: "dmchannel"}, nil)
	api.On("CreatePost", mock.Anything).Return(&model.Post{}, nil)

	p.forceDisconnectUser("user1", "")

	api.AssertExpectations(t)
}

func TestForceDisconnectUser_NoGitHubUsername_NoPropsFallback(t *testing.T) {
	p, api, mockKvStore, ctrl := setupRotationTest(t)
	defer ctrl.Finish()

	mockKvStore.EXPECT().Delete("user1" + githubTokenKey).Return(nil)
	mockKvStore.EXPECT().Delete("user1" + githubPrivateRepoKey).Return(nil)
	// No username available anywhere, so no mapping delete

	api.On("GetUser", "user1").Return(&model.User{
		Id:    "user1",
		Props: model.StringMap{},
	}, nil)
	api.On("PublishWebSocketEvent", wsEventDisconnect, map[string]any(nil),
		&model.WebsocketBroadcast{UserId: "user1"}).Times(1)
	api.On("GetDirectChannel", "user1", MockBotID).Return(&model.Channel{Id: "dmchannel"}, nil)
	api.On("CreatePost", mock.Anything).Return(&model.Post{}, nil)

	p.forceDisconnectUser("user1", "")

	api.AssertExpectations(t)
}

func TestForceDisconnectUser_DeleteErrors(t *testing.T) {
	p, api, mockKvStore, ctrl := setupRotationTest(t)
	defer ctrl.Finish()

	mockKvStore.EXPECT().Delete("user1" + githubTokenKey).Return(errors.New("delete failed"))
	mockKvStore.EXPECT().Delete("user1" + githubPrivateRepoKey).Return(errors.New("delete failed"))
	mockKvStore.EXPECT().Delete("ghuser1" + githubUsernameKey).Return(errors.New("delete failed"))

	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("GetUser", "user1").Return(nil, &model.AppError{Message: "user not found"})
	api.On("PublishWebSocketEvent", wsEventDisconnect, map[string]any(nil),
		&model.WebsocketBroadcast{UserId: "user1"}).Times(1)
	api.On("GetDirectChannel", "user1", MockBotID).Return(&model.Channel{Id: "dmchannel"}, nil)
	api.On("CreatePost", mock.Anything).Return(&model.Post{}, nil)

	p.forceDisconnectUser("user1", "ghuser1")

	api.AssertExpectations(t)
}

func TestTruncatePostMessage(t *testing.T) {
	t.Run("short message is returned unchanged", func(t *testing.T) {
		msg := "hello world"
		require.Equal(t, msg, truncatePostMessage(msg))
	})

	t.Run("message at the rune limit is returned unchanged", func(t *testing.T) {
		msg := strings.Repeat("a", model.PostMessageMaxRunesV2)
		require.Equal(t, msg, truncatePostMessage(msg))
	})

	t.Run("oversized ASCII message is truncated and marker appended", func(t *testing.T) {
		msg := strings.Repeat("a", model.PostMessageMaxRunesV2+5_000)
		out := truncatePostMessage(msg)

		require.LessOrEqual(t, utf8.RuneCountInString(out), model.PostMessageMaxRunesV2)
		require.True(t, strings.HasSuffix(out, "_… message truncated_"))
	})

	t.Run("oversized multibyte message keeps rune boundaries", func(t *testing.T) {
		// Each rune is 3 bytes, so byte-based truncation would corrupt the output.
		msg := strings.Repeat("✓", model.PostMessageMaxRunesV2+1_000)
		out := truncatePostMessage(msg)

		require.LessOrEqual(t, utf8.RuneCountInString(out), model.PostMessageMaxRunesV2)
		require.True(t, utf8.ValidString(out), "truncated output should remain valid UTF-8")
		require.True(t, strings.HasSuffix(out, "_… message truncated_"))
	})
}

func TestIsGitHubAuthFailure(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		require.False(t, isGitHubAuthFailure(nil))
	})

	t.Run("401 bad credentials string", func(t *testing.T) {
		require.True(t, isGitHubAuthFailure(errors.New(invalidTokenError)))
	})

	t.Run("401 from ErrorResponse", func(t *testing.T) {
		err := &github.ErrorResponse{Response: &http.Response{StatusCode: http.StatusUnauthorized}}
		require.True(t, isGitHubAuthFailure(err))
	})

	t.Run("401 from graphql client message", func(t *testing.T) {
		err := errors.New("non-200 OK status code: 401 Unauthorized")
		require.True(t, isGitHubAuthFailure(err))
	})

	t.Run("403 SAML from ErrorResponse", func(t *testing.T) {
		err := &github.ErrorResponse{
			Message: "Resource protected by organization SAML enforcement. You must grant your OAuth token access to this organization.",
			Response: &http.Response{
				StatusCode: http.StatusForbidden,
				Header:     http.Header{"X-Github-Sso": []string{"required; url=https://github.com/orgs/foo/sso"}},
			},
		}
		require.True(t, isGitHubAuthFailure(err))
	})

	t.Run("403 SAML from graphql error string", func(t *testing.T) {
		err := errors.New("error in executing query: GraphQL: Resource protected by organization SAML enforcement. You must grant your OAuth token access to this organization.")
		require.True(t, isGitHubAuthFailure(err))
	})

	t.Run("403 unrelated", func(t *testing.T) {
		err := &github.ErrorResponse{
			Message:  "Forbidden",
			Response: &http.Response{StatusCode: http.StatusForbidden},
		}
		require.False(t, isGitHubAuthFailure(err))
	})

	t.Run("generic error", func(t *testing.T) {
		require.False(t, isGitHubAuthFailure(errors.New("connection reset")))
	})
}

func connectedGitHubUserInfo(t *testing.T) *GitHubUserInfo {
	t.Helper()
	encryptedToken, err := encrypt([]byte(testNewKey), MockAccessToken)
	require.NoError(t, err)
	return &GitHubUserInfo{
		UserID:         "user1",
		GitHubUsername: "ghuser1",
		Token:          &oauth2.Token{AccessToken: encryptedToken},
		Settings:       &UserSettings{},
	}
}

func expectRevokedTokenNotification(api *plugintest.API, mockKvStore *mocks.MockKvStore, userInfo *GitHubUserInfo) {
	mockKvStore.EXPECT().Get(userInfo.UserID+githubTokenKey, gomock.Any()).DoAndReturn(
		func(_ string, out any) error {
			userInfoBytes, err := json.Marshal(userInfo)
			if err != nil {
				return err
			}
			return json.Unmarshal(userInfoBytes, out)
		},
	)
	mockKvStore.EXPECT().Delete(userInfo.UserID + githubTokenKey).Return(nil)
	mockKvStore.EXPECT().Delete(userInfo.GitHubUsername + githubUsernameKey).Return(nil)
	mockKvStore.EXPECT().Delete(userInfo.UserID + githubPrivateRepoKey).Return(nil)
	api.On("GetUser", userInfo.UserID).Return(&model.User{
		Id:    userInfo.UserID,
		Props: model.StringMap{"git_user": userInfo.GitHubUsername},
	}, nil)
	api.On("UpdateUser", mock.Anything).Return(&model.User{Id: userInfo.UserID, Props: model.StringMap{}}, nil)
	api.On("PublishWebSocketEvent", wsEventDisconnect, map[string]any(nil),
		&model.WebsocketBroadcast{UserId: userInfo.UserID}).Return()
	api.On("GetDirectChannel", userInfo.UserID, MockBotID).Return(&model.Channel{Id: "dmchannel"}, nil)
	api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
		return post.UserId == MockBotID &&
			post.ChannelId == "dmchannel" &&
			post.Type == "custom_git_revoked_token"
	})).Return(&model.Post{}, nil).Once()
}

func TestUseGitHubClient_AuthFailureNotifiesUser(t *testing.T) {
	samlGraphQLErr := errors.New("error in executing query: GraphQL: Resource protected by organization SAML enforcement. You must grant your OAuth token access to this organization.")

	tests := []struct {
		name   string
		err    error
		notify bool
	}{
		{
			name:   "401 bad credentials",
			err:    errors.New(invalidTokenError),
			notify: true,
		},
		{
			name: "403 SAML REST",
			err: &github.ErrorResponse{
				Message: "Resource protected by organization SAML enforcement. You must grant your OAuth token access to this organization.",
				Response: &http.Response{
					StatusCode: http.StatusForbidden,
					Header:     http.Header{"X-Github-Sso": []string{"required; url=https://github.com/orgs/foo/sso"}},
					Request:    &http.Request{},
				},
			},
			notify: true,
		},
		{
			name:   "403 SAML graphql",
			err:    samlGraphQLErr,
			notify: true,
		},
		{
			name: "403 unrelated",
			err: &github.ErrorResponse{
				Message:  "Forbidden",
				Response: &http.Response{StatusCode: http.StatusForbidden, Request: &http.Request{}},
			},
			notify: false,
		},
		{
			name:   "generic error",
			err:    errors.New("connection reset"),
			notify: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, api, mockKvStore, ctrl := setupRotationTest(t)
			defer ctrl.Finish()

			userInfo := connectedGitHubUserInfo(t)
			if tc.notify {
				expectRevokedTokenNotification(api, mockKvStore, userInfo)
			}

			err := p.useGitHubClient(userInfo, func(_ *GitHubUserInfo, _ *oauth2.Token) error {
				return tc.err
			})
			require.Equal(t, tc.err, err)
			api.AssertExpectations(t)
		})
	}
}
