// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"testing"
	"time"

	"github.com/google/go-github/v54/github"
	"github.com/stretchr/testify/assert"
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
