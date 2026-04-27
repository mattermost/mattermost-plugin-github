// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v54/github"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/mattermost/mattermost/server/public/pluginapi/cluster"
	"golang.org/x/oauth2"
)

const (
	// slaDigestDayKVKey stores the last local calendar day (server timezone) we posted or skipped the digest.
	slaDigestDayKVKey = "github_sla_digest_local_day"
	slaDigestMutexKey = "github_sla_digest_mutex"
)

type slaDigestEntry struct {
	DaysOverdue int
	Line        string
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

	entries := p.collectAllOverdueSLAItems(ctx)
	if len(entries) == 0 {
		if _, err := p.store.Set(slaDigestDayKVKey, []byte(day)); err != nil {
			p.client.Log.Warn("Failed to store SLA digest day marker", "error", err.Error())
		}
		return
	}

	msg := buildSLADigestMessage(entries, cfg.ReviewTargetDays)
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

func (p *Plugin) collectAllOverdueSLAItems(ctx context.Context) []slaDigestEntry {
	config := p.getConfiguration()
	targetDays := config.ReviewTargetDays
	baseURL := config.getBaseURL()
	orgList := config.getOrganizations()
	now := time.Now()

	checker := func(key string) (bool, error) {
		return strings.HasSuffix(key, githubTokenKey), nil
	}

	var out []slaDigestEntry

	for page := 0; ; page++ {
		if ctx.Err() != nil {
			return out
		}

		keys, err := p.store.ListKeys(page, keysPerPage, pluginapi.WithChecker(checker))
		if err != nil {
			p.client.Log.Warn("Failed to list keys for SLA digest", "error", err.Error())
			break
		}

		for _, key := range keys {
			if ctx.Err() != nil {
				return out
			}

			userID := strings.TrimSuffix(key, githubTokenKey)
			ghInfo, apiErr := p.getGitHubUserInfo(userID)
			if apiErr != nil || ghInfo == nil {
				time.Sleep(delayBetweenUsers)
				continue
			}

			githubClient := p.githubConnectUser(ctx, ghInfo)
			var allIssues []*github.Issue
			cErr := p.useGitHubClient(ghInfo, func(gi *GitHubUserInfo, token *oauth2.Token) error {
				query := getReviewSearchQuery(gi.GitHubUsername, orgList)
				opts := &github.SearchOptions{ListOptions: github.ListOptions{PerPage: 100}}
				for {
					result, resp, searchErr := githubClient.Search.Issues(ctx, query, opts)
					if searchErr != nil {
						return searchErr
					}
					allIssues = append(allIssues, result.Issues...)
					if resp.NextPage == 0 {
						break
					}
					opts.Page = resp.NextPage
				}
				return nil
			})
			if cErr != nil {
				p.client.Log.Debug("SLA digest skipped user review search", "user_id", userID, "error", cErr.Error())
				time.Sleep(delayBetweenUsers)
				continue
			}

			reviewerDisplay := p.resolveReviewerDisplayName(ghInfo.GitHubUsername)
			for _, pr := range allIssues {
				slaStart := p.effectiveReviewSLAStart(pr, baseURL, ghInfo.GitHubUsername)
				diff := slaCalendarDiffDays(slaStart, targetDays, now)
				if diff >= 0 {
					continue
				}
				daysOverdue := -diff
				line := formatChannelOverdueReviewLine(reviewerDisplay, pr.GetTitle(), pr.GetHTMLURL(), baseURL)
				out = append(out, slaDigestEntry{DaysOverdue: daysOverdue, Line: line})
			}

			time.Sleep(delayBetweenUsers)
		}

		if len(keys) < keysPerPage {
			break
		}
	}

	return out
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

func buildSLADigestMessage(entries []slaDigestEntry, targetDays int) string {
	bucketLines := make([][]string, len(slaBuckets))
	for _, e := range entries {
		idx := slaBucketIndex(e.DaysOverdue)
		if idx < 0 {
			continue
		}
		bucketLines[idx] = append(bucketLines[idx], e.Line)
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
		lines := bucketLines[i]
		if len(lines) == 0 {
			continue
		}
		sort.Strings(lines)
		fmt.Fprintf(&b, "#### %s\n", bucket.label)
		for _, line := range lines {
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}
