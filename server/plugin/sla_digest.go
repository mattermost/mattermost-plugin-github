// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v54/github"

	"github.com/mattermost/mattermost-plugin-github/server/plugin/graphql"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/mattermost/mattermost/server/public/pluginapi/cluster"
)

const (
	// slaDigestDayKVKey stores the last local calendar day (server timezone) we posted or skipped the digest.
	slaDigestDayKVKey = "github_sla_digest_local_day"
	slaDigestMutexKey = "github_sla_digest_mutex"
	// slaDigestMaxMessageRunes caps the channel digest below Mattermost post limits (64K runes).
	slaDigestMaxMessageRunes = 64000
	slaDigestClippedMarker   = "\n\n_This digest was clipped due to message size limits._"
)

// slaDigestEntry is one (reviewer, PR) pair past SLA. Stored as separate fields rather than
// a pre-baked single line so buildSLADigestMessage can group entries by reviewer within each
// bucket and avoid repeating the @-mention on every row.
type slaDigestEntry struct {
	DaysOverdue     int
	ReviewerDisplay string // e.g. "@harrison (hmhealey)" or "(not connected) - hmhealey"
	Body            string // e.g. "mattermost/mattermost - [Fix race condition](url)"
}

// maybePostDailyOverdueSLADigest posts one aggregated message per local calendar day to the
// configured channel (at most once daily; see runSLADigestScheduler).
func (p *Plugin) maybePostDailyOverdueSLADigest(ctx context.Context) {
	cfg := p.getConfiguration()
	if cfg.OverdueReviewsChannelID == "" || cfg.ReviewTargetDays <= 0 {
		return
	}

	day := time.Now().In(time.Local).Format("2006-01-02")
	var marker []byte
	if err := p.store.Get(slaDigestDayKVKey, &marker); err != nil {
		p.client.Log.Warn("Failed to read SLA digest day marker", "key", slaDigestDayKVKey, "error", err.Error())
		return
	}
	if string(marker) == day {
		return
	}

	m, err := cluster.NewMutex(p.API, slaDigestMutexKey)
	if err != nil {
		p.client.Log.Warn("Failed to create mutex for SLA digest", "error", err.Error())
		return
	}
	m.Lock()
	defer m.Unlock()

	if err := p.store.Get(slaDigestDayKVKey, &marker); err != nil {
		p.client.Log.Warn("Failed to read SLA digest day marker after lock", "key", slaDigestDayKVKey, "error", err.Error())
		return
	}
	if string(marker) == day {
		return
	}

	entries, ok := p.collectAllOverdueSLAItems(ctx)
	if !ok {
		// Distinguishes "digest could not complete a real scan" (config issue, no service user,
		// or every configured org's GraphQL fetch failed) from "scan ran and found nothing
		// overdue." We deliberately do NOT advance slaDigestDayKVKey here so the 5-minute
		// scheduler retries within the same local day instead of skipping until tomorrow.
		return
	}
	if len(entries) == 0 {
		if _, err := p.store.Set(slaDigestDayKVKey, []byte(day)); err != nil {
			p.client.Log.Warn("Failed to store SLA digest day marker", "error", err.Error())
		}
		return
	}

	msg := clipSLADigestMessage(buildSLADigestMessage(entries, cfg.ReviewTargetDays))
	post := &model.Post{
		ChannelId: cfg.OverdueReviewsChannelID,
		UserId:    p.BotUserID,
		Message:   msg,
	}
	if err := p.client.Post.CreatePost(post); err != nil {
		p.client.Log.Warn("Failed to post SLA digest to channel", "error", err.Error())
		return
	}

	if _, err := p.store.Set(slaDigestDayKVKey, []byte(day)); err != nil {
		p.client.Log.Warn("Failed to store SLA digest day marker", "error", err.Error())
	}
}

// resolveReviewerDisplayName turns a GitHub login into the digest's reviewer column.
// Connected users get an actual Mattermost @-mention so the post notifies them; unconnected
// users render as "(not connected) - <githublogin>" so admins can see who is missing.
func (p *Plugin) resolveReviewerDisplayName(githubLogin string) string {
	if githubLogin == "" {
		return ""
	}
	mmUsername := p.getGitHubToUsernameMapping(githubLogin)
	if mmUsername != "" {
		return fmt.Sprintf("@%s (%s)", mmUsername, githubLogin)
	}
	return fmt.Sprintf("(not connected) - %s", githubLogin)
}

