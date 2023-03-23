package plugin

import (
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v48/github"
	"github.com/stretchr/testify/require"
)

var repo = github.Repository{
	FullName:        sToP("mattermost-plugin-github"),
	StargazersCount: iToP(1),
	HTMLURL:         sToP("https://github.com/mattermost/mattermost-plugin-github"),
}

var pushEventRepository = github.PushEventRepository{
	FullName: sToP("mattermost-plugin-github"),
	HTMLURL:  sToP("https://github.com/mattermost/mattermost-plugin-github"),
}

var singleLabel = []*github.Label{
	{
		Name: sToP("Help Wanted"),
	},
}

var labels = []*github.Label{
	{
		Name: sToP("Help Wanted"),
	},
	{
		Name: sToP("Tech/Go"),
	},
}

var pullRequest = github.PullRequest{
	Number:    iToP(42),
	HTMLURL:   sToP("https://github.com/mattermost/mattermost-plugin-github/pull/42"),
	Title:     sToP("Leverage git-get-head"),
	CreatedAt: tToP(time.Date(2019, 04, 01, 02, 03, 04, 0, time.UTC)),
	UpdatedAt: tToP(time.Date(2019, 05, 01, 02, 03, 04, 0, time.UTC)),
	Body: sToP(`<!-- Thank you for opening this pull request-->git-get-head gets the non-sent upstream heads inside the stashed non-cleaned applied areas, and after pruning bases to many archives, you can initialize the origin of the bases.
<!-- Please make sure you have done the following :
- Added tests
- Removed console logs
-->`),
}

var pullRequestWithMentions = github.PullRequest{
	Number:    iToP(42),
	HTMLURL:   sToP("https://github.com/mattermost/mattermost-plugin-github/pull/42"),
	Title:     sToP("Leverage git-get-head"),
	CreatedAt: tToP(time.Date(2019, 04, 01, 02, 03, 04, 0, time.UTC)),
	UpdatedAt: tToP(time.Date(2019, 05, 01, 02, 03, 04, 0, time.UTC)),
	Body: sToP(`<!-- Thank you for opening this pull request-->git-get-head gets the non-sent upstream heads inside the stashed non-cleaned applied areas, and after pruning bases to many archives, you can initialize the origin of the bases.
` + gitHubMentions + `
<!-- Please make sure you have done the following :
- Added tests
- Removed console logs
-->`),
}

var pullRequestWithLabelAndAssignee = github.PullRequest{
	Number:    iToP(42),
	HTMLURL:   sToP("https://github.com/mattermost/mattermost-plugin-github/pull/42"),
	Title:     sToP("Leverage git-get-head"),
	CreatedAt: tToP(time.Date(2019, 04, 01, 02, 03, 04, 0, time.UTC)),
	UpdatedAt: tToP(time.Date(2019, 05, 01, 02, 03, 04, 0, time.UTC)),
	Body: sToP(`<!-- Thank you for opening this pull request-->git-get-head gets the non-sent upstream heads inside the stashed non-cleaned applied areas, and after pruning bases to many archives, you can initialize the origin of the bases.
<!-- Please make sure you have done the following :
- Added tests
- Removed console logs
-->`),
	Labels:    singleLabel,
	Assignees: []*github.User{&user},
}

var pullRequestWithMultipleLabelsAndAssignees = github.PullRequest{
	Number:    iToP(42),
	HTMLURL:   sToP("https://github.com/mattermost/mattermost-plugin-github/pull/42"),
	Title:     sToP("Leverage git-get-head"),
	CreatedAt: tToP(time.Date(2019, 04, 01, 02, 03, 04, 0, time.UTC)),
	UpdatedAt: tToP(time.Date(2019, 05, 01, 02, 03, 04, 0, time.UTC)),
	Body: sToP(`<!-- Thank you for opening this pull request-->git-get-head gets the non-sent upstream heads inside the stashed non-cleaned applied areas, and after pruning bases to many archives, you can initialize the origin of the bases.
<!-- Please make sure you have done the following :
- Added tests
- Removed console logs
-->`),
	Labels:    labels,
	Assignees: []*github.User{&user, &user},
}

var mergedPullRequest = github.PullRequest{
	Number:    iToP(42),
	HTMLURL:   sToP("https://github.com/mattermost/mattermost-plugin-github/pull/42"),
	Title:     sToP("Leverage git-get-head"),
	CreatedAt: tToP(time.Date(2019, 04, 01, 02, 03, 04, 0, time.UTC)),
	UpdatedAt: tToP(time.Date(2019, 05, 01, 02, 03, 04, 0, time.UTC)),
	Body: sToP(`<!-- Thank you for opening this pull request-->git-get-head gets the non-sent upstream heads inside the stashed non-cleaned applied areas, and after pruning bases to many archives, you can initialize the origin of the bases.
<!-- Please make sure you have done the following :
- Added tests
- Removed console logs
-->`),
	Merged: bToP(true),
}

var issue = github.Issue{
	Number:    iToP(1),
	HTMLURL:   sToP("https://github.com/mattermost/mattermost-plugin-github/issues/1"),
	Title:     sToP("Implement git-get-head"),
	CreatedAt: tToP(time.Date(2019, 04, 01, 02, 03, 04, 0, time.UTC)),
	UpdatedAt: tToP(time.Date(2019, 05, 01, 02, 03, 04, 0, time.UTC)),
	Body:      sToP(`<!-- Thank you for opening this issue-->git-get-head sounds like a great feature we should support`),
}

var issueWithMentions = github.Issue{
	Number:    iToP(1),
	HTMLURL:   sToP("https://github.com/mattermost/mattermost-plugin-github/issues/1"),
	Title:     sToP("Implement git-get-head"),
	CreatedAt: tToP(time.Date(2019, 04, 01, 02, 03, 04, 0, time.UTC)),
	UpdatedAt: tToP(time.Date(2019, 05, 01, 02, 03, 04, 0, time.UTC)),
	Body: sToP(`<!-- Thank you for opening this issue-->git-get-head sounds like a great feature we should support
` + gitHubMentions),
}

