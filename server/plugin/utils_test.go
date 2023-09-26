package plugin

import (
	"fmt"
	"testing"

	"github.com/google/go-github/v41/github"
	"github.com/mattermost/mattermost-server/v6/model"
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
		IssueNum string
	}{
		{Text: "https://api.github.com/repos/jwilander/mattermost-webapp/issues/123", Expected: "https://github.com/jwilander/mattermost-webapp/issues/123"},
		{Text: "https://api.github.com/repos/jwilander/mattermost-webapp/pulls/123", Expected: "https://github.com/jwilander/mattermost-webapp/pull/123"},
		{Text: "https://enterprise.github.com/api/v3/jwilander/mattermost-webapp/issues/123", Expected: "https://enterprise.github.com/jwilander/mattermost-webapp/issues/123"},
		{Text: "https://enterprise.github.com/api/v3/jwilander/mattermost-webapp/pull/123", Expected: "https://enterprise.github.com/jwilander/mattermost-webapp/pull/123"},
		{Text: "https://api.github.com/repos/mattermost/mattermost-server/commits/cc6c385d3e8903546fc6fc856bf468ad09b70913", Expected: "https://github.com/mattermost/mattermost-server/commit/cc6c385d3e8903546fc6fc856bf468ad09b70913"},
		{Text: "https://api.github.com/repos/user/rate_my_cakes/issues/comments/655139214", Expected: "https://github.com/user/rate_my_cakes/issues/4#issuecomment-655139214", IssueNum: "4"},
	}

	for _, tc := range tcs {
		assert.Equal(t, tc.Expected, fixGithubNotificationSubjectURL(tc.Text, tc.IssueNum))
	}
}

func TestParseOwnerAndRepo(t *testing.T) {
	tcs := []struct {
		Full          string
		BaseURL       string
		ExpectedOwner string
		ExpectedRepo  string
	}{
		{Full: "mattermost", BaseURL: "", ExpectedOwner: "mattermost", ExpectedRepo: ""},
		{Full: "mattermost", BaseURL: "https://github.com/", ExpectedOwner: "mattermost", ExpectedRepo: ""},
		{Full: "https://example.org/mattermost", BaseURL: "https://example.org/", ExpectedOwner: "mattermost", ExpectedRepo: ""},
		{Full: "https://github.com/mattermost", BaseURL: "https://github.com/", ExpectedOwner: "mattermost", ExpectedRepo: ""},
		{Full: "mattermost/mattermost-server", BaseURL: "", ExpectedOwner: "mattermost", ExpectedRepo: "mattermost-server"},
		{Full: "mattermost/mattermost-server", BaseURL: "https://github.com/", ExpectedOwner: "mattermost", ExpectedRepo: "mattermost-server"},
		{Full: "https://example.org/mattermost/mattermost-server", BaseURL: "https://example.org/", ExpectedOwner: "mattermost", ExpectedRepo: "mattermost-server"},
		{Full: "https://github.com/mattermost/mattermost-server", BaseURL: "https://github.com/", ExpectedOwner: "mattermost", ExpectedRepo: "mattermost-server"},
		{Full: "", BaseURL: "", ExpectedOwner: "", ExpectedRepo: ""},
		{Full: "mattermost/mattermost/invalid_repo_url", BaseURL: "", ExpectedOwner: "mattermost", ExpectedRepo: "mattermost"},
		{Full: "https://github.com/mattermost/mattermost/invalid_repo_url", BaseURL: "https://github.com/", ExpectedOwner: "mattermost", ExpectedRepo: "mattermost"},
	}

	for i, tc := range tcs {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			owner, repo := parseOwnerAndRepo(tc.Full, tc.BaseURL)

			assert.Equal(t, tc.ExpectedOwner, owner)
			assert.Equal(t, tc.ExpectedRepo, repo)
		})
	}
}

func TestIsFlag(t *testing.T) {
	tcs := []struct {
		Text     string
		Expected bool
	}{
		{Text: "--test-flag", Expected: true},
		{Text: "--testFlag", Expected: true},
		{Text: "test-no-flag", Expected: false},
		{Text: "testNoFlag", Expected: false},
		{Text: "test no flag", Expected: false},
	}

	for _, tc := range tcs {
		assert.Equal(t, tc.Expected, isFlag(tc.Text))
	}
}

func TestParseFlag(t *testing.T) {
	tcs := []struct {
		Text     string
		Expected string
	}{
		{Text: "--test-flag", Expected: "test-flag"},
		{Text: "--testFlag", Expected: "testFlag"},
		{Text: "testNoFlag", Expected: "testNoFlag"},
	}

	for _, tc := range tcs {
		assert.Equal(t, tc.Expected, parseFlag(tc.Text))
	}
}