// pickServiceGitHubUser returns the connected user the digest uses as a service caller for
// org-wide GitHub queries. When DigestServiceUsername is set, only that Mattermost user is
// considered; otherwise keys are sorted before iteration so the choice is deterministic
// across runs (and across cluster nodes).
//
// Returns nil when no usable connected user is available; the digest cannot run in that case.
func (p *Plugin) pickServiceGitHubUser(ctx context.Context) *GitHubUserInfo {
	if configured := strings.TrimSpace(p.getConfiguration().DigestServiceUsername); configured != "" {
		if ctx.Err() != nil {
			return nil
		}
		user, err := p.client.User.GetByUsername(configured)
		if err != nil {
			p.client.Log.Warn("SLA digest configured service user not found",
				"username", configured, "error", err.Error())
			return nil
		}
		if user == nil {
			p.client.Log.Warn("SLA digest configured service user not found", "username", configured)
			return nil
		}
		ghInfo, apiErr := p.getGitHubUserInfo(user.Id)
		if apiErr != nil || ghInfo == nil {
			p.client.Log.Warn("SLA digest configured service user is not connected to GitHub",
				"username", configured, "user_id", user.Id)
			return nil
		}
		p.client.Log.Info("SLA digest using configured service user",
			"mattermost_username", configured,
			"github_username", ghInfo.GitHubUsername,
			"user_id", user.Id)
		return ghInfo
	}

	checker := func(key string) (bool, error) {
		return strings.HasSuffix(key, githubTokenKey), nil
	}

	var allKeys []string
	for page := 0; ; page++ {
		if ctx.Err() != nil {
			return nil
		}
		keys, err := p.store.ListKeys(page, keysPerPage, pluginapi.WithChecker(checker))
		if err != nil {
			p.client.Log.Warn("SLA digest failed to list connected users", "error", err.Error())
			return nil
		}
		allKeys = append(allKeys, keys...)
		if len(keys) < keysPerPage {
			break
		}
	}
	sort.Strings(allKeys)

	for _, key := range allKeys {
		if ctx.Err() != nil {
			return nil
		}
		userID := strings.TrimSuffix(key, githubTokenKey)
		ghInfo, apiErr := p.getGitHubUserInfo(userID)
		if apiErr != nil || ghInfo == nil {
			continue
		}
		return ghInfo
	}
	return nil
}

// collectAllOverdueSLAItems returns the digest's overdue entries along with an "ok" flag the
// caller uses to decide whether to advance the daily marker. ok is false when the digest
// could not complete a real scan (no orgs configured, no connected service user, or every
// configured org's GraphQL fetch failed); the caller should retry on the next scheduler tick
// rather than treat that as "ran successfully and found nothing." A successful scan returns
// ok=true even when entries is empty.
func (p *Plugin) collectAllOverdueSLAItems(ctx context.Context) ([]slaDigestEntry, bool) {
	config := p.getConfiguration()
	targetDays := config.ReviewTargetDays
	orgList := config.getOrganizations()
	now := time.Now()

	if len(orgList) == 0 {
		p.client.Log.Warn("SLA digest cannot run without configured organizations (System Console -> Plugins -> GitHub -> GitHub Organizations)")
		return nil, false
	}

	serviceUser := p.pickServiceGitHubUser(ctx)
	if serviceUser == nil {
		p.client.Log.Warn("SLA digest cannot run: no connected GitHub user available to act as the service caller")
		return nil, false
	}

	githubClient := p.githubConnectUser(ctx, serviceUser)
	graphQLClient := p.graphQLConnect(serviceUser)

	allPRs, anyOrgOK := p.fetchAllOrgOpenPRs(ctx, graphQLClient, orgList)
	if !anyOrgOK {
		p.client.Log.Warn("SLA digest cannot run: no configured organization returned a successful PR search")
		return nil, false
	}
	if len(allPRs) == 0 {
		return nil, true
	}

	resolveTeam := newTeamMemberResolver(ctx, githubClient, p.client.Log)
	resolveSLAStart, summarizeSLAStart := p.newDigestSLAStartResolver(ctx, githubClient)
	defer summarizeSLAStart()

	out := make([]slaDigestEntry, 0, len(allPRs))
	seen := make(map[string]bool)
	for _, pr := range allPRs {
		if ctx.Err() != nil {
			// A canceled context means the scan was interrupted partway through (e.g.
			// scheduler shutdown). Whatever we've accumulated is a partial view, so we
			// must not let the caller treat it as a real scan and advance the day
			// marker. Return ok=false so the next scheduler tick retries.
			return nil, false
		}
		ref := prRef{
			Owner:     pr.Owner,
			Repo:      pr.Repo,
			Number:    pr.Number,
			CreatedAt: github.Timestamp{Time: pr.CreatedAt},
		}
		for _, rr := range gatherReviewersForPR(pr, resolveTeam) {
			entry := p.evaluateOverdueForReviewer(ref, pr, rr, targetDays, now, seen, resolveSLAStart)
			if entry != nil {
				out = append(out, *entry)
			}
		}
	}
	return out, true
}

