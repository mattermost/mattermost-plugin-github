// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v54/github"

	"github.com/mattermost/mattermost-plugin-github/server/plugin/graphql"
)

const slaReviewReqKeyPrefix = "slarr_v1_"

// reviewSLAStartKey returns a stable KV key for (repo, PR, requested reviewer login).
func reviewSLAStartKey(owner, repo string, prNumber int, githubLogin string) string {
	normalized := strings.ToLower(strings.TrimSpace(owner)) + "/" + strings.ToLower(strings.TrimSpace(repo)) +
		"#" + strconv.Itoa(prNumber) + "@" + strings.ToLower(strings.TrimSpace(githubLogin))
	sum := sha256.Sum256([]byte(normalized))
	return slaReviewReqKeyPrefix + hex.EncodeToString(sum[:16])
}

// recordReviewRequestSLAStart stores when a reviewer was requested (from pull_request / review_requested webhook).
// Each new request overwrites the previous time for that (repo, PR, reviewer) pair so the SLA clock restarts on re-request.
func (p *Plugin) recordReviewRequestSLAStart(event *github.PullRequestEvent, requestedGitHubLogin string) {
	if event.GetRepo() == nil || event.GetPullRequest() == nil {
		return
	}
	owner := event.GetRepo().GetOwner().GetLogin()
	repo := event.GetRepo().GetName()
	num := event.GetPullRequest().GetNumber()
	if owner == "" || repo == "" || num == 0 || requestedGitHubLogin == "" {
		return
	}
	key := reviewSLAStartKey(owner, repo, num, requestedGitHubLogin)
	at := time.Now().UTC()
	val := []byte(at.Format(time.RFC3339Nano))
	if _, err := p.store.Set(key, val); err != nil {
		p.client.Log.Warn("Failed to store review SLA start time", "key", key, "error", err.Error())
	}
}

// cleanupReviewSLAKeys deletes all stored SLA start-time keys for a closed/merged PR.
func (p *Plugin) cleanupReviewSLAKeys(event *github.PullRequestEvent) {
	if event.GetRepo() == nil || event.GetPullRequest() == nil {
		return
	}
	owner := event.GetRepo().GetOwner().GetLogin()
	repo := event.GetRepo().GetName()
	num := event.GetPullRequest().GetNumber()
	if owner == "" || repo == "" || num == 0 {
		return
	}
	for _, reviewer := range event.GetPullRequest().RequestedReviewers {
		login := reviewer.GetLogin()
		if login == "" {
			continue
		}
		key := reviewSLAStartKey(owner, repo, num, login)
		if err := p.store.Delete(key); err != nil {
			p.client.Log.Debug("Failed to delete SLA key on PR close", "key", key, "error", err.Error())
		}
	}
}

func (p *Plugin) getReviewSLAStartTime(owner, repo string, prNumber int, githubLogin string) time.Time {
	key := reviewSLAStartKey(owner, repo, prNumber, githubLogin)
	var raw []byte
	if err := p.store.Get(key, &raw); err != nil {
		return time.Time{}
	}
	if len(raw) == 0 {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, string(raw))
	if err != nil {
		t, err = time.Parse(time.RFC3339, string(raw))
	}
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

// issueOwnerRepo resolves owner/name for a search result issue (prefers API fields, else HTML URL).
func issueOwnerRepo(pr *github.Issue, baseURL string) (owner, repo string) {
	if pr.Repository != nil {
		if o := pr.GetRepository().GetOwner(); o != nil {
			owner = o.GetLogin()
		}
		repo = pr.GetRepository().GetName()
	}
	if owner != "" && repo != "" {
		return owner, repo
	}
	return parseOwnerAndRepo(pr.GetHTMLURL(), baseURL)
}

// effectiveReviewSLAStart returns the timestamp used for SLA: when we recorded a review_request webhook
// for this reviewer on this PR, else the PR created time.
func (p *Plugin) effectiveReviewSLAStart(pr *github.Issue, baseURL, reviewerGitHubLogin string) github.Timestamp {
	owner, repo := issueOwnerRepo(pr, baseURL)
	num := pr.GetNumber()
	if owner == "" || repo == "" || num == 0 {
		return pr.GetCreatedAt()
	}
	if t := p.getReviewSLAStartTime(owner, repo, num, reviewerGitHubLogin); !t.IsZero() {
		return github.Timestamp{Time: t}
	}
	return pr.GetCreatedAt()
}

// findMostRecentReviewRequestTime walks PR timeline events chronologically and returns the
// timestamp of the most recent surviving review_requested event for githubLogin. Returns the
// zero time if the user has no current pending request (e.g. the request was later removed
// without being re-requested).
func findMostRecentReviewRequestTime(events []*github.Timeline, githubLogin string) time.Time {
	target := strings.ToLower(strings.TrimSpace(githubLogin))
	if target == "" {
		return time.Time{}
	}

	// Defensive sort by CreatedAt ascending; GitHub typically returns events in chronological
	// order already, but pagination joins are not guaranteed to be ordered across pages.
	sorted := make([]*github.Timeline, 0, len(events))
	for _, e := range events {
		if e == nil || e.CreatedAt == nil {
			continue
		}
		sorted = append(sorted, e)
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].CreatedAt.Time.Before(sorted[j].CreatedAt.Time)
	})

	var current time.Time
	for _, e := range sorted {
		if e.Reviewer == nil {
			continue
		}
		if strings.ToLower(e.Reviewer.GetLogin()) != target {
			continue
		}
		switch e.GetEvent() {
		case "review_requested":
			current = e.CreatedAt.Time
		case "review_request_removed":
			current = time.Time{}
		}
	}
	return current
}

