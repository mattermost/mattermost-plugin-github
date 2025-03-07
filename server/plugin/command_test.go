// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-github/server/mocks"
)

// Function to get the plugin object for test cases.
func getPluginTest(api *plugintest.API, mockKvStore *mocks.MockKvStore) *Plugin {
	p := NewPlugin()
	p.setConfiguration(
		&Configuration{
			GitHubOrg:               "mockOrg",
			GitHubOAuthClientID:     "mockID",
			GitHubOAuthClientSecret: "mockSecret",
			EncryptionKey:           "mockKey123456789",
		})
	p.initializeAPI()

	p.store = mockKvStore

	p.BotUserID = "mockBotID"

	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)

	return p
}

func TestValidateFeatures(t *testing.T) {
	type output struct {
		valid           bool
		invalidFeatures []string
	}
	tests := []struct {
		name string
		args []string
		want output
	}{
		{
			name: "all features valid",
			args: []string{"creates", "pushes", "issue_comments"},
			want: output{true, []string{}},
		},
		{
			name: "all features invalid",
			args: []string{"create", "push"},
			want: output{false, []string{"create", "push"}},
		},
		{
			name: "first feature invalid",
			args: []string{"create", "pushes", "issue_comments"},
			want: output{false, []string{"create"}},
		},
		{
			name: "last feature invalid",
			args: []string{"creates", "push"},
			want: output{false, []string{"push"}},
		},
		{
			name: "multiple features invalid",
			args: []string{"create", "pushes", "issue"},
			want: output{false, []string{"create", "issue"}},
		},
		{
			name: "all features valid with label but issues and pulls missing",
			args: []string{"pushes", `label:"ruby"`},
			want: output{false, []string{}},
		},
		{
			name: "all features valid with label and issues in features",
			args: []string{"issues", `label:"ruby"`},
			want: output{true, []string{}},
		},
		{
			name: "all features valid with label and pulls in features",
			args: []string{"pulls", `label:"ruby"`},
			want: output{true, []string{}},
		},
		{
			name: "multiple features invalid with label but issues and pulls missing",
			args: []string{"issue", "push", `label:"ruby"`},
			want: output{false, []string{"issue", "push"}},
		},
		{
			name: "multiple features invalid with label and issues in features",
			args: []string{"issues", "push", "create", `label:"ruby"`},
			want: output{false, []string{"push", "create"}},
		},
		{
			name: "multiple features invalid with label and pulls in features",
			args: []string{"pulls", "push", "create", `label:"ruby"`},
			want: output{false, []string{"push", "create"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, fs := validateFeatures(tt.args)
			got := output{ok, fs}
			testFailureMessage := fmt.Sprintf("validateFeatures() = %v, want %v", got, tt.want)
			assert.EqualValues(t, tt.want, got, testFailureMessage)
		})
	}
}

func TestParseCommand(t *testing.T) {
	type output struct {
		command    string
		action     string
		parameters []string
	}

	tt := []struct {
		name  string
		input string
		want  output
	}{
		{
			name:  "no parameters",
			input: "/github subscribe",
			want: output{
				"/github",
				"subscribe",
				[]string(nil),
			},
		},
		{
			name:  "no action and no parameters",
			input: "/github",
			want: output{
				"/github",
				"",
				[]string(nil),
			},
		},
		{
			name:  "simple one-word label",
			input: `/github subscribe DHaussermann/hello-world issues,label:"Help"`,
			want: output{
				"/github",
				"subscribe",
				[]string{"DHaussermann/hello-world", `issues,label:"Help"`},
			},
		},
		{
			name:  "two-word label",
			input: `/github subscribe DHaussermann/hello-world issues,label:"Help Wanted"`,
			want: output{
				"/github",
				"subscribe",
				[]string{"DHaussermann/hello-world", `issues,label:"Help Wanted"`},
			},
		},
		{
			name:  "multi-word label",
			input: `/github subscribe DHaussermann/hello-world issues,label:"Good First Issue"`,
			want: output{
				"/github",
				"subscribe",
				[]string{"DHaussermann/hello-world", `issues,label:"Good First Issue"`},
			},
		},
		{
			name:  "multiple spaces inside double-quotes",
			input: `/github subscribe DHaussermann/hello-world issues,label:"Help    Wanted"`,
			want: output{
				"/github",
				"subscribe",
				[]string{"DHaussermann/hello-world", `issues,label:"Help    Wanted"`},
			},
		},
		{
			name:  "multiple spaces outside of double-quotes",
			input: `  /github    subscribe     DHaussermann/hello-world issues,label:"Help Wanted"`,
			want: output{
				"/github",
				"subscribe",
				[]string{"DHaussermann/hello-world", `issues,label:"Help Wanted"`},
			},
		},
		{
			name:  "trailing whitespaces",
			input: `/github subscribe DHaussermann/hello-world issues,label:"Help Wanted" `,
			want: output{
				"/github",
				"subscribe",
				[]string{"DHaussermann/hello-world", `issues,label:"Help Wanted"`},
			},
		},
		{
			name:  "non-ASCII characters",
			input: `/github subscribe طماطم issues,label:"日本語"`,
			want: output{
				"/github",
				"subscribe",
				[]string{"طماطم", `issues,label:"日本語"`},
			},
		},
		{
			name:  "line breaks",
			input: "/github \nsubscribe\nDHaussermann/hello-world\nissues,label:\"Good First Issue\"",
			want: output{
				"/github",
				"subscribe",
				[]string{"DHaussermann/hello-world", `issues,label:"Good First Issue"`},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			command, action, parameters := parseCommand(tc.input)
			got := output{command, action, parameters}
			testFailureMessage := fmt.Sprintf("validateFeatures() = %v, want %v", got, tc.want)
			assert.EqualValues(t, tc.want, got, testFailureMessage)
		})
	}
}

func TestCheckConflictingFeatures(t *testing.T) {
	type output struct {
		valid               bool
		conflictingFeatures []string
	}
	tests := []struct {
		name string
		args []string
		want output
	}{
		{
			name: "no conflicts",
			args: []string{"creates", "pushes", "issue_comments"},
			want: output{true, nil},
		},
		{
			name: "conflict with issue and issue creation",
			args: []string{"pulls", "issues", "issue_creations"},
			want: output{false, []string{"issues", "issue_creations"}},
		},
		{
			name: "conflict with pulls and pulls created",
			args: []string{"pulls", "issues", "pulls_created"},
			want: output{false, []string{"pulls", "pulls_created"}},
		},
		{
			name: "conflict with pulls and pulls merged",
			args: []string{"pulls", "pushes", "pulls_merged"},
			want: output{false, []string{"pulls", "pulls_merged"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, fs := checkFeatureConflict(tt.args)
			got := output{ok, fs}
			testFailureMessage := fmt.Sprintf("checkFeatureConflict() = %v, want %v", got, tt.want)
			assert.EqualValues(t, tt.want, got, testFailureMessage)
		})
	}
}

func TestExecuteCommand(t *testing.T) {
	tests := map[string]struct {
		commandArgs    *model.CommandArgs
		expectedMsg    string
		SetupMockStore func(*mocks.MockKvStore)
	}{
		"about command": {
			commandArgs:    &model.CommandArgs{Command: "/github about"},
			expectedMsg:    "GitHub version",
			SetupMockStore: func(mks *mocks.MockKvStore) {},
		},

		"help command": {
			commandArgs: &model.CommandArgs{Command: "/github help", ChannelId: "test-channelID", RootId: "test-rootID", UserId: "test-userID"},
			expectedMsg: "###### Mattermost GitHub Plugin - Slash Command Help\n",
			SetupMockStore: func(mks *mocks.MockKvStore) {
				mks.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					// Cast the value to the appropriate type and updated it
					if userInfoPtr, ok := value.(**GitHubUserInfo); ok {
						*userInfoPtr = &GitHubUserInfo{
							// Mock user info data
							Token: &oauth2.Token{
								AccessToken: "ycbODW-BWbNBGfF7ac4T5RL5ruNm5BChCXgbkY1bWHqMt80JTkLsicQwo8de3tqfqlfMaglpgjqGOmSHeGp0dA==",
							},
						}
					}
					return nil // no error, so return nil
				})
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			isSendEphemeralPostCalled := false

			// Controller for the mocks generated using mockgen
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockKvStore := mocks.NewMockKvStore(mockCtrl)

			tt.SetupMockStore(mockKvStore)

			currentTestAPI := &plugintest.API{}
			currentTestAPI.On("SendEphemeralPost", mock.AnythingOfType("string"), mock.AnythingOfType("*model.Post")).Run(func(args mock.Arguments) {
				isSendEphemeralPostCalled = true

				post := args.Get(1).(*model.Post)
				// Checking the contents of the post
				assert.Contains(t, post.Message, tt.expectedMsg)
			}).Once().Return(&model.Post{})

			p := getPluginTest(currentTestAPI, mockKvStore)

			_, err := p.ExecuteCommand(&plugin.Context{}, tt.commandArgs)
			require.Nil(t, err)

			assert.Equal(t, true, isSendEphemeralPostCalled)
		})
	}
}

