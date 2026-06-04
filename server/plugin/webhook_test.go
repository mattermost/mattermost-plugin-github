package plugin

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost/server/public/plugin/plugintest"

	"github.com/mattermost/mattermost-plugin-github/server/mocks"
)

func TestIgnoreRequestedReview(t *testing.T) {
	tests := map[string]struct {
		event           *FPullRequestEvent
		requestedUserID string
		userInfo        *ForgejoUserInfo
		expected        bool
	}{
		"empty user ID": {
			event: &FPullRequestEvent{
				PullRequest: &FPullRequest{
					RequestedReviewersTeams: []*FTeam{},
				},
			},
			requestedUserID: "",
			expected:        false,
		},
		"no team reviewers": {
			event: &FPullRequestEvent{
				PullRequest: &FPullRequest{
					RequestedReviewersTeams: []*FTeam{},
				},
			},
			requestedUserID: "test-userID",
			expected:        false,
		},
		"user is individual reviewer": {
			event: &FPullRequestEvent{
				PullRequest: &FPullRequest{
					RequestedReviewersTeams: []*FTeam{{Name: stringPtr("team1")}},
					RequestedReviewers: []*FUser{
						{Login: stringPtr("test-user")},
					},
				},
				RequestedReviewer: &FUser{Login: stringPtr("test-user")},
			},
			requestedUserID: "test-userID",
			userInfo: &ForgejoUserInfo{
				Token: &oauth2.Token{
					AccessToken:  testToken,
					RefreshToken: testToken,
				},
				Settings: &UserSettings{
					DisableTeamNotifications: true,
				},
			},
			expected: false,
		},
		"team notifications disabled": {
			event: &FPullRequestEvent{
				PullRequest: &FPullRequest{
					RequestedReviewersTeams: []*FTeam{{Name: stringPtr("team1")}},
				},
			},
			requestedUserID: "test-userID",
			userInfo: &ForgejoUserInfo{
				Token: &oauth2.Token{
					AccessToken:  testToken,
					RefreshToken: testToken,
				},
				Settings: &UserSettings{
					DisableTeamNotifications: true,
				},
			},
			expected: true,
		},
		"repository excluded": {
			event: &FPullRequestEvent{
				PullRequest: &FPullRequest{
					RequestedReviewersTeams: []*FTeam{{Name: stringPtr("team1")}},
				},
				Repo: &FRepository{
					FullName: stringPtr("org/repo1"),
				},
			},
			requestedUserID: "test-userID",
			userInfo: &ForgejoUserInfo{
				Token: &oauth2.Token{
					AccessToken:  testToken,
					RefreshToken: testToken,
				},
				Settings: &UserSettings{
					DisableTeamNotifications:       false,
					ExcludeTeamReviewNotifications: []string{"org/repo1"},
				},
			},
			expected: true,
		},
		"repository not excluded": {
			event: &FPullRequestEvent{
				PullRequest: &FPullRequest{
					RequestedReviewersTeams: []*FTeam{{Name: stringPtr("team1")}},
				},
				Repo: &FRepository{
					FullName: stringPtr("org/repo1"),
				},
			},
			requestedUserID: "test-userID",
			userInfo: &ForgejoUserInfo{
				Token: &oauth2.Token{
					AccessToken:  testToken,
					RefreshToken: testToken,
				},
				Settings: &UserSettings{
					DisableTeamNotifications:       false,
					ExcludeTeamReviewNotifications: []string{"org/other-repo"},
				},
			},
			expected: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockKvStore := mocks.NewMockKvStore(mockCtrl)
			currentTestAPI := &plugintest.API{}
			p := getPluginTest(currentTestAPI, mockKvStore)

			// Mock getGitHubUserInfo if userInfo is provided
			if tt.userInfo != nil {
				mockKvStore.EXPECT().
					Get("test-userID"+forgejoTokenKey, gomock.Any()).
					DoAndReturn(func(key string, value interface{}) error {
						*(value.(**ForgejoUserInfo)) = tt.userInfo
						return nil
					}).
					AnyTimes()
			}

			result := p.ignoreRequestedReview(tt.event, tt.requestedUserID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
