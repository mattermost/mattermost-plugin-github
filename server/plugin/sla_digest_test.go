// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-github/v54/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/mattermost/mattermost-plugin-github/server/mocks"
	"github.com/mattermost/mattermost-plugin-github/server/plugin/graphql"
)

func TestFormatChannelOverdueReviewLine(t *testing.T) {
	const baseURL = "https://github.com/"

	t.Run("renders connected reviewer with @-mention and bracketed github login", func(t *testing.T) {
		line := formatChannelOverdueReviewLine(
			"@harrison (hmhealey)",
			"Fix race condition",
			"https://github.com/mattermost/mattermost/pull/12345",
			baseURL,
		)
		assert.Equal(t, "- @harrison (hmhealey) - mattermost/mattermost - [Fix race condition](https://github.com/mattermost/mattermost/pull/12345)", line)
	})

	t.Run("renders unconnected reviewer with `(not connected) - login` prefix", func(t *testing.T) {
		line := formatChannelOverdueReviewLine(
			"(not connected) - hmhealey",
			"Fix race condition",
			"https://github.com/mattermost/mattermost/pull/12345",
			baseURL,
		)
		assert.Equal(t, "- (not connected) - hmhealey - mattermost/mattermost - [Fix race condition](https://github.com/mattermost/mattermost/pull/12345)", line)
	})

	t.Run("escapes closing brackets in the title so the link does not terminate early", func(t *testing.T) {
		line := formatChannelOverdueReviewLine(
			"hmhealey",
			"[MM-12345] Fix [thing]",
			"https://github.com/mattermost/mattermost/pull/1",
			baseURL,
		)
		// Only `]` needs escaping inside a markdown link's display text; `[` is allowed.
		assert.Contains(t, line, `[[MM-12345\] Fix [thing\]](https://github.com/mattermost/mattermost/pull/1)`)
	})

	t.Run("falls back to raw URL as the repo display when owner/repo cannot be parsed", func(t *testing.T) {
		line := formatChannelOverdueReviewLine(
			"hmhealey",
			"Title",
			"https://example.invalid/something/odd",
			baseURL,
		)
		assert.Contains(t, line, "https://example.invalid/something/odd - [Title]")
	})

	t.Run("truncates very long titles before linking", func(t *testing.T) {
		title := strings.Repeat("a", 250)
		line := formatChannelOverdueReviewLine(
			"hmhealey",
			title,
			"https://github.com/mattermost/mattermost/pull/1",
			baseURL,
		)
		assert.Contains(t, line, "...](https://github.com/mattermost/mattermost/pull/1)")
		// Title body inside the link should not exceed 200 'a's followed by the ellipsis.
		assert.NotContains(t, line, strings.Repeat("a", 201))
	})
}

func TestSLABucketIndex(t *testing.T) {
	cases := []struct {
		daysOverdue int
		wantLabel   string
	}{
		{0, ""},
		{-3, ""},
		{1, "1-3 days overdue"},
		{3, "1-3 days overdue"},
		{4, "4-7 days overdue"},
		{7, "4-7 days overdue"},
		{8, "8-14 days overdue"},
		{14, "8-14 days overdue"},
		{15, "15-30 days overdue"},
		{30, "15-30 days overdue"},
		{31, "31-90 days overdue"},
		{90, "31-90 days overdue"},
		{91, "91-365 days overdue"},
		{365, "91-365 days overdue"},
		{366, "More than 1 year overdue"},
		{1000, "More than 1 year overdue"},
	}

	for _, tc := range cases {
		idx := slaBucketIndex(tc.daysOverdue)
		if tc.wantLabel == "" {
			assert.Equal(t, -1, idx, "daysOverdue=%d expected no bucket", tc.daysOverdue)
			continue
		}
		if assert.NotEqual(t, -1, idx, "daysOverdue=%d expected a bucket", tc.daysOverdue) {
			assert.Equal(t, tc.wantLabel, slaBuckets[idx].label, "daysOverdue=%d", tc.daysOverdue)
		}
	}
}