func TestGetMutedUsernames(t *testing.T) {
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)
	userInfo, err := GetMockGHUserInfo(p)
	assert.NoError(t, err)

	tests := []struct {
		name       string
		setup      func()
		assertions func(t *testing.T, result []string, err error)
	}{
		{
			name: "Error retrieving muted usernames",
			setup: func() {
				mockKvStore.EXPECT().Get("mockUserID-muted-users", gomock.Any()).Return(errors.New("error retrieving muted users")).Times(1)
			},
			assertions: func(t *testing.T, result []string, err error) {
				assert.Nil(t, result)
				assert.ErrorContains(t, err, "error retrieving muted users")
			},
		},
		{
			name: "No muted usernames set for user",
			setup: func() {
				mockKvStore.EXPECT().Get("mockUserID-muted-users", gomock.Any()).DoAndReturn(func(key string, value *[]byte) error {
					*value = []byte("")
					return nil
				}).Times(1)
			},
			assertions: func(t *testing.T, result []string, _ error) {
				assert.Equal(t, []string(nil), result)
			},
		},
		{
			name: "Successfully retrieves muted usernames",
			setup: func() {
				mutedUsernames := []byte("user1,user2,user3")
				mockKvStore.EXPECT().Get("mockUserID-muted-users", gomock.Any()).DoAndReturn(func(key string, value *[]byte) error {
					*value = mutedUsernames
					return nil
				}).Times(1)
			},
			assertions: func(t *testing.T, result []string, _ error) {
				assert.Equal(t, []string{"user1", "user2", "user3"}, result)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			mutedUsernames, err := p.getMutedUsernames(userInfo)

			tc.assertions(t, mutedUsernames, err)
		})
	}
}

