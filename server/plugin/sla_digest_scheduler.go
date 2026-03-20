// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"context"
	"time"
)

// runSLADigestScheduler loops forever: when SLA channel + target days are configured, it sleeps
// until the next midnight in the server's local timezone, then runs the overdue digest. This matches
// a simple "once per day" schedule without tying to user logins.
func (p *Plugin) runSLADigestScheduler() {
	for {
		if !p.slaDigestSchedulingEnabled() {
			time.Sleep(5 * time.Minute)
			continue
		}

		d := durationUntilNextLocalMidnight()
		time.Sleep(d)

		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
		p.maybePostDailyOverdueSLADigest(ctx)
		cancel()

		time.Sleep(2 * time.Second)
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
