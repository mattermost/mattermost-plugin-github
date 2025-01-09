package plugin

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
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
