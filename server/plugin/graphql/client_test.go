// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package graphql

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shurcooL/githubv4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost/server/public/pluginapi"
)

func TestNewClientWithURLQueriesTheGivenEndpoint(t *testing.T) {
	var gotPath, gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"viewer":{"login":"octocat"}}}`))
	}))
	defer server.Close()

	client := NewClientWithURL(pluginapi.LogService{}, nil, oauth2.Token{AccessToken: "token"}, "octocat", "", server.URL+"/api/graphql")

	var query struct {
		Viewer struct {
			Login githubv4.String
		}
	}
	require.NoError(t, client.executeQuery(context.Background(), &query, nil))

	assert.Equal(t, "/api/graphql", gotPath)
	assert.Equal(t, "Bearer token", gotAuth)
	assert.Equal(t, githubv4.String("octocat"), query.Viewer.Login)
}

func TestNewClientUsesTheEnterpriseGraphQLPath(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"viewer":{"login":"octocat"}}}`))
	}))
	defer server.Close()

	client := NewClient(pluginapi.LogService{}, nil, oauth2.Token{AccessToken: "token"}, "octocat", "", server.URL)

	var query struct {
		Viewer struct {
			Login githubv4.String
		}
	}
	require.NoError(t, client.executeQuery(context.Background(), &query, nil))

	assert.Equal(t, "/api/graphql", gotPath)
}