// fetchAllOrgOpenPRs runs the org-wide PR search once per configured org, logging and skipping
// orgs that fail rather than aborting the whole digest. Returns the combined PR list and a
// flag that is true iff every configured org was visited and at least one returned
// successfully (a zero-PR org is still a success). A context cancellation mid-iteration
// forces anyOK to false: the caller uses anyOK to distinguish a "real but quiet scan" from
// an interrupted one, and a partial walk is the latter even if some orgs already responded.
func (p *Plugin) fetchAllOrgOpenPRs(ctx context.Context, graphQLClient *graphql.Client, orgList []string) (allPRs []graphql.DigestPR, anyOK bool) {
	for _, org := range orgList {
		if ctx.Err() != nil {
			return nil, false
		}
		prs, err := graphQLClient.GetOpenPRsWithRequestedReviewers(ctx, org)
		if err != nil {
			p.client.Log.Warn("SLA digest org PR fetch failed", "org", org, "error", err.Error())
			continue
		}
		anyOK = true
		allPRs = append(allPRs, prs...)
	}
	return allPRs, anyOK
}

// newTeamMemberResolver returns a closure that expands org/team references to member logins,
// memoizing the result so the same team is fetched at most once per digest run. Keys are
// lowercased so case differences in the GraphQL response don't fragment the cache.
func newTeamMemberResolver(ctx context.Context, githubClient *github.Client, log pluginapi.LogService) func(graphql.DigestTeamRef) []string {
	cache := make(map[string][]string)
	return func(team graphql.DigestTeamRef) []string {
		key := strings.ToLower(team.Org + "/" + team.Slug)
		if members, ok := cache[key]; ok {
			return members
		}
		var members []string
		opts := &github.TeamListTeamMembersOptions{ListOptions: github.ListOptions{PerPage: 100}}
		for {
			page, resp, err := githubClient.Teams.ListTeamMembersBySlug(ctx, team.Org, team.Slug, opts)
			if err != nil {
				log.Debug("SLA digest team expansion failed", "team", key, "error", err.Error())
				break
			}
			for _, u := range page {
				if login := u.GetLogin(); login != "" {
					members = append(members, login)
				}
			}
			if resp == nil || resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
		}
		cache[key] = members
		return members
	}
}

// reviewerRequest preserves a reviewer login alongside the team(s) they were added through, so
// the SLA backfill can fall back to a team-scoped review_requested timeline event when the
// reviewer has no user-scoped request. Direct (non-team) requests carry an empty Teams slice.
//
// Each login appears at most once: a user who is both directly requested and a member of one
// or more requested teams aggregates into a single entry whose Teams slice records every team
// they were also part of (used as a tie-break only when the user-scoped lookup is empty).
type reviewerRequest struct {
	Login string
	Teams []graphql.DigestTeamRef
}

// gatherReviewersForPR collapses a PR's directly-requested users and all members of its
// requested teams into a deduplicated reviewer set, preserving each reviewer's team origin so
// the SLA backfill can use a team-scoped review_requested event when no user-scoped event is
// available. Logins are matched case-insensitively for deduplication; the first-seen casing is
// preserved on the returned struct.
func gatherReviewersForPR(pr graphql.DigestPR, resolveTeam func(graphql.DigestTeamRef) []string) []reviewerRequest {
	byLogin := make(map[string]int)
	out := make([]reviewerRequest, 0, len(pr.RequestedUsers))
	// add is the single chokepoint for both call paths below (direct RequestedUsers and
	// team-expanded logins). The empty-login guard belongs here, not at the loop sites,
	// because it must apply uniformly to both: the GraphQL layer (digest_query.go) already
	// drops empty user logins, but a third-party resolveTeam implementation could surface
	// them and we'd still want them filtered before they leak into out[idx].Login.
	add := func(login string, team *graphql.DigestTeamRef) {
		if login == "" {
			return
		}
		key := strings.ToLower(login)
		idx, ok := byLogin[key]
		if !ok {
			out = append(out, reviewerRequest{Login: login})
			idx = len(out) - 1
			byLogin[key] = idx
		}
		if team != nil {
			out[idx].Teams = append(out[idx].Teams, *team)
		}
	}
	for _, login := range pr.RequestedUsers {
		add(login, nil)
	}
	for _, team := range pr.RequestedTeams {
		for _, login := range resolveTeam(team) {
			add(login, &team)
		}
	}
	return out
}