func TestContainsValue(t *testing.T) {
	tcs := []struct {
		List     []string
		Value    string
		Expected bool
	}{
		{List: []string{"value1", "value2"}, Value: "value1", Expected: true},
		{List: []string{}, Value: "value1", Expected: false},
		{List: []string{"value1", "value2"}, Value: "value2", Expected: true},
	}

	for _, tc := range tcs {
		assert.Equal(t, tc.Expected, containsValue(tc.List, tc.Value))
	}
}

func TestGetLineNumbers(t *testing.T) {
	tcs := []struct {
		input      string
		start, end int
	}{
		{
			input: "L19",
			start: 16,
			end:   22,
		}, {
			input: "L19-L23",
			start: 19,
			end:   23,
		}, {
			input: "L23-L19",
			start: -1,
			end:   -1,
		}, {
			input: "L",
			start: -1,
			end:   -1,
		}, {
			input: "bad",
			start: -1,
			end:   -1,
		}, {
			input: "L99-",
			start: 99,
			end:   -1,
		}, {
			input: "L2",
			start: 0,
			end:   5,
		},
	}
	for _, tc := range tcs {
		start, end := getLineNumbers(tc.input)
		assert.Equalf(t, tc.start, start, "unexpected start index for getLineNumbers(%q)", tc.input)
		assert.Equalf(t, tc.end, end, "unexpected end index for getLineNumbers(%q)", tc.input)
	}
}

func TestInsideLink(t *testing.T) {
	tcs := []struct {
		input    string
		index    int
		expected bool
	}{
		{
			input:    "[text](link)",
			index:    7,
			expected: true,
		}, {
			input:    "[text]( link space)",
			index:    8,
			expected: true,
		}, {
			input:    "text](link",
			index:    6,
			expected: true,
		}, {
			input:    "text] (link)",
			index:    7,
			expected: false,
		}, {
			input:    "text](link)",
			index:    6,
			expected: true,
		}, {
			input:    "link",
			index:    0,
			expected: false,
		}, {
			input:    " (link)",
			index:    2,
			expected: false,
		},
	}

	for _, tc := range tcs {
		assert.Equalf(t, tc.expected, isInsideLink(tc.input, tc.index), "unexpected result for isInsideLink(%q, %d)", tc.input, tc.index)
	}
}

func TestGetToDoDisplayText(t *testing.T) {
	type input struct {
		title      string
		url        string
		notifType  string
		repository *github.Repository
	}
	tcs := []struct {
		name string
		in   input
		want string
	}{
		{
			name: "title shorter than threshold, single-word repo name & empty notification type",
			in: input{
				"Issue title with less than 80 characters",
				"https://github.com/mattermost/repo/issues/42",
				"",
				nil,
			},
			want: "* [mattermost/repo](https://github.com/mattermost/repo) [Issue title with less than 80 characters](https://github.com/mattermost/repo/issues/42)\n",
		},
		{
			name: "title longer than threshold, multi-word repo name & Issue notification type",
			in: input{
				"This is an issue title which has with more than 80 characters and is completely random",
				"https://github.com/mattermost/mattermost-plugin-github/issues/42",
				"Issue",
				nil,
			},
			want: "* [mattermost/...github](https://github.com/mattermost/mattermost-plugin-github) Issue [This is an issue title which has with more than 80 characters and is completely...](https://github.com/mattermost/mattermost-plugin-github/issues/42)\n",
		},
		{
			name: "title longer than threshold, multi-word repo name & Issue notification type",
			in: input{
				"Test discussion title!",
				"",
				"Discussion",
				&github.Repository{
					HTMLURL: model.NewString("https://github.com/mattermost/mattermost-plugin-github"),
					Owner: &github.User{
						Login: model.NewString("mattermost"),
					},
					Name: model.NewString("mattermost-plugin-github"),
				},
			},
			want: "* [mattermost/...github](https://github.com/mattermost/mattermost-plugin-github) Discussion : Test discussion title!\n",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := getToDoDisplayText("https://github.com/", tc.in.title, tc.in.url, tc.in.notifType, tc.in.repository)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestLastN(t *testing.T) {
	tcs := []struct {
		Text     string
		N        int
		Expected string
	}{
		{Text: "", N: -99, Expected: ""},
		{Text: "", N: -1, Expected: ""},
		{Text: "", N: 0, Expected: ""},
		{Text: "", N: 1, Expected: ""},
		{Text: "", N: 99, Expected: ""},
		{Text: "abcdef", N: 4, Expected: "**cdef"},
		{Text: "abcdefghi", N: 2, Expected: "***hi"},
		{Text: "abcdefghi", N: 0, Expected: "***"},
		{Text: "abcdefghi", N: 99, Expected: "abcdefghi"},
	}

	for _, tc := range tcs {
		assert.Equal(t, tc.Expected, lastN(tc.Text, tc.N))
	}
}