func TestHandleMuteList(t *testing.T) {
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)
	userInfo, err := GetMockGHUserInfo(p)
	assert.NoError(t, err)

	tests := []struct {
		name       string
		setup      func()
		assertions func(t *testing.T, result string)
	}{
		{
			name: "Error retrieving muted usernames",
			setup: func() {
				mockAPI.On("LogError", "error occurred getting muted users.", "UserID", userInfo.UserID, "Error", mock.Anything)
				mockKvStore.EXPECT().Get("mockUserID-muted-users", gomock.Any()).Return(errors.New("error retrieving muted users")).Times(1)
			},
			assertions: func(t *testing.T, result string) {
				assert.Equal(t, "An error occurred getting muted users. Please try again later", result)
			},
		},
		{
			name: "No muted usernames set for user",
			setup: func() {
				mockKvStore.EXPECT().Get("mockUserID-muted-users", gomock.Any()).DoAndReturn(func(key string, value *[]byte) error {
					*value = []byte("")
					return nil
				}).Times(1)
			},
			assertions: func(t *testing.T, result string) {
				assert.Equal(t, "You have no muted users", result)
			},
		},
		{
			name: "Successfully retrieves and formats muted usernames",
			setup: func() {
				mutedUsernames := []byte("user1,user2,user3")
				mockKvStore.EXPECT().Get("mockUserID-muted-users", gomock.Any()).DoAndReturn(func(key string, value *[]byte) error {
					*value = mutedUsernames
					return nil
				}).Times(1)
			},
			assertions: func(t *testing.T, result string) {
				expectedOutput := "Your muted users:\n- user1\n- user2\n- user3\n"
				assert.Equal(t, expectedOutput, result)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			result := p.handleMuteList(nil, userInfo)

			tc.assertions(t, result)
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name       string
		slice      []string
		element    string
		assertions func(t *testing.T, result bool)
	}{
		{
			name:    "Element is present in slice",
			slice:   []string{"expectedElement1", "expectedElement2", "expectedElement3"},
			element: "expectedElement2",
			assertions: func(t *testing.T, result bool) {
				assert.True(t, result)
			},
		},
		{
			name:    "Element is not present in slice",
			slice:   []string{"expectedElement1", "expectedElement2", "expectedElement3"},
			element: "expectedElement4",
			assertions: func(t *testing.T, result bool) {
				assert.False(t, result)
			},
		},
		{
			name:    "Empty slice",
			slice:   []string{},
			element: "expectedElement1",
			assertions: func(t *testing.T, result bool) {
				assert.False(t, result)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := contains(tc.slice, tc.element)
			tc.assertions(t, result)
		})
	}
}

func TestHandleMuteAdd(t *testing.T) {
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)
	userInfo, err := GetMockGHUserInfo(p)
	assert.NoError(t, err)

	tests := []struct {
		name       string
		username   string
		setup      func()
		assertions func(t *testing.T, result string)
	}{
		{
			name: "Error retrieving muted usernames",
			setup: func() {
				mockAPI.On("LogError", "error occurred getting muted users.", "UserID", userInfo.UserID, "Error", mock.Anything)
				mockKvStore.EXPECT().Get("mockUserID-muted-users", gomock.Any()).Return(errors.New("error retrieving muted users")).Times(1)
			},
			assertions: func(t *testing.T, result string) {
				assert.Equal(t, "An error occurred getting muted users. Please try again later", result)
			},
		},
		{
			name:     "Error saving the new muted username",
			username: "errorUser",
			setup: func() {
				mockKvStore.EXPECT().Get(userInfo.UserID+"-muted-users", gomock.Any()).DoAndReturn(func(key string, value *[]byte) error {
					*value = []byte("existingUser")
					return nil
				}).Times(1)
				mockKvStore.EXPECT().Set(userInfo.UserID+"-muted-users", []byte("existingUser,errorUser")).Return(false, errors.New("store error")).Times(1)
			},
			assertions: func(t *testing.T, result string) {
				assert.Equal(t, "Error occurred saving list of muted users", result)
			},
		},
		{
			name:     "Username is already muted",
			username: "alreadyMutedUser",
			setup: func() {
				mockKvStore.EXPECT().Get(userInfo.UserID+"-muted-users", gomock.Any()).DoAndReturn(func(key string, value *[]byte) error {
					*value = []byte("alreadyMutedUser")
					return nil
				}).Times(1)
			},
			assertions: func(t *testing.T, result string) {
				assert.Equal(t, "alreadyMutedUser is already muted", result)
			},
		},
		{
			name:     "Invalid username with comma",
			username: "invalid,user",
			setup: func() {
				mockKvStore.EXPECT().Get(userInfo.UserID+"-muted-users", gomock.Any()).DoAndReturn(func(key string, value *[]byte) error {
					*value = []byte("")
					return nil
				}).Times(1)
			},
			assertions: func(t *testing.T, result string) {
				assert.Equal(t, "Invalid username provided", result)
			},
		},
		{
			name:     "Successfully adds new muted username",
			username: "newUser",
			setup: func() {
				mockKvStore.EXPECT().Get(userInfo.UserID+"-muted-users", gomock.Any()).DoAndReturn(func(key string, value *[]byte) error {
					*value = []byte("existingUser")
					return nil
				}).Times(1)
				mockKvStore.EXPECT().Set(userInfo.UserID+"-muted-users", []byte("existingUser,newUser")).Return(true, nil).Times(1)
			},
			assertions: func(t *testing.T, result string) {
				expectedMessage := "`newUser` is now muted. You'll no longer receive notifications for comments in your PRs and issues."
				assert.Equal(t, expectedMessage, result)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			result := p.handleMuteAdd(nil, tc.username, userInfo)
			tc.assertions(t, result)
		})
	}
}

