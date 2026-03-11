// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
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