// backfillReviewSLAStartFromTimeline reads the PR timeline for (owner, repo, prNumber) and, if
// it finds a surviving review_requested event for githubLogin, persists the timestamp under
// the same KV key used by recordReviewRequestSLAStart so subsequent digest runs are free.
// Returns the resolved time (zero if none was found) so callers can use it directly.
func (p *Plugin) backfillReviewSLAStartFromTimeline(ctx context.Context, gh *github.Client, owner, repo string, prNumber int, githubLogin string) (time.Time, error) {
	if gh == nil || owner == "" || repo == "" || prNumber == 0 || githubLogin == "" {
		return time.Time{}, nil
	}

	var events []*github.Timeline
	opts := &github.ListOptions{PerPage: 100}
	for {
		page, resp, err := gh.Issues.ListIssueTimeline(ctx, owner, repo, prNumber, opts)
		if err != nil {
			return time.Time{}, err
		}
		events = append(events, page...)
		if resp == nil || resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	found := findMostRecentReviewRequestTime(events, githubLogin)
	if found.IsZero() {
		return time.Time{}, nil
	}

	key := reviewSLAStartKey(owner, repo, prNumber, githubLogin)
	val := []byte(found.UTC().Format(time.RFC3339Nano))
	if _, err := p.store.Set(key, val); err != nil {
		p.client.Log.Warn("Failed to cache backfilled review SLA start time", "key", key, "error", err.Error())
	}
	return found.UTC(), nil
}

// effectiveReviewSLAStartForDigest is the digest's SLA-start resolver. It prefers the cached
// KV record, falls back to a one-time timeline backfill (caching the result), and finally
// falls back to PR open date. Unlike effectiveReviewSLAStart this can make a network call,
// so callers serving user-facing requests (/github todo, RHS) should keep using
// effectiveReviewSLAStart.
func (p *Plugin) effectiveReviewSLAStartForDigest(ctx context.Context, gh *github.Client, pr *github.Issue, baseURL, reviewerGitHubLogin string) github.Timestamp {
	owner, repo := issueOwnerRepo(pr, baseURL)
	num := pr.GetNumber()
	if owner == "" || repo == "" || num == 0 {
		return pr.GetCreatedAt()
	}
	if t := p.getReviewSLAStartTime(owner, repo, num, reviewerGitHubLogin); !t.IsZero() {
		return github.Timestamp{Time: t}
	}
	if gh != nil {
		if t, err := p.backfillReviewSLAStartFromTimeline(ctx, gh, owner, repo, num, reviewerGitHubLogin); err != nil {
			p.client.Log.Debug("Timeline backfill failed; falling back to PR created_at", "owner", owner, "repo", repo, "pr", num, "reviewer", reviewerGitHubLogin, "error", err.Error())
		} else if !t.IsZero() {
			return github.Timestamp{Time: t}
		}
	}
	return pr.GetCreatedAt()
}

// enrichReviewsWithSLAStart sets review_sla_start on LHS review items so the webapp can match server SLA logic.
func (p *Plugin) enrichReviewsWithSLAStart(reviews []*graphql.GithubPRDetails, reviewerLogin string) {
	cfg := p.getConfiguration()
	if cfg.ReviewTargetDays <= 0 {
		return
	}
	baseURL := cfg.getBaseURL()
	for _, d := range reviews {
		if d == nil || d.Issue == nil {
			continue
		}
		eff := p.effectiveReviewSLAStart(d.Issue, baseURL, reviewerLogin)
		if eff.IsZero() {
			continue
		}
		s := eff.Time.UTC().Format(time.RFC3339)
		d.ReviewSLAStartAt = &s
	}
}
