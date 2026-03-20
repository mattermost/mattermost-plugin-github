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

	msg := buildSLADigestMessage(entries)
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
		keys, err := p.store.ListKeys(page, keysPerPage, pluginapi.WithChecker(checker))
		if err != nil {
			p.client.Log.Warn("Failed to list keys for SLA digest", "error", err.Error())
			break
		}

		for _, key := range keys {
			userID := strings.TrimSuffix(key, githubTokenKey)
			ghInfo, apiErr := p.getGitHubUserInfo(userID)
			if apiErr != nil || ghInfo == nil {
				time.Sleep(delayBetweenUsers)
				continue
			}

			githubClient := p.githubConnectUser(ctx, ghInfo)
			var issueResults *github.IssuesSearchResult
			cErr := p.useGitHubClient(ghInfo, func(gi *GitHubUserInfo, token *oauth2.Token) error {
				var searchErr error
				issueResults, _, searchErr = githubClient.Search.Issues(ctx, getReviewSearchQuery(gi.GitHubUsername, orgList), &github.SearchOptions{})
				return searchErr
			})
			if cErr != nil {
				p.client.Log.Debug("SLA digest skipped user review search", "user_id", userID, "error", cErr.Error())
				time.Sleep(delayBetweenUsers)
				continue
			}

			for _, pr := range issueResults.Issues {
				slaStart := p.effectiveReviewSLAStart(pr, baseURL, ghInfo.GitHubUsername)
				diff := slaCalendarDiffDays(slaStart, targetDays, now)
				if diff >= 0 {
					continue
				}
				daysOverdue := -diff
				line := formatChannelOverdueReviewLine(ghInfo.GitHubUsername, pr.GetTitle(), pr.GetHTMLURL(), baseURL)
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

func buildSLADigestMessage(entries []slaDigestEntry) string {
	buckets := make(map[int][]string)
	for _, e := range entries {
		buckets[e.DaysOverdue] = append(buckets[e.DaysOverdue], e.Line)
	}

	days := make([]int, 0, len(buckets))
	for d := range buckets {
		days = append(days, d)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(days)))

	var b strings.Builder
	b.WriteString("### Pull request reviews past SLA\n\n")
	for _, d := range days {
		lines := buckets[d]
		sort.Strings(lines)
		unit := "days"
		if d == 1 {
			unit = "day"
		}
		fmt.Fprintf(&b, "#### %d %s overdue\n", d, unit)
		for _, line := range lines {
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}
