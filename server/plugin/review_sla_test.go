// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"testing"
	"time"

	"github.com/google/go-github/v54/github"
	"github.com/stretchr/testify/assert"

	"github.com/mattermost/mattermost-plugin-github/server/plugin/graphql"
)

func TestReviewSLAStartKeyStable(t *testing.T) {
	k1 := reviewSLAStartKey("Mattermost", "mattermost", 12345, "octocat")
	k2 := reviewSLAStartKey("mattermost", "mattermost", 12345, "OctoCat")
	assert.Equal(t, k1, k2, "key should be case-insensitive")

	k3 := reviewSLAStartKey("Mattermost", "mattermost", 99999, "octocat")
	assert.NotEqual(t, k1, k3)
}

func TestPrRefFromIssue(t *testing.T) {
	const baseURL = "https://github.com/"
	createdAt := github.Timestamp{Time: time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)}

	t.Run("prefers explicit Repository fields", func(t *testing.T) {
		pr := &github.Issue{
			Number:    github.Int(42),
			HTMLURL:   github.String("https://github.com/wrong/wrong/pull/42"),
			CreatedAt: &createdAt,
			Repository: &github.Repository{
				Name:  github.String("right"),
				Owner: &github.User{Login: github.String("owner")},
			},
		}
		got := prRefFromIssue(pr, baseURL)
		assert.Equal(t, prRef{Owner: "owner", Repo: "right", Number: 42, CreatedAt: createdAt}, got)
	})

	t.Run("falls back to HTMLURL parsing when Repository is missing", func(t *testing.T) {
		pr := &github.Issue{
			Number:    github.Int(7),
			HTMLURL:   github.String("https://github.com/mattermost/plugin-github/pull/7"),
			CreatedAt: &createdAt,
		}
		got := prRefFromIssue(pr, baseURL)
		assert.Equal(t, prRef{Owner: "mattermost", Repo: "plugin-github", Number: 7, CreatedAt: createdAt}, got)
	})
}

func TestFindMostRecentReviewRequestTime(t *testing.T) {
	user := func(login string) *github.User { return &github.User{Login: github.String(login)} }
	at := func(s string) *github.Timestamp {
		ts, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatalf("bad timestamp %q: %v", s, err)
		}
		return &github.Timestamp{Time: ts}
	}
	ev := func(name, login, ts string) *github.Timeline {
		return &github.Timeline{
			Event:     github.String(name),
			Reviewer:  user(login),
			CreatedAt: at(ts),
		}
	}

	t.Run("no events returns zero", func(t *testing.T) {
		got := findMostRecentReviewRequestTime(nil, "octocat")
		assert.True(t, got.IsZero())
	})

	t.Run("single review_requested returns its timestamp", func(t *testing.T) {
		events := []*github.Timeline{
			ev("review_requested", "octocat", "2026-04-20T10:00:00Z"),
		}
		got := findMostRecentReviewRequestTime(events, "octocat")
		assert.Equal(t, "2026-04-20T10:00:00Z", got.Format(time.RFC3339))
	})

	t.Run("login match is case-insensitive", func(t *testing.T) {
		events := []*github.Timeline{
			ev("review_requested", "OctoCat", "2026-04-20T10:00:00Z"),
		}
		got := findMostRecentReviewRequestTime(events, "octocat")
		assert.Equal(t, "2026-04-20T10:00:00Z", got.Format(time.RFC3339))
	})

	t.Run("review_request_removed after request invalidates pending", func(t *testing.T) {
		events := []*github.Timeline{
			ev("review_requested", "octocat", "2026-04-20T10:00:00Z"),
			ev("review_request_removed", "octocat", "2026-04-21T10:00:00Z"),
		}
		got := findMostRecentReviewRequestTime(events, "octocat")
		assert.True(t, got.IsZero(), "removed request should leave no pending start")
	})

	t.Run("re-request after remove uses the most recent request", func(t *testing.T) {
		events := []*github.Timeline{
			ev("review_requested", "octocat", "2026-04-20T10:00:00Z"),
			ev("review_request_removed", "octocat", "2026-04-21T10:00:00Z"),
			ev("review_requested", "octocat", "2026-04-22T10:00:00Z"),
		}
		got := findMostRecentReviewRequestTime(events, "octocat")
		assert.Equal(t, "2026-04-22T10:00:00Z", got.Format(time.RFC3339))
	})

	t.Run("events for other reviewers are ignored", func(t *testing.T) {
		events := []*github.Timeline{
			ev("review_requested", "alice", "2026-04-25T10:00:00Z"),
			ev("review_requested", "octocat", "2026-04-20T10:00:00Z"),
		}
		got := findMostRecentReviewRequestTime(events, "octocat")
		assert.Equal(t, "2026-04-20T10:00:00Z", got.Format(time.RFC3339))
	})

	t.Run("out-of-order pages are sorted defensively", func(t *testing.T) {
		events := []*github.Timeline{
			ev("review_requested", "octocat", "2026-04-22T10:00:00Z"),
			ev("review_request_removed", "octocat", "2026-04-21T10:00:00Z"),
			ev("review_requested", "octocat", "2026-04-20T10:00:00Z"),
		}
		got := findMostRecentReviewRequestTime(events, "octocat")
		assert.Equal(t, "2026-04-22T10:00:00Z", got.Format(time.RFC3339))
	})

	t.Run("non-review-request events are ignored", func(t *testing.T) {
		events := []*github.Timeline{
			ev("commented", "octocat", "2026-04-25T10:00:00Z"),
			ev("labeled", "octocat", "2026-04-26T10:00:00Z"),
			ev("review_requested", "octocat", "2026-04-20T10:00:00Z"),
		}
		got := findMostRecentReviewRequestTime(events, "octocat")
		assert.Equal(t, "2026-04-20T10:00:00Z", got.Format(time.RFC3339))
	})
}

