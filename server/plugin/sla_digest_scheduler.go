// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"context"
	"time"
)

// runSLADigestScheduler loops until ctx is cancelled: when SLA channel + target days are
// configured, it sleeps until the next midnight in the server's local timezone, then runs the
// overdue digest. This matches a simple "once per day" schedule without tying to user logins.
func (p *Plugin) runSLADigestScheduler(ctx context.Context) {
	p.client.Log.Info("SLA digest scheduler started", "timezone", time.Local.String())
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if !p.slaDigestSchedulingEnabled() {
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Minute):
			}
			continue
		}

		d := durationUntilNextLocalMidnight()
		select {
		case <-ctx.Done():
			return
		case <-time.After(d):
		}

		digestCtx, cancel := context.WithTimeout(ctx, 45*time.Minute)
		p.maybePostDailyOverdueSLADigest(digestCtx)
		cancel()

		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func (p *Plugin) slaDigestSchedulingEnabled() bool {
	cfg := p.getConfiguration()
	return cfg.OverdueReviewsChannelID != "" && cfg.ReviewTargetDays > 0
}

func durationUntilNextLocalMidnight() time.Duration {
	loc := time.Local
	now := time.Now().In(loc)
	next := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)
	return time.Until(next)
}