// newDigestSLAStartResolver returns a closure that resolves a reviewer's SLA-start time for
// the digest, with three layers of lookup:
//
//  1. The KV record from the live review_requested webhook (free).
//  2. A user-scoped review_requested event from the PR timeline. Timeline pages are fetched
//     at most once per PR per digest run and shared across reviewers on the same PR.
//  3. For reviewers added via team membership only, the earliest surviving team-scoped
//     review_requested event. This avoids overstating overdue days for users who are pending
//     solely because their team was requested.
//
// Resolved times (from either timeline path) are written back to KV under the user-scoped
// key so subsequent runs hit the fast path. Falls back to the PR open time when no
// review_requested event is found, matching the read-only effectiveReviewSLAStart contract.
//
// Also returns a summarize() callback the caller must invoke once the resolver is no longer
// in use (typically via defer). It logs cumulative cache hit / timeline-fetch counts at Info
// level so admins can verify the backfill is healthy across runs (see Day-1-vs-steady-state
// patterns in the docs). The callback is a no-op when neither counter moved, so digests that
// reach this point but never actually call the resolver (e.g. PRs with no requested
// reviewers) don't emit a misleading "0,0" line.
func (p *Plugin) newDigestSLAStartResolver(ctx context.Context, gh *github.Client) (resolve func(prRef, reviewerRequest) github.Timestamp, summarize func()) {
	type prKey struct {
		owner string
		repo  string
		num   int
	}
	timelineCache := make(map[prKey][]*github.Timeline)
	timelineFetched := make(map[prKey]bool)
	var fetched, hits int

	resolve = func(pr prRef, rr reviewerRequest) github.Timestamp {
		if pr.Owner == "" || pr.Repo == "" || pr.Number == 0 {
			return pr.CreatedAt
		}
		if t := p.getReviewSLAStartTime(pr.Owner, pr.Repo, pr.Number, rr.Login); !t.IsZero() {
			hits++
			return github.Timestamp{Time: t}
		}
		if gh == nil {
			return pr.CreatedAt
		}

		key := prKey{pr.Owner, pr.Repo, pr.Number}
		events, ok := timelineCache[key]
		if !ok && !timelineFetched[key] {
			var err error
			events, err = fetchPRTimeline(ctx, gh, pr.Owner, pr.Repo, pr.Number)
			if err != nil {
				p.client.Log.Debug("SLA digest timeline fetch failed; falling back to PR created_at",
					"owner", pr.Owner, "repo", pr.Repo, "pr", pr.Number, "error", err.Error())
				events = nil
			}
			timelineCache[key] = events
			timelineFetched[key] = true
			fetched++
		}

		if found := findMostRecentReviewRequestTime(events, rr.Login); !found.IsZero() {
			p.storeReviewSLAStart(pr.Owner, pr.Repo, pr.Number, rr.Login, found)
			return github.Timestamp{Time: found.UTC()}
		}
		// No user-scoped event. For team-membership requests, anchor to the earliest still-
		// surviving team request: the user has been on the hook since the first such ask.
		if found := findEarliestSurvivingTeamRequestTime(events, rr.Teams); !found.IsZero() {
			p.storeReviewSLAStart(pr.Owner, pr.Repo, pr.Number, rr.Login, found)
			return github.Timestamp{Time: found.UTC()}
		}
		return pr.CreatedAt
	}

	summarize = func() {
		if fetched == 0 && hits == 0 {
			return
		}
		p.client.Log.Info("SLA digest backfill summary",
			"timeline_pages_fetched", fetched,
			"kv_hits", hits)
	}

	return resolve, summarize
}

// evaluateOverdueForReviewer returns a digest entry for (pr, reviewer) when the reviewer is
// past SLA, or nil when not overdue, already accounted for, or missing identity. seen is
// mutated to record the (pr, login) pair. gatherReviewersForPR already deduplicates a single
// PR's reviewer set, but seen guards against any unexpected duplicates that slip through
// (e.g. case-mismatched names from different code paths).
func (p *Plugin) evaluateOverdueForReviewer(
	ref prRef,
	pr graphql.DigestPR,
	rr reviewerRequest,
	targetDays int,
	now time.Time,
	seen map[string]bool,
	resolveSLAStart func(prRef, reviewerRequest) github.Timestamp,
) *slaDigestEntry {
	if rr.Login == "" {
		return nil
	}
	dedupeKey := pr.Owner + "/" + pr.Repo + "#" + strconv.Itoa(pr.Number) + "@" + strings.ToLower(rr.Login)
	if seen[dedupeKey] {
		return nil
	}
	seen[dedupeKey] = true

	slaStart := resolveSLAStart(ref, rr)
	diff := slaCalendarDiffDays(slaStart, targetDays, now)
	if diff >= 0 {
		return nil
	}

	reviewerDisplay := p.resolveReviewerDisplayName(rr.Login)
	body := formatChannelOverduePRBody(pr.Title, pr.URL, p.getConfiguration().getBaseURL())
	return &slaDigestEntry{DaysOverdue: -diff, ReviewerDisplay: reviewerDisplay, Body: body}
}

