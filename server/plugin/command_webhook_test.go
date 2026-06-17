// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-github/v54/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

const (
	webhookTestSiteURL     = "https://mattermost.example.com"
	webhookTestOwner       = "mockOrg"
	webhookTestRepo        = "mockRepo"
	webhookTestMatchingURL = webhookTestSiteURL + "/plugins/github/webhook"
	webhookTestOtherURL    = "https://example.org/some-other-hook"
)

// hooksResponse renders a GitHub "list hooks" API response whose webhooks target the given URLs.
func hooksResponse(urls ...string) string {
	hooks := make([]string, 0, len(urls))
	for i, u := range urls {
		hooks = append(hooks, fmt.Sprintf(`{"id":%d,"type":"Repository","name":"web","active":true,"config":{"url":%q,"content_type":"json"}}`, i+1, u))
	}
	return "[" + strings.Join(hooks, ",") + "]"
}

// jsonHandler responds with the given status code and body.
func jsonHandler(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = fmt.Fprint(w, body)
	}
}

// newHooksTestClient builds a github.Client pointed at a test server that serves the given
// handlers keyed by request path. Paths without a handler return 404.
func newHooksTestClient(t *testing.T, handlers map[string]http.HandlerFunc) (*github.Client, func()) {
	t.Helper()
	mux := http.NewServeMux()
	for path, handler := range handlers {
		mux.HandleFunc(path, handler)
	}
	server := httptest.NewServer(mux)

	client := github.NewClient(nil)
	u, err := url.Parse(server.URL + "/")
	if err != nil {
		t.Fatalf("failed to parse test server URL: %v", err)
	}
	client.BaseURL = u
	client.UploadURL = u

	return client, server.Close
}

// setupWebhookCheckPlugin wires a Plugin with a mocked API that reports the configured site URL
// and swallows the warn/debug logs the webhook check may emit on fallback paths.
func setupWebhookCheckPlugin(t *testing.T) *Plugin {
	t.Helper()

	config := &model.Config{}
	config.ServiceSettings.SiteURL = model.NewPointer(webhookTestSiteURL)

	api := &plugintest.API{}
	api.On("GetConfig").Return(config).Maybe()
	// useGitHubClient logs at warn level on any GitHub error: message + "error" + value.
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Maybe()
	// The org-level fallback logs at debug level: message + Owner/Repo/error key-value pairs.
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

	p := NewPlugin()
	p.setConfiguration(&Configuration{EncryptionKey: "dummyEncryptKey1"})
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)

	return p
}

func repoHooksPath(owner, repo string) string {
	return fmt.Sprintf("/repos/%s/%s/hooks", owner, repo)
}

func orgHooksPath(owner string) string {
	return fmt.Sprintf("/orgs/%s/hooks", owner)
}

func TestCheckIfConfiguredWebhookExists(t *testing.T) {
	userInfo := &GitHubUserInfo{
		UserID: "mockUserID",
		Token:  &oauth2.Token{AccessToken: "mockToken"},
	}

	tests := []struct {
		name        string
		repo        string
		handlers    map[string]http.HandlerFunc
		expectFound bool
		expectErr   string // substring expected in the error; empty means no error
	}{
		{
			name: "repo subscription: repo-level webhook present",
			repo: webhookTestRepo,
			handlers: map[string]http.HandlerFunc{
				repoHooksPath(webhookTestOwner, webhookTestRepo): jsonHandler(http.StatusOK, hooksResponse(webhookTestOtherURL, webhookTestMatchingURL)),
			},
			expectFound: true,
		},
		{
			name: "repo subscription: repo-level absent but org-level present",
			repo: webhookTestRepo,
			handlers: map[string]http.HandlerFunc{
				repoHooksPath(webhookTestOwner, webhookTestRepo): jsonHandler(http.StatusOK, hooksResponse(webhookTestOtherURL)),
				orgHooksPath(webhookTestOwner):                   jsonHandler(http.StatusOK, hooksResponse(webhookTestMatchingURL)),
			},
			expectFound: true,
		},
		{
			name: "repo subscription: webhook absent at both repo and org level",
			repo: webhookTestRepo,
			handlers: map[string]http.HandlerFunc{
				repoHooksPath(webhookTestOwner, webhookTestRepo): jsonHandler(http.StatusOK, hooksResponse(webhookTestOtherURL)),
				orgHooksPath(webhookTestOwner):                   jsonHandler(http.StatusOK, hooksResponse()),
			},
			expectFound: false,
		},
		{
			name: "repo subscription: org-level listing forbidden is swallowed",
			repo: webhookTestRepo,
			handlers: map[string]http.HandlerFunc{
				repoHooksPath(webhookTestOwner, webhookTestRepo): jsonHandler(http.StatusOK, hooksResponse(webhookTestOtherURL)),
				orgHooksPath(webhookTestOwner):                   jsonHandler(http.StatusForbidden, `{"message":"Must have admin rights to Organization."}`),
			},
			expectFound: false,
		},
		{
			name: "repo subscription: org-level listing 404 is swallowed",
			repo: webhookTestRepo,
			handlers: map[string]http.HandlerFunc{
				repoHooksPath(webhookTestOwner, webhookTestRepo): jsonHandler(http.StatusOK, hooksResponse(webhookTestOtherURL)),
				orgHooksPath(webhookTestOwner):                   jsonHandler(http.StatusNotFound, `{"message":"Not Found"}`),
			},
			expectFound: false,
		},
		{
			name: "org subscription: org-level webhook present",
			repo: "",
			handlers: map[string]http.HandlerFunc{
				orgHooksPath(webhookTestOwner): jsonHandler(http.StatusOK, hooksResponse(webhookTestMatchingURL)),
			},
			expectFound: true,
		},
		{
			name: "org subscription: error is propagated for caller suppression",
			repo: "",
			handlers: map[string]http.HandlerFunc{
				orgHooksPath(webhookTestOwner): jsonHandler(http.StatusNotFound, `{"message":"Not Found"}`),
			},
			expectFound: false,
			expectErr:   "404 Not Found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := setupWebhookCheckPlugin(t)
			client, closeServer := newHooksTestClient(t, tc.handlers)
			defer closeServer()

			found, err := p.checkIfConfiguredWebhookExists(context.Background(), client, userInfo, tc.repo, webhookTestOwner)

			assert.Equal(t, tc.expectFound, found)
			if tc.expectErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErr)
			}
		})
	}
}
