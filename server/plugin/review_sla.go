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

// prRef is the minimal PR identity the SLA code needs: stable enough to derive the KV key,
// rich enough to fall back to PR open time when no review-request record exists. Centralizing
// this avoids forging *github.Issue values whenever a non-search caller (e.g. the digest)
// needs to ask SLA questions.
type prRef struct {
	Owner     string
	Repo      string
	Number    int
	CreatedAt github.Timestamp
}

// prRefFromIssue extracts a prRef from a search-result issue. Owner/repo come from the API
// response when present, otherwise from the HTML URL (mirrors issueOwnerRepo's behavior).
func prRefFromIssue(pr *github.Issue, baseURL string) prRef {
	owner, repo := issueOwnerRepo(pr, baseURL)
	return prRef{
		Owner:     owner,
		Repo:      repo,
		Number:    pr.GetNumber(),
		CreatedAt: pr.GetCreatedAt(),
	}
}

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
	p.storeReviewSLAStart(owner, repo, num, requestedGitHubLogin, time.Now().UTC())
}

// storeReviewSLAStart writes a review-request timestamp to KV under the canonical key. Used by
// both the live webhook path and the digest's timeline backfill so the wire format matches.
func (p *Plugin) storeReviewSLAStart(owner, repo string, prNumber int, githubLogin string, t time.Time) {
	key := reviewSLAStartKey(owner, repo, prNumber, githubLogin)
	val := []byte(t.UTC().Format(time.RFC3339Nano))
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

// effectiveReviewSLAStart returns the timestamp used for SLA: when we recorded a review_request
// webhook for this reviewer on this PR, else the PR created time. Read-only — callers serving
// user-facing requests (/github todo, RHS) can call freely without network I/O.
func (p *Plugin) effectiveReviewSLAStart(pr prRef, reviewerGitHubLogin string) github.Timestamp {
	if pr.Owner == "" || pr.Repo == "" || pr.Number == 0 {
		return pr.CreatedAt
	}
	if t := p.getReviewSLAStartTime(pr.Owner, pr.Repo, pr.Number, reviewerGitHubLogin); !t.IsZero() {
		return github.Timestamp{Time: t}
	}
	return pr.CreatedAt
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
		return sorted[i].CreatedAt.Before(sorted[j].CreatedAt.Time)
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

// findEarliestSurvivingTeamRequestTime walks PR timeline events and returns the earliest
// surviving review_requested event time across any of the given teams. "Surviving" means
// not later cancelled by a matching review_request_removed event for the same team. Returns
// the zero time when no team has a still-active request (or when teams is empty).
//
// Used as a fallback for reviewers added solely via team membership: their user-scoped
// timeline doesn't contain a review_requested event, so without this they'd fall all the way
// back to the PR's created_at and overstate days-overdue. The earliest still-active team
// request is the right anchor: if a user is in two requested teams, they have been on the
// hook since the first ask, and a later team request doesn't reset that clock.
//
// Match is on team slug only. A PR's timeline lives within a single GitHub org, and team
// slugs are unique within an org, so cross-org slug collision isn't possible here.
func findEarliestSurvivingTeamRequestTime(events []*github.Timeline, teams []graphql.DigestTeamRef) time.Time {
	if len(teams) == 0 {
		return time.Time{}
	}
	wantedSlugs := make(map[string]bool, len(teams))
	for _, t := range teams {
		slug := strings.ToLower(strings.TrimSpace(t.Slug))
		if slug != "" {
			wantedSlugs[slug] = true
		}
	}
	if len(wantedSlugs) == 0 {
		return time.Time{}
	}

	sorted := make([]*github.Timeline, 0, len(events))
	for _, e := range events {
		if e == nil || e.CreatedAt == nil {
			continue
		}
		sorted = append(sorted, e)
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].CreatedAt.Before(sorted[j].CreatedAt.Time)
	})

	surviving := make(map[string]time.Time)
	for _, e := range sorted {
		if e.RequestedTeam == nil {
			continue
		}
		slug := strings.ToLower(e.RequestedTeam.GetSlug())
		if !wantedSlugs[slug] {
			continue
		}
		switch e.GetEvent() {
		case "review_requested":
			surviving[slug] = e.CreatedAt.Time
		case "review_request_removed":
			delete(surviving, slug)
		}
	}

	var earliest time.Time
	for _, t := range surviving {
		if earliest.IsZero() || t.Before(earliest) {
			earliest = t
		}
	}
	return earliest
}

// fetchPRTimeline returns every timeline event for (owner, repo, prNumber), paging until done.
// Pulled out of the digest's backfill so it can be cached at a higher level (one fetch per PR
// even when many reviewers on the same PR need backfilling).
func fetchPRTimeline(ctx context.Context, gh *github.Client, owner, repo string, prNumber int) ([]*github.Timeline, error) {
	if gh == nil || owner == "" || repo == "" || prNumber == 0 {
		return nil, nil
	}

	var events []*github.Timeline
	opts := &github.ListOptions{PerPage: 100}
	for {
		page, resp, err := gh.Issues.ListIssueTimeline(ctx, owner, repo, prNumber, opts)
		if err != nil {
			return nil, err
		}
		events = append(events, page...)
		if resp == nil || resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return events, nil
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
		eff := p.effectiveReviewSLAStart(prRefFromIssue(d.Issue, baseURL), reviewerLogin)
		if eff.IsZero() {
			continue
		}
		s := eff.Time.UTC().Format(time.RFC3339)
		d.ReviewSLAStartAt = &s
	}
}