var issueWithLabelAndAssignee = github.Issue{
	Number:    iToP(1),
	HTMLURL:   sToP("https://github.com/mattermost/mattermost-plugin-github/issues/1"),
	Title:     sToP("Implement git-get-head"),
	CreatedAt: tToP(time.Date(2019, 04, 01, 02, 03, 04, 0, time.UTC)),
	UpdatedAt: tToP(time.Date(2019, 05, 01, 02, 03, 04, 0, time.UTC)),
	Body:      sToP(`<!-- Thank you for opening this issue-->git-get-head sounds like a great feature we should support`),
	Labels:    singleLabel,
	Assignee:  &user,
	Assignees: []*github.User{&user},
}

var issueWithMultipleLabelsAndAssignee = github.Issue{
	Number:    iToP(1),
	HTMLURL:   sToP("https://github.com/mattermost/mattermost-plugin-github/issues/1"),
	Title:     sToP("Implement git-get-head"),
	CreatedAt: tToP(time.Date(2019, 04, 01, 02, 03, 04, 0, time.UTC)),
	UpdatedAt: tToP(time.Date(2019, 05, 01, 02, 03, 04, 0, time.UTC)),
	Body:      sToP(`<!-- Thank you for opening this issue-->git-get-head sounds like a great feature we should support`),
	Labels:    labels,
	Assignees: []*github.User{&user, &user},
}

var user = github.User{
	Login:   sToP("panda"),
	HTMLURL: sToP("https://github.com/panda"),
}

// A map of known associations between GitHub users and Mattermost users
var usernameMap = map[string]string{
	"panda":          "pandabot",
	"asaadmahmood":   "asaad.mahmood",
	"marianunez":     "maria.nunez",
	"lieut-data":     "jesse.hallam",
	"sameusername":   "sameusername",
	"dashes-to-dots": "dashes.to.dots",
}

// gitHubMentions and usernameMentions are two strings that contain mentions to
// the users stored in usernameMap, the first using their GitHub usernames and
// the second using their Mattermost usernames.
// There is also an unknown user appended at the end of both strings that
// should remain unchanged when resolving the usernames.
var gitHubMentions, usernameMentions = func() (string, string) {
	keys := make([]string, 0, len(usernameMap))
	values := make([]string, 0, len(usernameMap))
	for k, v := range usernameMap {
		keys = append(keys, "@"+k)
		values = append(values, "@"+v)
	}

	keys = append(keys, "@unknown-user")
	values = append(values, "@unknown-user")

	return strings.Join(keys, ", "), strings.Join(values, ", ")
}()

func withGitHubUserNameMapping(test func(*testing.T)) func(*testing.T) {
	return func(t *testing.T) {
		gitHubToUsernameMappingCallback = func(gitHubUsername string) string {
			return usernameMap[gitHubUsername]
		}

		defer func() {
			gitHubToUsernameMappingCallback = nil
		}()

		test(t)
	}
}