func TestBuildSLADigestMessage(t *testing.T) {
	t.Run("groups entries into the correct buckets in display order", func(t *testing.T) {
		entries := []slaDigestEntry{
			{DaysOverdue: 2, Line: "- a"},
			{DaysOverdue: 5, Line: "- b"},
			{DaysOverdue: 12, Line: "- c"},
			{DaysOverdue: 100, Line: "- d"},
			{DaysOverdue: 400, Line: "- e"},
		}

		msg := buildSLADigestMessage(entries, 3)

		assert.True(t, strings.HasPrefix(msg, "### Pull request reviews past SLA (target: 3 days from most recent review request)"))

		// Verify bucket order in message: most overdue first.
		idxYear := strings.Index(msg, "#### More than 1 year overdue")
		idx91 := strings.Index(msg, "#### 91-365 days overdue")
		idx8 := strings.Index(msg, "#### 8-14 days overdue")
		idx4 := strings.Index(msg, "#### 4-7 days overdue")
		idx1 := strings.Index(msg, "#### 1-3 days overdue")
		assert.True(t, idxYear >= 0 && idx91 > idxYear && idx8 > idx91 && idx4 > idx8 && idx1 > idx4, "buckets should appear in most-overdue-first order")

		assert.NotContains(t, msg, "#### 31-90 days overdue", "empty bucket should be omitted")
		assert.NotContains(t, msg, "#### 15-30 days overdue", "empty bucket should be omitted")
	})

	t.Run("non-overdue entries are dropped", func(t *testing.T) {
		entries := []slaDigestEntry{
			{DaysOverdue: 0, Line: "- skipped"},
			{DaysOverdue: -2, Line: "- skipped-too"},
			{DaysOverdue: 1, Line: "- kept"},
		}
		msg := buildSLADigestMessage(entries, 3)
		assert.Contains(t, msg, "- kept")
		assert.NotContains(t, msg, "- skipped")
		assert.NotContains(t, msg, "- skipped-too")
	})

	t.Run("singular target days uses 'day'", func(t *testing.T) {
		msg := buildSLADigestMessage([]slaDigestEntry{{DaysOverdue: 1, Line: "- x"}}, 1)
		assert.Contains(t, msg, "target: 1 day from")
	})

	t.Run("zero target days falls back to plain header", func(t *testing.T) {
		msg := buildSLADigestMessage([]slaDigestEntry{{DaysOverdue: 1, Line: "- x"}}, 0)
		assert.True(t, strings.HasPrefix(msg, "### Pull request reviews past SLA\n"))
	})

	t.Run("lines within a bucket are sorted alphabetically", func(t *testing.T) {
		entries := []slaDigestEntry{
			{DaysOverdue: 2, Line: "- zeta"},
			{DaysOverdue: 2, Line: "- alpha"},
			{DaysOverdue: 2, Line: "- mu"},
		}
		msg := buildSLADigestMessage(entries, 3)
		ai := strings.Index(msg, "- alpha")
		mi := strings.Index(msg, "- mu")
		zi := strings.Index(msg, "- zeta")
		assert.True(t, ai >= 0 && mi > ai && zi > mi, "expected alpha < mu < zeta in output")
	})
}

func TestGatherReviewersForPR(t *testing.T) {
	t.Run("flattens direct user requests", func(t *testing.T) {
		pr := graphql.DigestPR{
			RequestedUsers: []string{"alice", "bob"},
		}
		got := gatherReviewersForPR(pr, func(graphql.DigestTeamRef) []string { return nil })
		assert.Equal(t, []string{"alice", "bob"}, got)
	})

	t.Run("expands team references through the resolver", func(t *testing.T) {
		pr := graphql.DigestPR{
			RequestedUsers: []string{"alice"},
			RequestedTeams: []graphql.DigestTeamRef{{Org: "mattermost", Slug: "core"}},
		}
		resolver := func(team graphql.DigestTeamRef) []string {
			if team.Slug == "core" {
				return []string{"bob", "carol"}
			}
			return nil
		}
		got := gatherReviewersForPR(pr, resolver)
		assert.Equal(t, []string{"alice", "bob", "carol"}, got)
	})

	t.Run("does not mutate the underlying RequestedUsers slice", func(t *testing.T) {
		original := []string{"alice"}
		pr := graphql.DigestPR{
			RequestedUsers: original,
			RequestedTeams: []graphql.DigestTeamRef{{Org: "mattermost", Slug: "core"}},
		}
		resolver := func(graphql.DigestTeamRef) []string { return []string{"bob"} }
		_ = gatherReviewersForPR(pr, resolver)
		assert.Equal(t, []string{"alice"}, original, "team expansion must not append into the caller's slice")
	})

	t.Run("returns empty slice when no reviewers are requested", func(t *testing.T) {
		got := gatherReviewersForPR(graphql.DigestPR{}, func(graphql.DigestTeamRef) []string { return nil })
		assert.Empty(t, got)
	})
}

func TestPickServiceGitHubUser_DeterministicOrdering(t *testing.T) {
	// Returns the lexicographically-first connected user even when KV iteration order is
	// reversed, so digest output is stable across runs and cluster nodes.
	p, mockKvStore, ctrl := setupServiceUserPickTest(t)
	defer ctrl.Finish()

	mockKvStore.EXPECT().ListKeys(0, keysPerPage, gomock.Any()).Return(
		[]string{
			"user-z" + githubTokenKey,
			"user-a" + githubTokenKey,
			"user-m" + githubTokenKey,
		},
		nil,
	)

	encryptedToken, err := encrypt([]byte("dummyEncryptKey1"), MockAccessToken)
	require.NoError(t, err)
	storedInfo := GitHubUserInfo{
		UserID:         "user-a",
		GitHubUsername: "user-a-gh",
		Token:          &oauth2.Token{AccessToken: encryptedToken},
		Settings:       &UserSettings{},
	}
	infoBytes, err := json.Marshal(&storedInfo)
	require.NoError(t, err)

	mockKvStore.EXPECT().Get("user-a"+githubTokenKey, gomock.Any()).DoAndReturn(
		func(_ string, out any) error {
			return json.Unmarshal(infoBytes, out)
		},
	)

	picked := p.pickServiceGitHubUser(context.Background())
	require.NotNil(t, picked, "expected a connected user to be picked")
	assert.Equal(t, "user-a", picked.UserID, "expected lexicographically-first user")
}

