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
	"unicode/utf8"

	"github.com/golang/mock/gomock"
	"github.com/google/go-github/v54/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/mattermost/mattermost-plugin-github/server/mocks"
	"github.com/mattermost/mattermost-plugin-github/server/plugin/graphql"
)

func TestFormatChannelOverduePRBody(t *testing.T) {
	const baseURL = "https://github.com/"

	t.Run("renders owner/repo and a markdown-linked title", func(t *testing.T) {
		body := formatChannelOverduePRBody(
			"Fix race condition",
			"https://github.com/mattermost/mattermost/pull/12345",
			baseURL,
		)
		assert.Equal(t, "mattermost/mattermost - [Fix race condition](https://github.com/mattermost/mattermost/pull/12345)", body)
	})

	t.Run("escapes closing brackets in the title so the link does not terminate early", func(t *testing.T) {
		body := formatChannelOverduePRBody(
			"[MM-12345] Fix [thing]",
			"https://github.com/mattermost/mattermost/pull/1",
			baseURL,
		)
		// Only `]` needs escaping inside a markdown link's display text; `[` is allowed.
		assert.Contains(t, body, `[[MM-12345\] Fix [thing\]](https://github.com/mattermost/mattermost/pull/1)`)
	})

	t.Run("falls back to raw URL as the repo display when owner/repo cannot be parsed", func(t *testing.T) {
		body := formatChannelOverduePRBody(
			"Title",
			"https://example.invalid/something/odd",
			baseURL,
		)
		assert.Contains(t, body, "https://example.invalid/something/odd - [Title]")
	})

	t.Run("truncates very long titles before linking", func(t *testing.T) {
		title := strings.Repeat("a", 250)
		body := formatChannelOverduePRBody(
			title,
			"https://github.com/mattermost/mattermost/pull/1",
			baseURL,
		)
		assert.Contains(t, body, "...](https://github.com/mattermost/mattermost/pull/1)")
		// Title text inside the link should not exceed 200 'a's followed by the ellipsis.
		assert.NotContains(t, body, strings.Repeat("a", 201))
	})

	t.Run("does not include a leading bullet or reviewer prefix", func(t *testing.T) {
		// Locks in the contract that the per-PR body is composable: the digest builder
		// is responsible for the `- ` outer bullet and the reviewer header, not this helper.
		body := formatChannelOverduePRBody("Fix", "https://github.com/m/m/pull/1", baseURL)
		assert.False(t, strings.HasPrefix(body, "- "), "body should be composable, not pre-bulleted")
		assert.False(t, strings.HasPrefix(body, "@"), "body should not carry a reviewer prefix")
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
	entry := func(days int, reviewer, body string) slaDigestEntry {
		return slaDigestEntry{DaysOverdue: days, ReviewerDisplay: reviewer, Body: body}
	}

	t.Run("groups entries into the correct buckets in display order", func(t *testing.T) {
		entries := []slaDigestEntry{
			entry(2, "@a (a-gh)", "owner/repo - [A](url)"),
			entry(5, "@b (b-gh)", "owner/repo - [B](url)"),
			entry(12, "@c (c-gh)", "owner/repo - [C](url)"),
			entry(100, "@d (d-gh)", "owner/repo - [D](url)"),
			entry(400, "@e (e-gh)", "owner/repo - [E](url)"),
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
			entry(0, "@skip (skip-gh)", "owner/repo - [skipped](url)"),
			entry(-2, "@skip2 (skip2-gh)", "owner/repo - [skipped-too](url)"),
			entry(1, "@keep (keep-gh)", "owner/repo - [kept](url)"),
		}
		msg := buildSLADigestMessage(entries, 3)
		assert.Contains(t, msg, "[kept]")
		assert.NotContains(t, msg, "[skipped]")
		assert.NotContains(t, msg, "[skipped-too]")
	})

	t.Run("singular target days uses 'day'", func(t *testing.T) {
		msg := buildSLADigestMessage([]slaDigestEntry{entry(1, "@x (x-gh)", "owner/repo - [X](url)")}, 1)
		assert.Contains(t, msg, "target: 1 day from")
	})

	t.Run("zero target days falls back to plain header", func(t *testing.T) {
		msg := buildSLADigestMessage([]slaDigestEntry{entry(1, "@x (x-gh)", "owner/repo - [X](url)")}, 0)
		assert.True(t, strings.HasPrefix(msg, "### Pull request reviews past SLA\n"))
	})

	t.Run("reviewers within a bucket are sorted alphabetically (case-insensitive)", func(t *testing.T) {
		entries := []slaDigestEntry{
			entry(2, "@Zeta (zeta-gh)", "owner/repo - [PR-z](url)"),
			entry(2, "@alpha (alpha-gh)", "owner/repo - [PR-a](url)"),
			entry(2, "@Mu (mu-gh)", "owner/repo - [PR-m](url)"),
		}
		msg := buildSLADigestMessage(entries, 3)
		ai := strings.Index(msg, "@alpha")
		mi := strings.Index(msg, "@Mu")
		zi := strings.Index(msg, "@Zeta")
		assert.True(t, ai >= 0 && mi > ai && zi > mi, "expected alpha < Mu < Zeta (case-insensitive) in output")
	})

	t.Run("multiple PRs by the same reviewer in one bucket render under a single reviewer header with sorted indented bodies", func(t *testing.T) {
		const reviewer = "@harrison (hmhealey)"
		entries := []slaDigestEntry{
			entry(2, reviewer, "owner/repo - [zeta-pr](https://example/pr/3)"),
			entry(2, reviewer, "owner/repo - [alpha-pr](https://example/pr/1)"),
			entry(2, reviewer, "owner/repo - [mu-pr](https://example/pr/2)"),
		}
		msg := buildSLADigestMessage(entries, 3)

		// The reviewer header must appear EXACTLY once in this bucket — that's the whole
		// point of grouping; otherwise the digest still @-spams the reviewer per-PR.
		bucketStart := strings.Index(msg, "#### 1-3 days overdue")
		require.True(t, bucketStart >= 0, "expected the 1-3 days bucket header")
		bucketSlice := msg[bucketStart:]
		assert.Equal(t, 1, strings.Count(bucketSlice, "- "+reviewer+"\n"),
			"reviewer should appear once per bucket as the outer bullet")

		// Bodies should be indented two spaces and sorted alphabetically within the group.
		ai := strings.Index(bucketSlice, "  - owner/repo - [alpha-pr]")
		mi := strings.Index(bucketSlice, "  - owner/repo - [mu-pr]")
		zi := strings.Index(bucketSlice, "  - owner/repo - [zeta-pr]")
		assert.True(t, ai > 0 && mi > ai && zi > mi,
			"PR bodies should be indented (`  - `) and sorted alphabetically under the reviewer header")
	})

	t.Run("two reviewers in the same bucket render as two separate reviewer groups", func(t *testing.T) {
		entries := []slaDigestEntry{
			entry(2, "@alice (alice-gh)", "o/r - [a-pr](url)"),
			entry(2, "@alice (alice-gh)", "o/r - [b-pr](url)"),
			entry(2, "@bob (bob-gh)", "o/r - [c-pr](url)"),
		}
		msg := buildSLADigestMessage(entries, 3)
		bucketStart := strings.Index(msg, "#### 1-3 days overdue")
		require.True(t, bucketStart >= 0)
		bucket := msg[bucketStart:]

		// Each reviewer header appears exactly once; alice's two bodies are nested under hers.
		assert.Equal(t, 1, strings.Count(bucket, "- @alice (alice-gh)\n"))
		assert.Equal(t, 1, strings.Count(bucket, "- @bob (bob-gh)\n"))
		assert.Contains(t, bucket, "- @alice (alice-gh)\n  - o/r - [a-pr](url)\n  - o/r - [b-pr](url)\n")
	})

	t.Run("groupBucketEntriesByReviewer is deterministic and sorts bodies within a group", func(t *testing.T) {
		// Direct unit test on the helper: more focused than scanning the full message.
		bucketEntries := []slaDigestEntry{
			{ReviewerDisplay: "@bob (bob-gh)", Body: "o/r - [b1](url)"},
			{ReviewerDisplay: "@alice (alice-gh)", Body: "o/r - [zeta](url)"},
			{ReviewerDisplay: "@alice (alice-gh)", Body: "o/r - [alpha](url)"},
		}
		groups := groupBucketEntriesByReviewer(bucketEntries)
		require.Len(t, groups, 2)
		assert.Equal(t, "@alice (alice-gh)", groups[0].ReviewerDisplay)
		assert.Equal(t, []string{"o/r - [alpha](url)", "o/r - [zeta](url)"}, groups[0].Bodies,
			"bodies inside a reviewer group should be sorted alphabetically")
		assert.Equal(t, "@bob (bob-gh)", groups[1].ReviewerDisplay)
	})
}

func TestClipSLADigestMessage(t *testing.T) {
	t.Run("short message is unchanged", func(t *testing.T) {
		msg := "hello"
		assert.Equal(t, msg, clipSLADigestMessage(msg))
	})

	t.Run("oversized message is clipped with marker", func(t *testing.T) {
		msg := strings.Repeat("a", slaDigestMaxMessageRunes+1000)
		out := clipSLADigestMessage(msg)
		assert.LessOrEqual(t, utf8.RuneCountInString(out), slaDigestMaxMessageRunes)
		assert.True(t, strings.HasSuffix(out, slaDigestClippedMarker))
	})

	t.Run("multibyte runes are not split", func(t *testing.T) {
		msg := strings.Repeat("✓", slaDigestMaxMessageRunes+500)
		out := clipSLADigestMessage(msg)
		assert.LessOrEqual(t, utf8.RuneCountInString(out), slaDigestMaxMessageRunes)
		assert.True(t, utf8.ValidString(out))
	})
}

func TestGatherReviewersForPR(t *testing.T) {
	logins := func(rrs []reviewerRequest) []string {
		out := make([]string, 0, len(rrs))
		for _, r := range rrs {
			out = append(out, r.Login)
		}
		return out
	}

	t.Run("flattens direct user requests with empty Teams", func(t *testing.T) {
		pr := graphql.DigestPR{
			RequestedUsers: []string{"alice", "bob"},
		}
		got := gatherReviewersForPR(pr, func(graphql.DigestTeamRef) []string { return nil })
		assert.Equal(t, []string{"alice", "bob"}, logins(got))
		for _, r := range got {
			assert.Empty(t, r.Teams, "direct request should carry no team origin")
		}
	})

	t.Run("expands team references and tags reviewer with the originating team", func(t *testing.T) {
		core := graphql.DigestTeamRef{Org: "mattermost", Slug: "core"}
		pr := graphql.DigestPR{
			RequestedUsers: []string{"alice"},
			RequestedTeams: []graphql.DigestTeamRef{core},
		}
		resolver := func(team graphql.DigestTeamRef) []string {
			if team.Slug == "core" {
				return []string{"bob", "carol"}
			}
			return nil
		}
		got := gatherReviewersForPR(pr, resolver)
		assert.Equal(t, []string{"alice", "bob", "carol"}, logins(got))
		assert.Empty(t, got[0].Teams, "alice was directly requested, not via team")
		assert.Equal(t, []graphql.DigestTeamRef{core}, got[1].Teams)
		assert.Equal(t, []graphql.DigestTeamRef{core}, got[2].Teams)
	})

	t.Run("user in both direct list and team list is deduplicated to one entry retaining team origin", func(t *testing.T) {
		core := graphql.DigestTeamRef{Org: "mattermost", Slug: "core"}
		pr := graphql.DigestPR{
			RequestedUsers: []string{"bob"},
			RequestedTeams: []graphql.DigestTeamRef{core},
		}
		resolver := func(graphql.DigestTeamRef) []string { return []string{"bob"} }
		got := gatherReviewersForPR(pr, resolver)
		require.Len(t, got, 1, "bob should be merged into a single entry")
		assert.Equal(t, "bob", got[0].Login)
		assert.Equal(t, []graphql.DigestTeamRef{core}, got[0].Teams,
			"team origin must survive deduplication so the SLA backfill can use the team request as a fallback")
	})

	t.Run("user requested via two teams accumulates both teams", func(t *testing.T) {
		core := graphql.DigestTeamRef{Org: "mattermost", Slug: "core"}
		platform := graphql.DigestTeamRef{Org: "mattermost", Slug: "platform"}
		pr := graphql.DigestPR{
			RequestedTeams: []graphql.DigestTeamRef{core, platform},
		}
		resolver := func(graphql.DigestTeamRef) []string { return []string{"bob"} }
		got := gatherReviewersForPR(pr, resolver)
		require.Len(t, got, 1)
		assert.ElementsMatch(t, []graphql.DigestTeamRef{core, platform}, got[0].Teams)
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

	t.Run("empty logins are dropped from both direct and team-expanded paths", func(t *testing.T) {
		// Locks in the invariant that no reviewerRequest with Login=="" can reach the
		// caller, regardless of which code path a bad value comes in on. The GraphQL
		// layer already filters empty user logins, but resolveTeam is a callback whose
		// implementation we don't control, so the protection has to apply uniformly.
		core := graphql.DigestTeamRef{Org: "mattermost", Slug: "core"}
		pr := graphql.DigestPR{
			RequestedUsers: []string{"", "alice", ""},
			RequestedTeams: []graphql.DigestTeamRef{core},
		}
		resolver := func(graphql.DigestTeamRef) []string { return []string{"", "bob", ""} }
		got := gatherReviewersForPR(pr, resolver)
		assert.Equal(t, []string{"alice", "bob"}, logins(got),
			"empty logins must be dropped from both pr.RequestedUsers and team-expanded results")
		for _, r := range got {
			assert.NotEmpty(t, r.Login, "no reviewerRequest may reach the caller with an empty login")
		}
	})

	t.Run("dedupe is case-insensitive on login", func(t *testing.T) {
		core := graphql.DigestTeamRef{Org: "mattermost", Slug: "core"}
		pr := graphql.DigestPR{
			RequestedUsers: []string{"Bob"},
			RequestedTeams: []graphql.DigestTeamRef{core},
		}
		resolver := func(graphql.DigestTeamRef) []string { return []string{"bob"} }
		got := gatherReviewersForPR(pr, resolver)
		require.Len(t, got, 1)
		assert.Equal(t, "Bob", got[0].Login, "first-seen casing wins")
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

func TestPickServiceGitHubUser_ConfiguredMattermostUsername(t *testing.T) {
	p, mockKvStore, ctrl := setupServiceUserPickTest(t)
	defer ctrl.Finish()

	p.setConfiguration(&Configuration{
		EncryptionKey:         "dummyEncryptKey1",
		DigestServiceUsername: "it33",
	})

	api, ok := p.API.(*plugintest.API)
	require.True(t, ok, "expected plugintest.API")
	api.On("GetUserByUsername", "it33").Return(&model.User{Id: "ceo-user-id", Username: "it33"}, nil)
	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

	encryptedToken, err := encrypt([]byte("dummyEncryptKey1"), MockAccessToken)
	require.NoError(t, err)
	storedInfo := GitHubUserInfo{
		UserID:         "ceo-user-id",
		GitHubUsername: "ceo-gh",
		Token:          &oauth2.Token{AccessToken: encryptedToken},
		Settings:       &UserSettings{},
	}
	infoBytes, err := json.Marshal(&storedInfo)
	require.NoError(t, err)
	mockKvStore.EXPECT().Get("ceo-user-id"+githubTokenKey, gomock.Any()).DoAndReturn(
		func(_ string, out any) error {
			return json.Unmarshal(infoBytes, out)
		},
	)

	picked := p.pickServiceGitHubUser(context.Background())
	require.NotNil(t, picked)
	assert.Equal(t, "ceo-user-id", picked.UserID)
	assert.Equal(t, "ceo-gh", picked.GitHubUsername)
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
	}, reviewerRequest{Login: "alice"})
	require.Equal(t, 0, logCalls, "summary must not fire on resolver invocation")
	require.True(t, got.UTC().Equal(cachedTime),
		"resolver should return the cached SLA start time; got %v want %v", got.UTC(), cachedTime)

	summarize()
	assert.Equal(t, 1, logCalls, "summary should fire exactly once when summarize() is called")
	assert.Equal(t, 1, loggedHits, "kv_hits should reflect the resolver's actual usage after one cached lookup")
	assert.Equal(t, 0, loggedFetches, "no timeline pages fetched on the cached path")
}

func TestFetchAllOrgOpenPRs_CanceledContextReturnsNotOK(t *testing.T) {
	// A canceled context mid-iteration must NOT report anyOK=true to the caller, even if
	// earlier orgs in the list had already returned successfully (here we cancel before any
	// org runs, which is the strict version: anyOK starts false and must stay false). The
	// caller uses this to decide whether to advance slaDigestDayKVKey — an interrupted scan
	// should retry on the next scheduler tick, not silently skip until tomorrow.
	p, _, ctrl := setupServiceUserPickTest(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// graphQLClient is intentionally nil: the ctx.Err() check fires before any GraphQL call,
	// so a nil client must never be dereferenced on the cancellation path. Locks in the
	// fast-fail-without-network behavior alongside the failure-flag contract.
	prs, ok := p.fetchAllOrgOpenPRs(ctx, nil, []string{"mattermost"})
	assert.Nil(t, prs, "canceled context should not surface partial results")
	assert.False(t, ok, "canceled context must not be reported as a successful scan")
}

func TestNewDigestSLAStartResolver_SkipsSummaryLogWhenNoLookups(t *testing.T) {
	// summarize() is wired up via defer in collectAllOverdueSLAItems, so it can fire even
	// when the resolver was never invoked (e.g. PRs with no requested reviewers, or
	// degenerate prRefs that take the early return inside resolve). When no lookups ran,
	// the "0,0" summary line is misleading and is intentionally suppressed.
	p, _, ctrl := setupServiceUserPickTest(t)
	defer ctrl.Finish()

	api, ok := p.API.(*plugintest.API)
	require.True(t, ok, "expected plugintest.API")
	var logCalls int
	api.On("LogInfo",
		"SLA digest backfill summary",
		"timeline_pages_fetched", mock.AnythingOfType("int"),
		"kv_hits", mock.AnythingOfType("int"),
	).Run(func(mock.Arguments) { logCalls++ }).Maybe()

	_, summarize := p.newDigestSLAStartResolver(context.Background(), nil)
	summarize()
	assert.Equal(t, 0, logCalls, "summary must be suppressed when no lookups ran")
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
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

	p := NewPlugin()
	p.store = mockKvStore
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)
	p.setConfiguration(&Configuration{EncryptionKey: "dummyEncryptKey1"})

	return p, mockKvStore, ctrl
}
