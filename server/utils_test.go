package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseGitHubUsernameFromText(t *testing.T) {
	tcs := []struct {
		Text     string
		Expected []string
	}{
		{Text: "@jwilander", Expected: []string{"jwilander"}},
		{Text: "@jwilander.", Expected: []string{"jwilander"}},
		{Text: ".@jwilander", Expected: []string{"jwilander"}},
		{Text: " @jwilander ", Expected: []string{"jwilander"}},
		{Text: "@1jwilander", Expected: []string{"1jwilander"}},
		{Text: "@j", Expected: []string{"j"}},
		{Text: "@", Expected: []string{}},
		{Text: "", Expected: []string{}},
		{Text: "jwilander", Expected: []string{}},
		{Text: "@jwilander-", Expected: []string{}},
		{Text: "@-jwilander", Expected: []string{}},
		{Text: "@jwil--ander", Expected: []string{}},
		{Text: "@jwilander @jwilander2", Expected: []string{"jwilander", "jwilander2"}},
		{Text: "@jwilander2 @jwilander", Expected: []string{"jwilander2", "jwilander"}},
		{Text: "Hey @jwilander and @jwilander2!", Expected: []string{"jwilander", "jwilander2"}},
		{Text: "@jwilander @jwilan--der2", Expected: []string{"jwilander"}},
	}

	for _, tc := range tcs {
		assert.Equal(t, tc.Expected, parseGitHubUsernamesFromText(tc.Text))
	}
}

func TestFixGithubNotificationSubjectURL(t *testing.T) {
	tcs := []struct {
		Text     string
		Expected string
	}{
		{Text: "https://api.github.com/repos/jwilander/mattermost-webapp/issues/123", Expected: "https://github.com/jwilander/mattermost-webapp/issues/123"},
		{Text: "https://api.github.com/repos/jwilander/mattermost-webapp/pulls/123", Expected: "https://github.com/jwilander/mattermost-webapp/pull/123"},
		{Text: "https://enterprise.github.com/api/v3/jwilander/mattermost-webapp/issues/123", Expected: "https://enterprise.github.com/jwilander/mattermost-webapp/issues/123"},
		{Text: "https://enterprise.github.com/api/v3/jwilander/mattermost-webapp/pull/123", Expected: "https://enterprise.github.com/jwilander/mattermost-webapp/pull/123"},
	}

	for _, tc := range tcs {
		assert.Equal(t, tc.Expected, fixGithubNotificationSubjectURL(tc.Text))
	}
}