func TestHandleUnmute(t *testing.T) {
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)
	userInfo, err := GetMockGHUserInfo(p)
	assert.NoError(t, err)

	tests := []struct {
		name           string
		username       string
		setup          func()
		expectedResult string
	}{
		{
			name: "Error retrieving muted usernames",
			setup: func() {
				mockAPI.On("LogError", "error occurred getting muted users.", "UserID", userInfo.UserID, "Error", mock.Anything)
				mockKvStore.EXPECT().Get("mockUserID-muted-users", gomock.Any()).Return(errors.New("error retrieving muted users")).Times(1)
			},
			expectedResult: "An error occurred getting muted users. Please try again later",
		},
		{
			name:     "Error occurred while unmuting the user",
			username: "user1",
			setup: func() {
				mutedUsernames := []byte("user1,user2,user3")
				mockKvStore.EXPECT().Get("mockUserID-muted-users", gomock.Any()).DoAndReturn(func(key string, value *[]byte) error {
					*value = mutedUsernames
					return nil
				}).Times(1)
				mockKvStore.EXPECT().Set(userInfo.UserID+"-muted-users", gomock.Any()).Return(false, errors.New("error saving muted users")).Times(1)
			},
			expectedResult: "Error occurred unmuting users",
		},
		{
			name:     "Successfully unmute a user",
			username: "user1",
			setup: func() {
				mutedUsernames := []byte("user1,user2,user3")
				mockKvStore.EXPECT().Get("mockUserID-muted-users", gomock.Any()).DoAndReturn(func(key string, value *[]byte) error {
					*value = mutedUsernames
					return nil
				}).Times(1)
				mockKvStore.EXPECT().Set(userInfo.UserID+"-muted-users", gomock.Any()).Return(true, nil).Times(1)
			},
			expectedResult: "`user1` is no longer muted",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			result := p.handleUnmute(nil, tc.username, userInfo)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestHandleUnmuteAll(t *testing.T) {
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)
	userInfo, err := GetMockGHUserInfo(p)
	assert.NoError(t, err)

	tests := []struct {
		name           string
		setup          func()
		assertions     func(string)
		expectedResult string
	}{
		{
			name: "Error occurred while unmuting all users",
			setup: func() {
				mockKvStore.EXPECT().Set(userInfo.UserID+"-muted-users", []byte("")).Return(false, errors.New("error saving muted users")).Times(1)
			},
			assertions: func(expectedResult string) {
				assert.Equal(t, expectedResult, "Error occurred unmuting users")
			},
		},
		{
			name: "Successfully unmute all users",
			setup: func() {
				mockKvStore.EXPECT().Set(userInfo.UserID+"-muted-users", []byte("")).Return(true, nil).Times(1)
			},
			assertions: func(expectedResult string) {
				assert.Equal(t, expectedResult, "Unmuted all users")
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			result := p.handleUnmuteAll(nil, userInfo)
			tc.assertions(result)
		})
	}
}
