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

func TestGetIssueNumberFromURL(t *testing.T) {
	tcs := []struct {
		Text     string
		Expected string
	}{
		{Text: "https://github.com/jwilander/mattermost-webapp/pull/13", Expected: "13"},
		{Text: "https://github.com/jwilander/mattermost-webapp/issues/42", Expected: "42"},
		{Text: "https://github.com/jwilander/mattermost-webapp", Expected: ""},
	}

	for _, tc := range tcs {
		assert.Equal(t, tc.Expected, getIssueNumberFromURL(tc.Text))
	}
}