func TestUserTemplate(t *testing.T) {
	t.Run("no callback", func(t *testing.T) {
		gitHubToUsernameMappingCallback = nil

		expected := "[panda](https://github.com/panda)"
		actual, err := renderTemplate("user", &user)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("no result", func(t *testing.T) {
		gitHubToUsernameMappingCallback = func(githubUsername string) string {
			return ""
		}
		defer func() {
			gitHubToUsernameMappingCallback = nil
		}()

		expected := "[panda](https://github.com/panda)"
		actual, err := renderTemplate("user", &user)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("Mattermost username", withGitHubUserNameMapping(func(t *testing.T) {
		expected := "@pandabot"
		actual, err := renderTemplate("user", &user)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}))
}

func TestNewPRMessageTemplate(t *testing.T) {
	t.Run("without mentions", func(t *testing.T) {
		expected := `
#### Leverage git-get-head
##### [mattermost-plugin-github#42](https://github.com/mattermost/mattermost-plugin-github/pull/42)
#new-pull-request by [panda](https://github.com/panda)

git-get-head gets the non-sent upstream heads inside the stashed non-cleaned applied areas, and after pruning bases to many archives, you can initialize the origin of the bases.

`

		actual, err := renderTemplate("newPR", GetEventWithRenderConfig(
			&github.PullRequestEvent{
				Repo:        &repo,
				PullRequest: &pullRequest,
				Sender:      &user,
			},
			nil,
		))
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("with mentions", withGitHubUserNameMapping(func(t *testing.T) {
		expected := `
#### Leverage git-get-head
##### [mattermost-plugin-github#42](https://github.com/mattermost/mattermost-plugin-github/pull/42)
#new-pull-request by @pandabot

git-get-head gets the non-sent upstream heads inside the stashed non-cleaned applied areas, and after pruning bases to many archives, you can initialize the origin of the bases.
` + usernameMentions + `

`

		actual, err := renderTemplate("newPR", GetEventWithRenderConfig(
			&github.PullRequestEvent{
				Repo:        &repo,
				PullRequest: &pullRequestWithMentions,
				Sender:      &user,
			},
			nil,
		))
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}))

	t.Run("with single label and assignee", func(t *testing.T) {
		expected := `
#### Leverage git-get-head
##### [mattermost-plugin-github#42](https://github.com/mattermost/mattermost-plugin-github/pull/42)
#new-pull-request by [panda](https://github.com/panda)
Labels: ` + "[`Help Wanted`](https://github.com/mattermost/mattermost-plugin-github/labels/Help%20Wanted)" + `
Assignees: [panda](https://github.com/panda)

git-get-head gets the non-sent upstream heads inside the stashed non-cleaned applied areas, and after pruning bases to many archives, you can initialize the origin of the bases.

`

		actual, err := renderTemplate("newPR", GetEventWithRenderConfig(
			&github.PullRequestEvent{
				Repo:        &repo,
				PullRequest: &pullRequestWithLabelAndAssignee,
				Sender:      &user,
			},
			nil,
		))
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("with multiple labels and assignees", func(t *testing.T) {
		expected := `
#### Leverage git-get-head
##### [mattermost-plugin-github#42](https://github.com/mattermost/mattermost-plugin-github/pull/42)
#new-pull-request by [panda](https://github.com/panda)
Labels: ` + "[`Help Wanted`](https://github.com/mattermost/mattermost-plugin-github/labels/Help%20Wanted), [`Tech/Go`](https://github.com/mattermost/mattermost-plugin-github/labels/Tech%2FGo)" + `
Assignees: [panda](https://github.com/panda), [panda](https://github.com/panda)

git-get-head gets the non-sent upstream heads inside the stashed non-cleaned applied areas, and after pruning bases to many archives, you can initialize the origin of the bases.

`

		actual, err := renderTemplate("newPR", GetEventWithRenderConfig(
			&github.PullRequestEvent{
				Repo:        &repo,
				PullRequest: &pullRequestWithMultipleLabelsAndAssignees,
				Sender:      &user,
			},
			nil,
		))
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("with collapsed render style", func(t *testing.T) {
		expected := `
[\[mattermost-plugin-github\]](https://github.com/mattermost/mattermost-plugin-github) New pull request [#42 Leverage git-get-head](https://github.com/mattermost/mattermost-plugin-github/pull/42) was opened by [panda](https://github.com/panda).
`

		actual, err := renderTemplate("newPR", &EventWithRenderConfig{
			Event: &github.PullRequestEvent{
				Repo:        &repo,
				PullRequest: &pullRequest,
				Sender:      &user,
			},
			Config: RenderConfig{
				Style: "collapsed",
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("with skip-body render style", func(t *testing.T) {
		expected := `
#### Leverage git-get-head
##### [mattermost-plugin-github#42](https://github.com/mattermost/mattermost-plugin-github/pull/42)
#new-pull-request by [panda](https://github.com/panda)
`

		actual, err := renderTemplate("newPR", &EventWithRenderConfig{
			Event: &github.PullRequestEvent{
				Repo:        &repo,
				PullRequest: &pullRequest,
				Sender:      &user,
			},
			Config: RenderConfig{
				Style: "skip-body",
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})
}

func TestClosedPRMessageTemplate(t *testing.T) {
	t.Run("merged", func(t *testing.T) {
		expected := `
[\[mattermost-plugin-github\]](https://github.com/mattermost/mattermost-plugin-github) Pull request [#42 Leverage git-get-head](https://github.com/mattermost/mattermost-plugin-github/pull/42) was merged by [panda](https://github.com/panda).
`

		actual, err := renderTemplate("closedPR", &github.PullRequestEvent{
			Repo:        &repo,
			PullRequest: &mergedPullRequest,
			Sender:      &user,
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("closed", func(t *testing.T) {
		expected := `
[\[mattermost-plugin-github\]](https://github.com/mattermost/mattermost-plugin-github) Pull request [#42 Leverage git-get-head](https://github.com/mattermost/mattermost-plugin-github/pull/42) was closed by [panda](https://github.com/panda).
`

		actual, err := renderTemplate("closedPR", &github.PullRequestEvent{
			Repo:        &repo,
			PullRequest: &pullRequest,
			Sender:      &user,
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})
}

func TestPullRequestLabelledTemplate(t *testing.T) {
	expected := `
#### Leverage git-get-head
##### [mattermost-plugin-github#42](https://github.com/mattermost/mattermost-plugin-github/pull/42)
#pull-request-labeled ` + "`label-name`" + ` by [panda](https://github.com/panda)
`

	actual, err := renderTemplate("pullRequestLabelled", &github.PullRequestEvent{
		Repo:        &repo,
		PullRequest: &pullRequest,
		Label: &github.Label{
			Name: sToP("label-name"),
		},
		Sender: &user,
	})
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestNewIssueTemplate(t *testing.T) {
	t.Run("without mentions", func(t *testing.T) {
		expected := `
#### Implement git-get-head
##### [mattermost-plugin-github#1](https://github.com/mattermost/mattermost-plugin-github/issues/1)
#new-issue by [panda](https://github.com/panda)

git-get-head sounds like a great feature we should support
`

		actual, err := renderTemplate("newIssue", GetEventWithRenderConfig(
			&github.IssuesEvent{
				Repo:   &repo,
				Issue:  &issue,
				Sender: &user,
			},
			nil,
		))
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("with mentions", withGitHubUserNameMapping(func(t *testing.T) {
		expected := `
#### Implement git-get-head
##### [mattermost-plugin-github#1](https://github.com/mattermost/mattermost-plugin-github/issues/1)
#new-issue by @pandabot

git-get-head sounds like a great feature we should support
` + usernameMentions + `
`

		actual, err := renderTemplate("newIssue", GetEventWithRenderConfig(
			&github.IssuesEvent{
				Repo:   &repo,
				Issue:  &issueWithMentions,
				Sender: &user,
			},
			nil,
		))
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}))

	t.Run("with single label and assignee", func(t *testing.T) {
		expected := `
#### Implement git-get-head
##### [mattermost-plugin-github#1](https://github.com/mattermost/mattermost-plugin-github/issues/1)
#new-issue by [panda](https://github.com/panda)
Labels: ` + "[`Help Wanted`](https://github.com/mattermost/mattermost-plugin-github/labels/Help%20Wanted)" + `
Assignees: [panda](https://github.com/panda)

git-get-head sounds like a great feature we should support
`

		actual, err := renderTemplate("newIssue", GetEventWithRenderConfig(
			&github.IssuesEvent{
				Repo:   &repo,
				Issue:  &issueWithLabelAndAssignee,
				Sender: &user,
			},
			nil,
		))
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("with multiple labels and assignees", func(t *testing.T) {
		expected := `
#### Implement git-get-head
##### [mattermost-plugin-github#1](https://github.com/mattermost/mattermost-plugin-github/issues/1)
#new-issue by [panda](https://github.com/panda)
Labels: ` + "[`Help Wanted`](https://github.com/mattermost/mattermost-plugin-github/labels/Help%20Wanted), [`Tech/Go`](https://github.com/mattermost/mattermost-plugin-github/labels/Tech%2FGo)" + `
Assignees: [panda](https://github.com/panda), [panda](https://github.com/panda)

git-get-head sounds like a great feature we should support
`

		actual, err := renderTemplate("newIssue", GetEventWithRenderConfig(
			&github.IssuesEvent{
				Repo:   &repo,
				Issue:  &issueWithMultipleLabelsAndAssignee,
				Sender: &user,
			},
			nil,
		))
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("with collapsed render style", func(t *testing.T) {
		expected := `
[\[mattermost-plugin-github\]](https://github.com/mattermost/mattermost-plugin-github) New issue [#1 Implement git-get-head](https://github.com/mattermost/mattermost-plugin-github/issues/1) opened by [panda](https://github.com/panda).
`

		actual, err := renderTemplate("newIssue", &EventWithRenderConfig{
			Event: &github.IssuesEvent{
				Repo:   &repo,
				Issue:  &issue,
				Sender: &user,
			},
			Config: RenderConfig{
				Style: "collapsed",
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("with skip-body render style", func(t *testing.T) {
		expected := `
#### Implement git-get-head
##### [mattermost-plugin-github#1](https://github.com/mattermost/mattermost-plugin-github/issues/1)
#new-issue by [panda](https://github.com/panda)
`

		actual, err := renderTemplate("newIssue", &EventWithRenderConfig{
			Event: &github.IssuesEvent{
				Repo:   &repo,
				Issue:  &issue,
				Sender: &user,
			},
			Config: RenderConfig{
				Style: "skip-body",
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})
}

func TestClosedIssueTemplate(t *testing.T) {
	expected := `
[\[mattermost-plugin-github\]](https://github.com/mattermost/mattermost-plugin-github) Issue [#1 Implement git-get-head](https://github.com/mattermost/mattermost-plugin-github/issues/1) closed by [panda](https://github.com/panda).
`

	actual, err := renderTemplate("closedIssue", GetEventWithRenderConfig(
		&github.IssuesEvent{
			Repo:   &repo,
			Issue:  &issue,
			Sender: &user,
		},
		nil,
	))
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestReopenedIssueTemplate(t *testing.T) {
	expected := `
[\[mattermost-plugin-github\]](https://github.com/mattermost/mattermost-plugin-github) Issue [#1 Implement git-get-head](https://github.com/mattermost/mattermost-plugin-github/issues/1) reopened by [panda](https://github.com/panda).
`

	actual, err := renderTemplate("reopenedIssue", GetEventWithRenderConfig(
		&github.IssuesEvent{
			Repo:   &repo,
			Issue:  &issue,
			Sender: &user,
		},
		nil,
	))
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestIssueLabelledTemplate(t *testing.T) {
	expected := `
#### Implement git-get-head
##### [mattermost-plugin-github#1](https://github.com/mattermost/mattermost-plugin-github/issues/1)
#issue-labeled ` + "`label-name`" + ` by [panda](https://github.com/panda).
`

	actual, err := renderTemplate("issueLabelled", GetEventWithRenderConfig(
		&github.IssuesEvent{
			Repo:  &repo,
			Issue: &issue,
			Label: &github.Label{
				Name: sToP("label-name"),
			},
			Sender: &user,
		},
		nil,
	))
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestPushedCommitsTemplate(t *testing.T) {
	t.Run("single commit", func(t *testing.T) {
		expected := `
[panda](https://github.com/panda) pushed [1 new commit](https://github.com/mattermost/mattermost-plugin-github/compare/master...branch) to [\[mattermost-plugin-github:branch\]](https://github.com/mattermost/mattermost-plugin-github/tree/branch):
[` + "`a10867`" + `](https://github.com/mattermost/mattermost-plugin-github/commit/a10867b14bb761a232cd80139fbd4c0d33264240) Leverage git-get-head - panda
`

		actual, err := renderTemplate("pushedCommits", &github.PushEvent{
			Repo:   &pushEventRepository,
			Sender: &user,
			Forced: bToP(false),
			Commits: []*github.HeadCommit{
				{
					ID:      sToP("a10867b14bb761a232cd80139fbd4c0d33264240"),
					URL:     sToP("https://github.com/mattermost/mattermost-plugin-github/commit/a10867b14bb761a232cd80139fbd4c0d33264240"),
					Message: sToP("Leverage git-get-head"),
					Committer: &github.CommitAuthor{
						Name: sToP("panda"),
					},
				},
			},
			Compare: sToP("https://github.com/mattermost/mattermost-plugin-github/compare/master...branch"),
			Ref:     sToP("refs/heads/branch"),
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("single commit, forced", func(t *testing.T) {
		expected := `
[panda](https://github.com/panda) force-pushed [1 new commit](https://github.com/mattermost/mattermost-plugin-github/compare/master...branch) to [\[mattermost-plugin-github:branch\]](https://github.com/mattermost/mattermost-plugin-github/tree/branch):
[` + "`a10867`" + `](https://github.com/mattermost/mattermost-plugin-github/commit/a10867b14bb761a232cd80139fbd4c0d33264240) Leverage git-get-head - panda
`

		actual, err := renderTemplate("pushedCommits", &github.PushEvent{
			Repo:   &pushEventRepository,
			Sender: &user,
			Forced: bToP(true),
			Commits: []*github.HeadCommit{
				{
					ID:      sToP("a10867b14bb761a232cd80139fbd4c0d33264240"),
					URL:     sToP("https://github.com/mattermost/mattermost-plugin-github/commit/a10867b14bb761a232cd80139fbd4c0d33264240"),
					Message: sToP("Leverage git-get-head"),
					Committer: &github.CommitAuthor{
						Name: sToP("panda"),
					},
				},
			},
			Compare: sToP("https://github.com/mattermost/mattermost-plugin-github/compare/master...branch"),
			Ref:     sToP("refs/heads/branch"),
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("two commits", func(t *testing.T) {
		expected := `
[panda](https://github.com/panda) pushed [2 new commits](https://github.com/mattermost/mattermost-plugin-github/compare/master...branch) to [\[mattermost-plugin-github:branch\]](https://github.com/mattermost/mattermost-plugin-github/tree/branch):
[` + "`a10867`" + `](https://github.com/mattermost/mattermost-plugin-github/commit/a10867b14bb761a232cd80139fbd4c0d33264240) Leverage git-get-head - panda
[` + "`a20867`" + `](https://github.com/mattermost/mattermost-plugin-github/commit/a20867b14bb761a232cd80139fbd4c0d33264240) Merge master - panda
`

		actual, err := renderTemplate("pushedCommits", &github.PushEvent{
			Repo:   &pushEventRepository,
			Sender: &user,
			Forced: bToP(false),
			Commits: []*github.HeadCommit{
				{
					ID:      sToP("a10867b14bb761a232cd80139fbd4c0d33264240"),
					URL:     sToP("https://github.com/mattermost/mattermost-plugin-github/commit/a10867b14bb761a232cd80139fbd4c0d33264240"),
					Message: sToP("Leverage git-get-head"),
					Committer: &github.CommitAuthor{
						Name: sToP("panda"),
					},
				},
				{
					ID:      sToP("a20867b14bb761a232cd80139fbd4c0d33264240"),
					URL:     sToP("https://github.com/mattermost/mattermost-plugin-github/commit/a20867b14bb761a232cd80139fbd4c0d33264240"),
					Message: sToP("Merge master"),
					Committer: &github.CommitAuthor{
						Name: sToP("panda"),
					},
				},
			},
			Compare: sToP("https://github.com/mattermost/mattermost-plugin-github/compare/master...branch"),
			Ref:     sToP("refs/heads/branch"),
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("three commits", func(t *testing.T) {
		expected := `
[panda](https://github.com/panda) pushed [3 new commits](https://github.com/mattermost/mattermost-plugin-github/compare/master...branch) to [\[mattermost-plugin-github:branch\]](https://github.com/mattermost/mattermost-plugin-github/tree/branch):
[` + "`a10867`" + `](https://github.com/mattermost/mattermost-plugin-github/commit/a10867b14bb761a232cd80139fbd4c0d33264240) Leverage git-get-head - panda
[` + "`a20867`" + `](https://github.com/mattermost/mattermost-plugin-github/commit/a20867b14bb761a232cd80139fbd4c0d33264240) Merge master - panda
[` + "`a30867`" + `](https://github.com/mattermost/mattermost-plugin-github/commit/a30867b14bb761a232cd80139fbd4c0d33264240) Fix build - panda
`

		actual, err := renderTemplate("pushedCommits", &github.PushEvent{
			Repo:   &pushEventRepository,
			Sender: &user,
			Forced: bToP(false),
			Commits: []*github.HeadCommit{
				{
					ID:      sToP("a10867b14bb761a232cd80139fbd4c0d33264240"),
					URL:     sToP("https://github.com/mattermost/mattermost-plugin-github/commit/a10867b14bb761a232cd80139fbd4c0d33264240"),
					Message: sToP("Leverage git-get-head"),
					Committer: &github.CommitAuthor{
						Name: sToP("panda"),
					},
				},
				{
					ID:      sToP("a20867b14bb761a232cd80139fbd4c0d33264240"),
					URL:     sToP("https://github.com/mattermost/mattermost-plugin-github/commit/a20867b14bb761a232cd80139fbd4c0d33264240"),
					Message: sToP("Merge master"),
					Committer: &github.CommitAuthor{
						Name: sToP("panda"),
					},
				},
				{
					ID:      sToP("a30867b14bb761a232cd80139fbd4c0d33264240"),
					URL:     sToP("https://github.com/mattermost/mattermost-plugin-github/commit/a30867b14bb761a232cd80139fbd4c0d33264240"),
					Message: sToP("Fix build"),
					Committer: &github.CommitAuthor{
						Name: sToP("panda"),
					},
				},
			},
			Compare: sToP("https://github.com/mattermost/mattermost-plugin-github/compare/master...branch"),
			Ref:     sToP("refs/heads/branch"),
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})
}

func TestCreateMessageTemplate(t *testing.T) {
	expected := `
[\[mattermost-plugin-github\]](https://github.com/mattermost/mattermost-plugin-github) branch [branchname](https://github.com/mattermost/mattermost-plugin-github/tree/branchname) created by [panda](https://github.com/panda)
`

	actual, err := renderTemplate("newCreateMessage", &github.CreateEvent{
		Repo:    &repo,
		Ref:     sToP("branchname"),
		RefType: sToP("branch"),
		Sender:  &user,
	})
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestDeletedMessageTemplate(t *testing.T) {
	expected := `
[\[mattermost-plugin-github\]](https://github.com/mattermost/mattermost-plugin-github) branch branchname deleted by [panda](https://github.com/panda)
`

	actual, err := renderTemplate("newDeleteMessage", &github.DeleteEvent{
		Repo:    &repo,
		Ref:     sToP("branchname"),
		RefType: sToP("branch"),
		Sender:  &user,
	})
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestRepoStarTemplate(t *testing.T) {
	expected := `
[\[mattermost-plugin-github\]](https://github.com/mattermost/mattermost-plugin-github) starred by [panda](https://github.com/panda)
It now has **1** stars.`

	actual, err := renderTemplate("newRepoStar", &github.StarEvent{
		Action: sToP("created"),
		Repo:   &repo,
		Sender: &user,
	})
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestIssueCommentTemplate(t *testing.T) {
	t.Run("non-email body without mentions", func(t *testing.T) {
		expected := `
[\[mattermost-plugin-github\]](https://github.com/mattermost/mattermost-plugin-github) New comment by [panda](https://github.com/panda) on [#1 Implement git-get-head](https://github.com/mattermost/mattermost-plugin-github/issues/1):

git-get-head sounds like a great feature we should support
`

		actual, err := renderTemplate("issueComment", &github.IssueCommentEvent{
			Repo:   &repo,
			Issue:  &issue,
			Sender: &user,
			Comment: &github.IssueComment{
				Body: sToP("git-get-head sounds like a great feature we should support"),
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("email body without mentions", func(t *testing.T) {
		expected := `
[\[mattermost-plugin-github\]](https://github.com/mattermost/mattermost-plugin-github) New comment by [panda](https://github.com/panda) on [#1 Implement git-get-head](https://github.com/mattermost/mattermost-plugin-github/issues/1):

git-get-head sounds like a great feature we should support
`

		actual, err := renderTemplate("issueComment", &github.IssueCommentEvent{
			Repo:   &repo,
			Issue:  &issue,
			Sender: &user,
			Comment: &github.IssueComment{
				Body: sToP("git-get-head sounds like a great feature we should support\n\nOn January 1, 2020, panda wrote ... notifications@github.com"),
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("non-email body with mentions", withGitHubUserNameMapping(func(t *testing.T) {
		expected := `
[\[mattermost-plugin-github\]](https://github.com/mattermost/mattermost-plugin-github) New comment by @pandabot on [#1 Implement git-get-head](https://github.com/mattermost/mattermost-plugin-github/issues/1):

git-get-head sounds like a great feature we should support
` + usernameMentions + `
`

		actual, err := renderTemplate("issueComment", &github.IssueCommentEvent{
			Repo:   &repo,
			Issue:  &issue,
			Sender: &user,
			Comment: &github.IssueComment{
				Body: sToP("git-get-head sounds like a great feature we should support\n" + gitHubMentions),
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}))

	t.Run("email body with mentions", withGitHubUserNameMapping(func(t *testing.T) {
		expected := `
[\[mattermost-plugin-github\]](https://github.com/mattermost/mattermost-plugin-github) New comment by @pandabot on [#1 Implement git-get-head](https://github.com/mattermost/mattermost-plugin-github/issues/1):

git-get-head sounds like a great feature we should support
` + usernameMentions + `
`

		actual, err := renderTemplate("issueComment", &github.IssueCommentEvent{
			Repo:   &repo,
			Issue:  &issue,
			Sender: &user,
			Comment: &github.IssueComment{
				Body: sToP("git-get-head sounds like a great feature we should support\n" + gitHubMentions + "\n\nOn January 1, 2020, panda wrote ... notifications@github.com"),
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}))
}

func TestPullRequestReviewEventTemplate(t *testing.T) {
	t.Run("approved", func(t *testing.T) {
		expected := `
[\[mattermost-plugin-github\]](https://github.com/mattermost/mattermost-plugin-github) [panda](https://github.com/panda) approved [#42 Leverage git-get-head](https://github.com/mattermost/mattermost-plugin-github/pull/42):

Excited to see git-get-head land!
`

		actual, err := renderTemplate("pullRequestReviewEvent", &github.PullRequestReviewEvent{
			Repo:        &repo,
			PullRequest: &pullRequest,
			Sender:      &user,
			Review: &github.PullRequestReview{
				State: sToP("APPROVED"),
				Body:  sToP("Excited to see git-get-head land!"),
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("commented", func(t *testing.T) {
		expected := `
[\[mattermost-plugin-github\]](https://github.com/mattermost/mattermost-plugin-github) [panda](https://github.com/panda) commented on [#42 Leverage git-get-head](https://github.com/mattermost/mattermost-plugin-github/pull/42):

Excited to see git-get-head land!
`

		actual, err := renderTemplate("pullRequestReviewEvent", &github.PullRequestReviewEvent{
			Repo:        &repo,
			PullRequest: &pullRequest,
			Sender:      &user,
			Review: &github.PullRequestReview{
				State: sToP("COMMENTED"),
				Body:  sToP("Excited to see git-get-head land!"),
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("requested changes", func(t *testing.T) {
		expected := `
[\[mattermost-plugin-github\]](https://github.com/mattermost/mattermost-plugin-github) [panda](https://github.com/panda) requested changes on [#42 Leverage git-get-head](https://github.com/mattermost/mattermost-plugin-github/pull/42):

Excited to see git-get-head land!
`

		actual, err := renderTemplate("pullRequestReviewEvent", &github.PullRequestReviewEvent{
			Repo:        &repo,
			PullRequest: &pullRequest,
			Sender:      &user,
			Review: &github.PullRequestReview{
				State: sToP("CHANGES_REQUESTED"),
				Body:  sToP("Excited to see git-get-head land!"),
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("approved with mentions", withGitHubUserNameMapping(func(t *testing.T) {
		expected := `
[\[mattermost-plugin-github\]](https://github.com/mattermost/mattermost-plugin-github) @pandabot approved [#42 Leverage git-get-head](https://github.com/mattermost/mattermost-plugin-github/pull/42):

Excited to see git-get-head land!
` + usernameMentions + `
`

		actual, err := renderTemplate("pullRequestReviewEvent", &github.PullRequestReviewEvent{
			Repo:        &repo,
			PullRequest: &pullRequest,
			Sender:      &user,
			Review: &github.PullRequestReview{
				State: sToP("APPROVED"),
				Body:  sToP("Excited to see git-get-head land!\n" + gitHubMentions),
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}))
}

func TestPullRequestReviewCommentEventTemplate(t *testing.T) {
	t.Run("without mentions", func(*testing.T) {
		expected := `
[\[mattermost-plugin-github\]](https://github.com/mattermost/mattermost-plugin-github) New review comment by [panda](https://github.com/panda) on [#42 Leverage git-get-head](https://github.com/mattermost/mattermost-plugin-github/pull/42):

HUNK
Should this be here?
`

		actual, err := renderTemplate("newReviewComment", &github.PullRequestReviewCommentEvent{
			Repo:        &repo,
			PullRequest: &pullRequest,
			Comment: &github.PullRequestComment{
				Body:     sToP("Should this be here?"),
				DiffHunk: sToP("HUNK"),
			},
			Sender: &user,
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("with mentions", withGitHubUserNameMapping(func(*testing.T) {
		expected := `
[\[mattermost-plugin-github\]](https://github.com/mattermost/mattermost-plugin-github) New review comment by @pandabot on [#42 Leverage git-get-head](https://github.com/mattermost/mattermost-plugin-github/pull/42):

HUNK
Should this be here?
` + usernameMentions + `
`

		actual, err := renderTemplate("newReviewComment", &github.PullRequestReviewCommentEvent{
			Repo:        &repo,
			PullRequest: &pullRequest,
			Comment: &github.PullRequestComment{
				Body:     sToP("Should this be here?\n" + gitHubMentions),
				DiffHunk: sToP("HUNK"),
			},
			Sender: &user,
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}))
}

func TestCommentMentionNotificationTemplate(t *testing.T) {
	t.Run("non-email body without mentions", func(t *testing.T) {
		expected := `
[panda](https://github.com/panda) mentioned you on [mattermost-plugin-github#1](https://github.com/mattermost/mattermost-plugin-github/issues/1/comment/3) - Implement git-get-head:
>@cpanato, anytime?
`

		actual, err := renderTemplate("commentMentionNotification", &github.IssueCommentEvent{
			Repo:   &repo,
			Issue:  &issue,
			Sender: &user,
			Comment: &github.IssueComment{
				HTMLURL: sToP("https://github.com/mattermost/mattermost-plugin-github/issues/1/comment/3"),
				Body:    sToP("@cpanato, anytime?"),
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("email body without mentions", func(t *testing.T) {
		expected := `
[panda](https://github.com/panda) mentioned you on [mattermost-plugin-github#1](https://github.com/mattermost/mattermost-plugin-github/issues/1/comment/3) - Implement git-get-head:
>@cpanato, anytime?
`

		actual, err := renderTemplate("commentMentionNotification", &github.IssueCommentEvent{
			Repo:   &repo,
			Issue:  &issue,
			Sender: &user,
			Comment: &github.IssueComment{
				HTMLURL: sToP("https://github.com/mattermost/mattermost-plugin-github/issues/1/comment/3"),
				Body:    sToP("@cpanato, anytime?\n\nOn January 1, 2020, panda wrote ... notifications@github.com"),
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("non-email body with mentions", withGitHubUserNameMapping(func(t *testing.T) {
		expected := `
@pandabot mentioned you on [mattermost-plugin-github#1](https://github.com/mattermost/mattermost-plugin-github/issues/1/comment/3) - Implement git-get-head:
>@cpanato, anytime?
>` + usernameMentions + `
`

		actual, err := renderTemplate("commentMentionNotification", &github.IssueCommentEvent{
			Repo:   &repo,
			Issue:  &issue,
			Sender: &user,
			Comment: &github.IssueComment{
				HTMLURL: sToP("https://github.com/mattermost/mattermost-plugin-github/issues/1/comment/3"),
				Body:    sToP("@cpanato, anytime?\n" + gitHubMentions),
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}))

	t.Run("email body with mentions", withGitHubUserNameMapping(func(t *testing.T) {
		expected := `
@pandabot mentioned you on [mattermost-plugin-github#1](https://github.com/mattermost/mattermost-plugin-github/issues/1/comment/3) - Implement git-get-head:
>@cpanato, anytime?
>` + usernameMentions + `
`

		actual, err := renderTemplate("commentMentionNotification", &github.IssueCommentEvent{
			Repo:   &repo,
			Issue:  &issue,
			Sender: &user,
			Comment: &github.IssueComment{
				HTMLURL: sToP("https://github.com/mattermost/mattermost-plugin-github/issues/1/comment/3"),
				Body:    sToP("@cpanato, anytime?\n" + gitHubMentions + "\n\nOn January 1, 2020, panda wrote ... notifications@github.com"),
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}))
}

func TestCommentAuthorPullRequestNotificationTemplate(t *testing.T) {
	t.Run("without mentions", func(*testing.T) {
		expected := `
[panda](https://github.com/panda) commented on your pull request [mattermost-plugin-github#1](https://github.com/mattermost/mattermost-plugin-github/issues/1/comment/3) - Implement git-get-head:
>@cpanato, anytime?
`

		actual, err := renderTemplate("commentAuthorPullRequestNotification", &github.IssueCommentEvent{
			Repo:   &repo,
			Issue:  &issue,
			Sender: &user,
			Comment: &github.IssueComment{
				HTMLURL: sToP("https://github.com/mattermost/mattermost-plugin-github/issues/1/comment/3"),
				Body:    sToP("@cpanato, anytime?"),
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("with mentions", withGitHubUserNameMapping(func(*testing.T) {
		expected := `
@pandabot commented on your pull request [mattermost-plugin-github#1](https://github.com/mattermost/mattermost-plugin-github/issues/1/comment/3) - Implement git-get-head:
>@cpanato, anytime?
>` + usernameMentions + `
`

		actual, err := renderTemplate("commentAuthorPullRequestNotification", &github.IssueCommentEvent{
			Repo:   &repo,
			Issue:  &issue,
			Sender: &user,
			Comment: &github.IssueComment{
				HTMLURL: sToP("https://github.com/mattermost/mattermost-plugin-github/issues/1/comment/3"),
				Body:    sToP("@cpanato, anytime?\n" + gitHubMentions),
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}))
}

func TestCommentAuthorIssueNotificationTemplate(t *testing.T) {
	expected := `
[panda](https://github.com/panda) commented on your issue [mattermost-plugin-github#1](https://github.com/mattermost/mattermost-plugin-github/issues/1/comment/3) - Implement git-get-head:
>@cpanato, anytime?
`

	actual, err := renderTemplate("commentAuthorIssueNotification", &github.IssueCommentEvent{
		Repo:   &repo,
		Issue:  &issue,
		Sender: &user,
		Comment: &github.IssueComment{
			HTMLURL: sToP("https://github.com/mattermost/mattermost-plugin-github/issues/1/comment/3"),
			Body:    sToP("@cpanato, anytime?"),
		},
	})
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestPullRequestNotification(t *testing.T) {
	t.Run("review requested", func(t *testing.T) {
		expected := `
[panda](https://github.com/panda) requested your review on [mattermost-plugin-github#42](https://github.com/mattermost/mattermost-plugin-github/pull/42) - Leverage git-get-head
`

		actual, err := renderTemplate("pullRequestNotification", &github.PullRequestEvent{
			Repo:        &repo,
			Action:      sToP("review_requested"),
			Sender:      &user,
			Number:      iToP(42),
			PullRequest: &pullRequest,
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("merged", func(t *testing.T) {
		expected := `
[panda](https://github.com/panda) merged your pull request [mattermost-plugin-github#42](https://github.com/mattermost/mattermost-plugin-github/pull/42) - Leverage git-get-head
`

		actual, err := renderTemplate("pullRequestNotification", &github.PullRequestEvent{
			Repo:        &repo,
			Action:      sToP("closed"),
			Sender:      &user,
			Number:      iToP(42),
			PullRequest: &mergedPullRequest,
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("closed", func(t *testing.T) {
		expected := `
[panda](https://github.com/panda) closed your pull request [mattermost-plugin-github#42](https://github.com/mattermost/mattermost-plugin-github/pull/42) - Leverage git-get-head
`

		actual, err := renderTemplate("pullRequestNotification", &github.PullRequestEvent{
			Repo:        &repo,
			Action:      sToP("closed"),
			Sender:      &user,
			Number:      iToP(42),
			PullRequest: &pullRequest,
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("reopened", func(t *testing.T) {
		expected := `
[panda](https://github.com/panda) reopened your pull request [mattermost-plugin-github#42](https://github.com/mattermost/mattermost-plugin-github/pull/42) - Leverage git-get-head
`

		actual, err := renderTemplate("pullRequestNotification", &github.PullRequestEvent{
			Repo:        &repo,
			Action:      sToP("reopened"),
			Sender:      &user,
			Number:      iToP(42),
			PullRequest: &pullRequest,
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("assigned", func(t *testing.T) {
		expected := `
[panda](https://github.com/panda) assigned you to pull request [mattermost-plugin-github#42](https://github.com/mattermost/mattermost-plugin-github/pull/42) - Leverage git-get-head
`

		actual, err := renderTemplate("pullRequestNotification", &github.PullRequestEvent{
			Repo:        &repo,
			Action:      sToP("assigned"),
			Sender:      &user,
			Number:      iToP(42),
			PullRequest: &pullRequest,
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})
}

func TestIssueNotification(t *testing.T) {
	t.Run("closed", func(t *testing.T) {
		expected := `
[panda](https://github.com/panda) closed your issue [mattermost-plugin-github#1](https://github.com/mattermost/mattermost-plugin-github/issues/1) - Implement git-get-head
`

		actual, err := renderTemplate("issueNotification", &github.IssuesEvent{
			Repo:   &repo,
			Action: sToP("closed"),
			Sender: &user,
			Issue:  &issue,
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("reopened", func(t *testing.T) {
		expected := `
[panda](https://github.com/panda) reopened your issue [mattermost-plugin-github#1](https://github.com/mattermost/mattermost-plugin-github/issues/1) - Implement git-get-head
`

		actual, err := renderTemplate("issueNotification", &github.IssuesEvent{
			Repo:   &repo,
			Action: sToP("reopened"),
			Sender: &user,
			Issue:  &issue,
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("assigned you", func(t *testing.T) {
		expected := `
[panda](https://github.com/panda) assigned you to issue [mattermost-plugin-github#1](https://github.com/mattermost/mattermost-plugin-github/issues/1) - Implement git-get-head
`

		actual, err := renderTemplate("issueNotification", &github.IssuesEvent{
			Repo:   &repo,
			Action: sToP("assigned"),
			Sender: &user,
			Issue:  &issue,
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})
}

func TestPullRequestReviewNotification(t *testing.T) {
	t.Run("approved", func(t *testing.T) {
		expected := `
[panda](https://github.com/panda) approved your pull request [mattermost-plugin-github#42](https://github.com/mattermost/mattermost-plugin-github/pull/42#issuecomment-123456) - Leverage git-get-head
>Excited to see git-get-head land!
`

		actual, err := renderTemplate("pullRequestReviewNotification", &github.PullRequestReviewEvent{
			Repo:        &repo,
			PullRequest: &pullRequest,
			Sender:      &user,
			Review: &github.PullRequestReview{
				HTMLURL: sToP("https://github.com/mattermost/mattermost-plugin-github/pull/42#issuecomment-123456"),
				State:   sToP("approved"),
				Body:    sToP("Excited to see git-get-head land!"),
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("changes_requested", func(t *testing.T) {
		expected := `
[panda](https://github.com/panda) requested changes on your pull request [mattermost-plugin-github#42](https://github.com/mattermost/mattermost-plugin-github/pull/42#issuecomment-123456) - Leverage git-get-head
>Excited to see git-get-head land!
`

		actual, err := renderTemplate("pullRequestReviewNotification", &github.PullRequestReviewEvent{
			Repo:        &repo,
			PullRequest: &pullRequest,
			Sender:      &user,
			Review: &github.PullRequestReview{
				HTMLURL: sToP("https://github.com/mattermost/mattermost-plugin-github/pull/42#issuecomment-123456"),
				State:   sToP("changes_requested"),
				Body:    sToP("Excited to see git-get-head land!"),
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("commented", func(t *testing.T) {
		expected := `
[panda](https://github.com/panda) commented on your pull request [mattermost-plugin-github#42](https://github.com/mattermost/mattermost-plugin-github/pull/42#issuecomment-123456) - Leverage git-get-head
>Excited to see git-get-head land!
`

		actual, err := renderTemplate("pullRequestReviewNotification", &github.PullRequestReviewEvent{
			Repo:        &repo,
			PullRequest: &pullRequest,
			Sender:      &user,
			Review: &github.PullRequestReview{
				HTMLURL: sToP("https://github.com/mattermost/mattermost-plugin-github/pull/42#issuecomment-123456"),
				State:   sToP("commented"),
				Body:    sToP("Excited to see git-get-head land!"),
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})
	t.Run("approved with mentions", withGitHubUserNameMapping(func(t *testing.T) {
		expected := `
@pandabot approved your pull request [mattermost-plugin-github#42](https://github.com/mattermost/mattermost-plugin-github/pull/42#issuecomment-123456) - Leverage git-get-head
>Excited to see git-get-head land!
>` + usernameMentions + `
`

		actual, err := renderTemplate("pullRequestReviewNotification", &github.PullRequestReviewEvent{
			Repo:        &repo,
			PullRequest: &pullRequest,
			Sender:      &user,
			Review: &github.PullRequestReview{
				HTMLURL: sToP("https://github.com/mattermost/mattermost-plugin-github/pull/42#issuecomment-123456"),
				State:   sToP("approved"),
				Body:    sToP("Excited to see git-get-head land!\n" + gitHubMentions),
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}))
	t.Run("review with no body", withGitHubUserNameMapping(func(t *testing.T) {
		expected := `
@pandabot approved your pull request [mattermost-plugin-github#42](https://github.com/mattermost/mattermost-plugin-github/pull/42#issuecomment-123456) - Leverage git-get-head
`

		actual, err := renderTemplate("pullRequestReviewNotification", &github.PullRequestReviewEvent{
			Repo:        &repo,
			PullRequest: &pullRequest,
			Sender:      &user,
			Review: &github.PullRequestReview{
				HTMLURL: sToP("https://github.com/mattermost/mattermost-plugin-github/pull/42#issuecomment-123456"),
				State:   sToP("approved"),
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}))
}

func TestGitHubUsernameRegex(t *testing.T) {
	stringAndMatchMap := map[string]string{
		// Contain valid usernames
		"@u":          "@u",
		"@username":   "@username",
		"@user-name":  "@user-name",
		"@1":          "@1",
		"@1-a":        "@1-a",
		"ñ@username":  "@username",
		" @username":  "@username",
		"@username ":  "@username",
		" @username ": "@username",
		"!@username":  "@username",
		"-@username":  "@username",

		// Contain partially valid usernames
		"@user--name": "@user",
		"@username-":  "@username",
		"@user_name":  "@user",
		"@user.name":  "@user",
	}

	invalidUsernames := []string{
		"email@provider.com",
		"@-username",
		"`@user_name",
		"_@username",
	}

	for string, match := range stringAndMatchMap {
		require.Equal(t, match, gitHubUsernameRegex.FindStringSubmatch(string)[2])
	}

	for _, string := range invalidUsernames {
		require.False(t, gitHubUsernameRegex.MatchString(string))
	}
}

func sToP(s string) *string {
	return &s
}

func iToP(i int) *int {
	return &i
}

func tToP(t time.Time) *time.Time {
	return &t
}

func bToP(b bool) *bool {
	return &b
}
