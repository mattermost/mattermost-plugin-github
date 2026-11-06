// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shurcooL/githubv4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func searchPayload(title string, hasNext bool, cursor string) map[string]any {
	return map[string]any{
		"issueCount": 1,
		"nodes": []map[string]any{{
			"title":      title,
			"number":     1,
			"url":        "https://example.com/pull/1",
			"repository": map[string]any{"url": "https://example.com/repo"},
			"author":     map[string]any{"login": "someone"},
		}},
		"pageInfo": map[string]any{"endCursor": cursor, "hasNextPage": hasNext},
	}
}

func writeSearchResponse(w http.ResponseWriter, review, assignee, open map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": map[string]any{"pullRequest": review, "assignee": assignee, "graphql": open},
	})
}

func readVariables(t *testing.T, r *http.Request) map[string]any {
	var body struct {
		Variables map[string]any `json:"variables"`
	}
	require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
	return body.Variables
}

func newTestClient(url, username string) *Client {
	return &Client{
		client:           githubv4.NewEnterpriseClient(url, http.DefaultClient),
		username:         username,
		getOrganizations: func() []string { return nil },
	}
}

func TestGetLHSData_ConcurrentUsersGetOwnData(t *testing.T) {
	const users = 2
	var mu sync.Mutex
	arrived := 0
	allArrived := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := readVariables(t, r)
		query, _ := vars[queryParamOpenPRQueryArg].(string)
		username := strings.Fields(strings.TrimPrefix(query, "author:"))[0]

		mu.Lock()
		arrived++
		if arrived == users {
			close(allArrived)
		}
		mu.Unlock()

		select {
		case <-allArrived:
		case <-time.After(5 * time.Second):
		}

		title := "title-" + username
		writeSearchResponse(w, searchPayload(title, false, ""), searchPayload(title, false, ""), searchPayload(title, false, ""))
	}))
	defer server.Close()

	var wg sync.WaitGroup
	for i := 0; i < users; i++ {
		username := fmt.Sprintf("user%d", i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			reviews, assignees, prs, err := newTestClient(server.URL, username).GetLHSData(context.Background())
			assert.NoError(t, err)
			require.Len(t, reviews, 1)
			require.Len(t, assignees, 1)
			require.Len(t, prs, 1)
			assert.Equal(t, "title-"+username, reviews[0].GetTitle())
			assert.Equal(t, "title-"+username, assignees[0].GetTitle())
			assert.Equal(t, "title-"+username, prs[0].GetTitle())
		}()
	}
	wg.Wait()
}

func TestGetLHSData_AccumulatesPages(t *testing.T) {
	var cursors []any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := readVariables(t, r)
		cursors = append(cursors, vars[queryParamReviewsCursor])

		if vars[queryParamReviewsCursor] == nil {
			writeSearchResponse(w, searchPayload("first", true, "next"), searchPayload("a", false, ""), searchPayload("p", false, ""))
			return
		}
		writeSearchResponse(w, searchPayload("second", false, ""), searchPayload("a", false, ""), searchPayload("p", false, ""))
	}))
	defer server.Close()

	reviews, assignees, prs, err := newTestClient(server.URL, "user").GetLHSData(context.Background())
	require.NoError(t, err)

	require.Len(t, reviews, 2)
	assert.Equal(t, "first", reviews[0].GetTitle())
	assert.Equal(t, "second", reviews[1].GetTitle())
	assert.Len(t, assignees, 1)
	assert.Len(t, prs, 1)
	assert.Equal(t, []any{nil, "next"}, cursors)
}
