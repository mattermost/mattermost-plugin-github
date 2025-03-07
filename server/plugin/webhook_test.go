// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest/mock"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"golang.org/x/oauth2"

	"github.com/google/go-github/v54/github"
	"github.com/stretchr/testify/assert"
)

const webhookSecret = "whsecret"
const orgMember = "org-member"
const orgCollaborator = "org-collaborator"
const gitHubOrginization = "test-org"

func newPlugin(userID string, gitHubURL string) *Plugin {
	p := NewPlugin()
	p.initializeAPI()
	p.SetDriver(&plugintest.Driver{})
	p.store = &pluginapi.MemoryStore{}
	token, _ := generateSecret()
	encryptionKey, _ := generateSecret()
	encryptedToken, _ := encrypt([]byte(encryptionKey), token)
	_, _ = p.store.Set(userID+githubTokenKey, GitHubUserInfo{
		UserID: userID,
		Token: &oauth2.Token{
			AccessToken: encryptedToken,
		},
	})
	p.setConfiguration(&Configuration{
		EncryptionKey:       encryptionKey,
		GitHubOrg:           gitHubOrginization,
		WebhookSecret:       webhookSecret,
		EnterpriseBaseURL:   gitHubURL,
		EnterpriseUploadURL: gitHubURL,
	})

	_ = p.AddSubscription(
		gitHubOrginization+"/test-repo",
		&Subscription{
			ChannelID: "1",
			CreatorID: userID,
			Features:  Features(strings.Join([]string{featureIssues, featureIssueCreation}, ",")),
			Flags:     SubscriptionFlags{IncludeOnlyOrgMembers: true},
		},
	)

	return p
}

func mockGitHubServer(user string) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != fmt.Sprintf("/api/v3/orgs/%v/members/%v", gitHubOrginization, user) {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		if user == orgMember {
			w.WriteHeader(http.StatusNoContent)
		} else if user == orgCollaborator {
			w.WriteHeader(http.StatusFound)
		}
	}))

	return ts
}

func TestIncludeOnlyOrgMembers(t *testing.T) {
	tests := []struct {
		name         string
		user         github.User
		subscription Subscription
		expectWarn   bool
		want         bool
	}{
		{
			name: "IncludeOnlyOrgMembers flag is false",
			user: github.User{
				Login: github.String(orgMember),
			},
			subscription: Subscription{
				Flags: SubscriptionFlags{IncludeOnlyOrgMembers: false},
			},
			expectWarn: false,
			want:       false,
		},
		{
			name: "Failed to get GitHub Client",
			user: github.User{
				Login: github.String(orgMember),
			},
			subscription: Subscription{
				CreatorID: model.NewId(),
				Flags:     SubscriptionFlags{IncludeOnlyOrgMembers: true},
			},
			expectWarn: true,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			user := *tt.user.Login
			server := mockGitHubServer(user)
			gitHubPlugin := newPlugin(model.NewId(), server.URL)
			api := plugintest.NewAPI(t)
			if tt.expectWarn {
				api.On("LogWarn", mock.AnythingOfType("string"), "error", mock.AnythingOfType("string")).Return(nil)
			}
			gitHubPlugin.SetAPI(api)
			gitHubPlugin.client = pluginapi.NewClient(gitHubPlugin.API, gitHubPlugin.Driver)

			got := gitHubPlugin.shouldDenyEventDueToNotOrgMember(&tt.user, &tt.subscription)
			assert.Equal(t, tt.want, got)
		})
	}
}