func TestPickServiceGitHubUser_SkipsUsersWithBrokenTokens(t *testing.T) {
	// A user whose KV record can't be loaded should be skipped, not abort the digest. We
	// expect the iterator to fall through to the next user.
	p, mockKvStore, ctrl := setupServiceUserPickTest(t)
	defer ctrl.Finish()

	mockKvStore.EXPECT().ListKeys(0, keysPerPage, gomock.Any()).Return(
		[]string{"user-a" + githubTokenKey, "user-b" + githubTokenKey},
		nil,
	)

	mockKvStore.EXPECT().Get("user-a"+githubTokenKey, gomock.Any()).Return(errors.New("kv read failed"))

	encryptedToken, err := encrypt([]byte("dummyEncryptKey1"), MockAccessToken)
	require.NoError(t, err)
	storedInfo := GitHubUserInfo{
		UserID:         "user-b",
		GitHubUsername: "user-b-gh",
		Token:          &oauth2.Token{AccessToken: encryptedToken},
		Settings:       &UserSettings{},
	}
	infoBytes, err := json.Marshal(&storedInfo)
	require.NoError(t, err)
	mockKvStore.EXPECT().Get("user-b"+githubTokenKey, gomock.Any()).DoAndReturn(
		func(_ string, out any) error {
			return json.Unmarshal(infoBytes, out)
		},
	)

	picked := p.pickServiceGitHubUser(context.Background())
	require.NotNil(t, picked)
	assert.Equal(t, "user-b", picked.UserID)
}

func TestPickServiceGitHubUser_NoConnectedUsers(t *testing.T) {
	p, mockKvStore, ctrl := setupServiceUserPickTest(t)
	defer ctrl.Finish()

	mockKvStore.EXPECT().ListKeys(0, keysPerPage, gomock.Any()).Return([]string{}, nil)

	assert.Nil(t, p.pickServiceGitHubUser(context.Background()))
}

func TestNewDigestSLAStartResolver_LogsAccurateCounters(t *testing.T) {
	// The summary log must (a) not fire until summarize() is called and (b) reflect the
	// resolver's actual usage when it does. Locks in the fix for a prior bug where the log
	// was deferred inside the constructor and so always reported zeroes.
	p, mockKvStore, ctrl := setupServiceUserPickTest(t)
	defer ctrl.Finish()

	cachedTime := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	mockKvStore.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ string, out any) error {
			b, ok := out.(*[]byte)
			require.True(t, ok, "getReviewSLAStartTime should pass *[]byte")
			*b = []byte(cachedTime.Format(time.RFC3339Nano))
			return nil
		},
	).AnyTimes()

	api, ok := p.API.(*plugintest.API)
	require.True(t, ok, "expected plugintest.API")
	var (
		logCalls      int
		loggedHits    int
		loggedFetches int
	)
	api.On("LogInfo",
		"SLA digest backfill summary",
		"timeline_pages_fetched", mock.AnythingOfType("int"),
		"kv_hits", mock.AnythingOfType("int"),
	).Run(func(args mock.Arguments) {
		logCalls++
		loggedFetches = args.Get(2).(int)
		loggedHits = args.Get(4).(int)
	}).Maybe()

	resolve, summarize := p.newDigestSLAStartResolver(context.Background(), nil)
	require.Equal(t, 0, logCalls, "summary must not fire at construction time")

	got := resolve(prRef{
		Owner:     "owner",
		Repo:      "repo",
		Number:    1,
		CreatedAt: github.Timestamp{Time: time.Now().UTC()},
	}, "alice")
	require.Equal(t, 0, logCalls, "summary must not fire on resolver invocation")
	require.True(t, got.UTC().Equal(cachedTime),
		"resolver should return the cached SLA start time; got %v want %v", got.UTC(), cachedTime)

	summarize()
	assert.Equal(t, 1, logCalls, "summary should fire exactly once when summarize() is called")
	assert.Equal(t, 1, loggedHits, "kv_hits should reflect the resolver's actual usage after one cached lookup")
	assert.Equal(t, 0, loggedFetches, "no timeline pages fetched on the cached path")
}

// setupServiceUserPickTest wires up a Plugin instance with a mocked KvStore and a mock
// plugintest.API that swallows the warn-level logs pickServiceGitHubUser emits on failure
// paths. Returns the plugin, the KV mock for setting expectations, and the gomock controller
// for the caller to Finish().
func setupServiceUserPickTest(t *testing.T) (*Plugin, *mocks.MockKvStore, *gomock.Controller) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockKvStore := mocks.NewMockKvStore(ctrl)

	api := &plugintest.API{}
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

	p := NewPlugin()
	p.store = mockKvStore
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)
	p.setConfiguration(&Configuration{EncryptionKey: "dummyEncryptKey1"})

	return p, mockKvStore, ctrl
}