// slaBuckets enumerates the digest's overdue buckets in display order (most overdue first).
// Each bucket emits a header even if empty buckets in between are skipped.
var slaBuckets = []struct {
	label string
	min   int // inclusive; -1 for open-ended upper bucket
	max   int // inclusive; -1 for open-ended upper bucket
}{
	{label: "More than 1 year overdue", min: 366, max: -1},
	{label: "91-365 days overdue", min: 91, max: 365},
	{label: "31-90 days overdue", min: 31, max: 90},
	{label: "15-30 days overdue", min: 15, max: 30},
	{label: "8-14 days overdue", min: 8, max: 14},
	{label: "4-7 days overdue", min: 4, max: 7},
	{label: "1-3 days overdue", min: 1, max: 3},
}

// slaBucketIndex returns the index into slaBuckets for the given days-overdue value, or -1 if not overdue.
func slaBucketIndex(daysOverdue int) int {
	if daysOverdue < 1 {
		return -1
	}
	for i, b := range slaBuckets {
		if b.max == -1 {
			if daysOverdue >= b.min {
				return i
			}
			continue
		}
		if daysOverdue >= b.min && daysOverdue <= b.max {
			return i
		}
	}
	return -1
}

// reviewerBucketGroup is one reviewer's overdue PRs within a single bucket, used by
// buildSLADigestMessage to render `- <reviewer>\n  - <body>\n  - <body>` instead of
// repeating the reviewer prefix on every PR row.
type reviewerBucketGroup struct {
	ReviewerDisplay string
	Bodies          []string
}

// groupBucketEntriesByReviewer collapses a bucket's entries into one group per reviewer,
// sorts bodies within each group, and sorts the groups themselves by reviewer display
// (case-insensitive). The deterministic ordering is required for stable channel posts and
// snapshot-style assertions; map iteration order alone would leave the output flaky.
func groupBucketEntriesByReviewer(entries []slaDigestEntry) []reviewerBucketGroup {
	idxByReviewer := make(map[string]int, len(entries))
	out := make([]reviewerBucketGroup, 0, len(entries))
	for _, e := range entries {
		idx, ok := idxByReviewer[e.ReviewerDisplay]
		if !ok {
			out = append(out, reviewerBucketGroup{ReviewerDisplay: e.ReviewerDisplay})
			idx = len(out) - 1
			idxByReviewer[e.ReviewerDisplay] = idx
		}
		out[idx].Bodies = append(out[idx].Bodies, e.Body)
	}
	for i := range out {
		sort.Strings(out[i].Bodies)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return strings.ToLower(out[i].ReviewerDisplay) < strings.ToLower(out[j].ReviewerDisplay)
	})
	return out
}

func buildSLADigestMessage(entries []slaDigestEntry, targetDays int) string {
	bucketEntries := make([][]slaDigestEntry, len(slaBuckets))
	for _, e := range entries {
		idx := slaBucketIndex(e.DaysOverdue)
		if idx < 0 {
			continue
		}
		bucketEntries[idx] = append(bucketEntries[idx], e)
	}

	var b strings.Builder
	if targetDays > 0 {
		unit := "days"
		if targetDays == 1 {
			unit = "day"
		}
		fmt.Fprintf(&b, "### Pull request reviews past SLA (target: %d %s from most recent review request)\n\n", targetDays, unit)
	} else {
		b.WriteString("### Pull request reviews past SLA\n\n")
	}
	for i, bucket := range slaBuckets {
		bes := bucketEntries[i]
		if len(bes) == 0 {
			continue
		}
		fmt.Fprintf(&b, "#### %s\n", bucket.label)
		for _, g := range groupBucketEntriesByReviewer(bes) {
			fmt.Fprintf(&b, "- %s\n", g.ReviewerDisplay)
			for _, body := range g.Bodies {
				fmt.Fprintf(&b, "  - %s\n", body)
			}
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func clipSLADigestMessage(message string) string {
	return truncateMessageAtRunes(message, slaDigestMaxMessageRunes, slaDigestClippedMarker)
}