func TestFindEarliestSurvivingTeamRequestTime(t *testing.T) {
	at := func(s string) *github.Timestamp {
		ts, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatalf("bad timestamp %q: %v", s, err)
		}
		return &github.Timestamp{Time: ts}
	}
	teamEv := func(name, slug, ts string) *github.Timeline {
		return &github.Timeline{
			Event:         github.String(name),
			RequestedTeam: &github.Team{Slug: github.String(slug)},
			CreatedAt:     at(ts),
		}
	}

	core := graphql.DigestTeamRef{Org: "mattermost", Slug: "core"}
	platform := graphql.DigestTeamRef{Org: "mattermost", Slug: "platform"}

	t.Run("empty teams returns zero", func(t *testing.T) {
		got := findEarliestSurvivingTeamRequestTime(nil, nil)
		assert.True(t, got.IsZero())
	})

	t.Run("single team request returns its timestamp", func(t *testing.T) {
		events := []*github.Timeline{teamEv("review_requested", "core", "2026-04-20T10:00:00Z")}
		got := findEarliestSurvivingTeamRequestTime(events, []graphql.DigestTeamRef{core})
		assert.Equal(t, "2026-04-20T10:00:00Z", got.Format(time.RFC3339))
	})

	t.Run("removed team request leaves no surviving anchor", func(t *testing.T) {
		events := []*github.Timeline{
			teamEv("review_requested", "core", "2026-04-20T10:00:00Z"),
			teamEv("review_request_removed", "core", "2026-04-21T10:00:00Z"),
		}
		got := findEarliestSurvivingTeamRequestTime(events, []graphql.DigestTeamRef{core})
		assert.True(t, got.IsZero())
	})

	t.Run("re-request after remove uses the most recent request for that team", func(t *testing.T) {
		events := []*github.Timeline{
			teamEv("review_requested", "core", "2026-04-20T10:00:00Z"),
			teamEv("review_request_removed", "core", "2026-04-21T10:00:00Z"),
			teamEv("review_requested", "core", "2026-04-22T10:00:00Z"),
		}
		got := findEarliestSurvivingTeamRequestTime(events, []graphql.DigestTeamRef{core})
		assert.Equal(t, "2026-04-22T10:00:00Z", got.Format(time.RFC3339))
	})

	t.Run("two surviving teams return the earlier ask", func(t *testing.T) {
		events := []*github.Timeline{
			teamEv("review_requested", "platform", "2026-04-15T10:00:00Z"),
			teamEv("review_requested", "core", "2026-04-22T10:00:00Z"),
		}
		got := findEarliestSurvivingTeamRequestTime(events, []graphql.DigestTeamRef{core, platform})
		assert.Equal(t, "2026-04-15T10:00:00Z", got.Format(time.RFC3339),
			"reviewer in two requested teams has been on the hook since the EARLIEST surviving ask")
	})

	t.Run("removed team is excluded from the earliest calculation", func(t *testing.T) {
		events := []*github.Timeline{
			teamEv("review_requested", "platform", "2026-04-15T10:00:00Z"),
			teamEv("review_request_removed", "platform", "2026-04-16T10:00:00Z"),
			teamEv("review_requested", "core", "2026-04-22T10:00:00Z"),
		}
		got := findEarliestSurvivingTeamRequestTime(events, []graphql.DigestTeamRef{core, platform})
		assert.Equal(t, "2026-04-22T10:00:00Z", got.Format(time.RFC3339))
	})

	t.Run("events for non-requested teams are ignored", func(t *testing.T) {
		events := []*github.Timeline{
			teamEv("review_requested", "other-team", "2026-04-15T10:00:00Z"),
			teamEv("review_requested", "core", "2026-04-22T10:00:00Z"),
		}
		got := findEarliestSurvivingTeamRequestTime(events, []graphql.DigestTeamRef{core})
		assert.Equal(t, "2026-04-22T10:00:00Z", got.Format(time.RFC3339))
	})

	t.Run("user-scoped events with nil RequestedTeam are skipped", func(t *testing.T) {
		userEv := &github.Timeline{
			Event:     github.String("review_requested"),
			Reviewer:  &github.User{Login: github.String("alice")},
			CreatedAt: at("2026-04-15T10:00:00Z"),
		}
		events := []*github.Timeline{
			userEv,
			teamEv("review_requested", "core", "2026-04-22T10:00:00Z"),
		}
		got := findEarliestSurvivingTeamRequestTime(events, []graphql.DigestTeamRef{core})
		assert.Equal(t, "2026-04-22T10:00:00Z", got.Format(time.RFC3339))
	})

	t.Run("out-of-order pages are sorted defensively", func(t *testing.T) {
		events := []*github.Timeline{
			teamEv("review_requested", "core", "2026-04-22T10:00:00Z"),
			teamEv("review_request_removed", "core", "2026-04-21T10:00:00Z"),
			teamEv("review_requested", "core", "2026-04-20T10:00:00Z"),
		}
		got := findEarliestSurvivingTeamRequestTime(events, []graphql.DigestTeamRef{core})
		assert.Equal(t, "2026-04-22T10:00:00Z", got.Format(time.RFC3339),
			"defensive sort means the latest event wins after re-request, even if pages arrived out of order")
	})

	t.Run("team slug match is case-insensitive", func(t *testing.T) {
		events := []*github.Timeline{teamEv("review_requested", "Core", "2026-04-20T10:00:00Z")}
		got := findEarliestSurvivingTeamRequestTime(events, []graphql.DigestTeamRef{{Slug: "CORE"}})
		assert.Equal(t, "2026-04-20T10:00:00Z", got.Format(time.RFC3339))
	})
}
