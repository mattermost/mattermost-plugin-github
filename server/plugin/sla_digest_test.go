// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatChannelOverdueReviewLine(t *testing.T) {
	const baseURL = "https://github.com/"

	t.Run("renders connected reviewer with @-mention and bracketed github login", func(t *testing.T) {
		line := formatChannelOverdueReviewLine(
			"@harrison (hmhealey)",
			"Fix race condition",
			"https://github.com/mattermost/mattermost/pull/12345",
			baseURL,
		)
		assert.Equal(t, "- @harrison (hmhealey) - mattermost/mattermost - [Fix race condition](https://github.com/mattermost/mattermost/pull/12345)", line)
	})

	t.Run("renders unconnected reviewer with `(not connected) - login` prefix", func(t *testing.T) {
		line := formatChannelOverdueReviewLine(
			"(not connected) - hmhealey",
			"Fix race condition",
			"https://github.com/mattermost/mattermost/pull/12345",
			baseURL,
		)
		assert.Equal(t, "- (not connected) - hmhealey - mattermost/mattermost - [Fix race condition](https://github.com/mattermost/mattermost/pull/12345)", line)
	})

	t.Run("escapes closing brackets in the title so the link does not terminate early", func(t *testing.T) {
		line := formatChannelOverdueReviewLine(
			"hmhealey",
			"[MM-12345] Fix [thing]",
			"https://github.com/mattermost/mattermost/pull/1",
			baseURL,
		)
		// Only `]` needs escaping inside a markdown link's display text; `[` is allowed.
		assert.Contains(t, line, `[[MM-12345\] Fix [thing\]](https://github.com/mattermost/mattermost/pull/1)`)
	})

	t.Run("falls back to raw URL as the repo display when owner/repo cannot be parsed", func(t *testing.T) {
		line := formatChannelOverdueReviewLine(
			"hmhealey",
			"Title",
			"https://example.invalid/something/odd",
			baseURL,
		)
		assert.Contains(t, line, "https://example.invalid/something/odd - [Title]")
	})

	t.Run("truncates very long titles before linking", func(t *testing.T) {
		title := strings.Repeat("a", 250)
		line := formatChannelOverdueReviewLine(
			"hmhealey",
			title,
			"https://github.com/mattermost/mattermost/pull/1",
			baseURL,
		)
		assert.Contains(t, line, "...](https://github.com/mattermost/mattermost/pull/1)")
		// Title body inside the link should not exceed 200 'a's followed by the ellipsis.
		assert.NotContains(t, line, strings.Repeat("a", 201))
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
	t.Run("groups entries into the correct buckets in display order", func(t *testing.T) {
		entries := []slaDigestEntry{
			{DaysOverdue: 2, Line: "- a"},
			{DaysOverdue: 5, Line: "- b"},
			{DaysOverdue: 12, Line: "- c"},
			{DaysOverdue: 100, Line: "- d"},
			{DaysOverdue: 400, Line: "- e"},
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
			{DaysOverdue: 0, Line: "- skipped"},
			{DaysOverdue: -2, Line: "- skipped-too"},
			{DaysOverdue: 1, Line: "- kept"},
		}
		msg := buildSLADigestMessage(entries, 3)
		assert.Contains(t, msg, "- kept")
		assert.NotContains(t, msg, "- skipped")
		assert.NotContains(t, msg, "- skipped-too")
	})

	t.Run("singular target days uses 'day'", func(t *testing.T) {
		msg := buildSLADigestMessage([]slaDigestEntry{{DaysOverdue: 1, Line: "- x"}}, 1)
		assert.Contains(t, msg, "target: 1 day from")
	})

	t.Run("zero target days falls back to plain header", func(t *testing.T) {
		msg := buildSLADigestMessage([]slaDigestEntry{{DaysOverdue: 1, Line: "- x"}}, 0)
		assert.True(t, strings.HasPrefix(msg, "### Pull request reviews past SLA\n"))
	})

	t.Run("lines within a bucket are sorted alphabetically", func(t *testing.T) {
		entries := []slaDigestEntry{
			{DaysOverdue: 2, Line: "- zeta"},
			{DaysOverdue: 2, Line: "- alpha"},
			{DaysOverdue: 2, Line: "- mu"},
		}
		msg := buildSLADigestMessage(entries, 3)
		ai := strings.Index(msg, "- alpha")
		mi := strings.Index(msg, "- mu")
		zi := strings.Index(msg, "- zeta")
		assert.True(t, ai >= 0 && mi > ai && zi > mi, "expected alpha < mu < zeta in output")
	})
}
