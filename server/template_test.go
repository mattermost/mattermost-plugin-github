package main

import (
	"testing"
	"time"

	"github.com/google/go-github/github"
	"github.com/stretchr/testify/require"
)

var repo = github.Repository{
	FullName: sToP("mattermost-plugin-github"),
	HTMLURL:  sToP("https://github.com/mattermost/mattermost-plugin-github"),
}

var pushEventRepository = github.PushEventRepository{
	FullName: sToP("mattermost-plugin-github"),
	HTMLURL:  sToP("https://github.com/mattermost/mattermost-plugin-github"),
}

var pullRequest = github.PullRequest{
	Number:    iToP(42),
	HTMLURL:   sToP("https://github.com/mattermost/mattermost-plugin-github/pull/42"),
	Title:     sToP("Leverage git-get-head"),
	CreatedAt: tToP(time.Date(2019, 04, 01, 02, 03, 04, 0, time.UTC)),
	UpdatedAt: tToP(time.Date(2019, 05, 01, 02, 03, 04, 0, time.UTC)),
	Body:      sToP("git-get-head gets the non-sent upstream heads inside the stashed non-cleaned applied areas, and after pruning bases to many archives, you can initialize the origin of the bases."),
}

var mergedPullRequest = github.PullRequest{
	Number:    iToP(42),
	HTMLURL:   sToP("https://github.com/mattermost/mattermost-plugin-github/pull/42"),
	Title:     sToP("Leverage git-get-head"),
	CreatedAt: tToP(time.Date(2019, 04, 01, 02, 03, 04, 0, time.UTC)),
	UpdatedAt: tToP(time.Date(2019, 05, 01, 02, 03, 04, 0, time.UTC)),
	Body:      sToP("git-get-head gets the non-sent upstream heads inside the stashed non-cleaned applied areas, and after pruning bases to many archives, you can initialize the origin of the bases."),
	Merged:    bToP(true),
}

var issue = github.Issue{
	Number:    iToP(1),
	HTMLURL:   sToP("https://github.com/mattermost/mattermost-plugin-github/issues/1"),
	Title:     sToP("Implement git-get-head"),
	CreatedAt: tToP(time.Date(2019, 04, 01, 02, 03, 04, 0, time.UTC)),
	UpdatedAt: tToP(time.Date(2019, 05, 01, 02, 03, 04, 0, time.UTC)),
	Body:      sToP("git-get-head sounds like a great feature we should support"),
}

var user = github.User{
	Login:   sToP("panda"),
	HTMLURL: sToP("https://github.com/panda"),
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

	t.Run("Mattermost username", func(t *testing.T) {
		gitHubToUsernameMappingCallback = func(githubUsername string) string {
			return "pandabot"
		}
		defer func() {
			gitHubToUsernameMappingCallback = nil
		}()

		expected := "@pandabot"
		actual, err := renderTemplate("user", &user)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})
}

func TestNewPRMessageTemplate(t *testing.T) {
	expected := `
#### Leverage git-get-head
##### [mattermost-plugin-github#42](https://github.com/mattermost/mattermost-plugin-github/pull/42)
#new-pull-request by [panda](https://github.com/panda) on [2019-04-01 02:03:04 +0000 UTC](https://github.com/mattermost/mattermost-plugin-github/pull/42)

git-get-head gets the non-sent upstream heads inside the stashed non-cleaned applied areas, and after pruning bases to many archives, you can initialize the origin of the bases.
`

	actual, err := renderTemplate("newPR", &github.PullRequestEvent{
		Repo:        &repo,
		PullRequest: &pullRequest,
		Sender:      &user,
	})
	require.NoError(t, err)
	require.Equal(t, expected, actual)
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
#pull-request-labeled ` + "`label-name`" + ` by [panda](https://github.com/panda) on [2019-05-01 02:03:04 +0000 UTC](https://github.com/mattermost/mattermost-plugin-github/pull/42)
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
	expected := `
#### Implement git-get-head
##### [mattermost-plugin-github#1](https://github.com/mattermost/mattermost-plugin-github/issues/1)
#new-issue by [panda](https://github.com/panda) on [2019-04-01 02:03:04 +0000 UTC](https://github.com/mattermost/mattermost-plugin-github/issues/1)

git-get-head sounds like a great feature we should support
`

	actual, err := renderTemplate("newIssue", &github.IssuesEvent{
		Repo:   &repo,
		Issue:  &issue,
		Sender: &user,
	})
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestClosedIssueTemplate(t *testing.T) {
	expected := `
[\[mattermost-plugin-github\]](https://github.com/mattermost/mattermost-plugin-github) Issue [#1 Implement git-get-head](https://github.com/mattermost/mattermost-plugin-github/issues/1) closed by [panda](https://github.com/panda).
`

	actual, err := renderTemplate("closedIssue", &github.IssuesEvent{
		Repo:   &repo,
		Issue:  &issue,
		Sender: &user,
	})
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestIssueLabelledTemplate(t *testing.T) {
	expected := `
#### Implement git-get-head
##### [mattermost-plugin-github#1](https://github.com/mattermost/mattermost-plugin-github/issues/1)
#issue-labeled ` + "`label-name`" + ` by [panda](https://github.com/panda) on [2019-05-01 02:03:04 +0000 UTC](https://github.com/mattermost/mattermost-plugin-github/issues/1).
`

	actual, err := renderTemplate("issueLabelled", &github.IssuesEvent{
		Repo:  &repo,
		Issue: &issue,
		Label: &github.Label{
			Name: sToP("label-name"),
		},
		Sender: &user,
	})
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
			Commits: []github.PushEventCommit{
				github.PushEventCommit{
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
			Commits: []github.PushEventCommit{
				github.PushEventCommit{
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
			Commits: []github.PushEventCommit{
				github.PushEventCommit{
					ID:      sToP("a10867b14bb761a232cd80139fbd4c0d33264240"),
					URL:     sToP("https://github.com/mattermost/mattermost-plugin-github/commit/a10867b14bb761a232cd80139fbd4c0d33264240"),
					Message: sToP("Leverage git-get-head"),
					Committer: &github.CommitAuthor{
						Name: sToP("panda"),
					},
				},
				github.PushEventCommit{
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
			Commits: []github.PushEventCommit{
				github.PushEventCommit{
					ID:      sToP("a10867b14bb761a232cd80139fbd4c0d33264240"),
					URL:     sToP("https://github.com/mattermost/mattermost-plugin-github/commit/a10867b14bb761a232cd80139fbd4c0d33264240"),
					Message: sToP("Leverage git-get-head"),
					Committer: &github.CommitAuthor{
						Name: sToP("panda"),
					},
				},
				github.PushEventCommit{
					ID:      sToP("a20867b14bb761a232cd80139fbd4c0d33264240"),
					URL:     sToP("https://github.com/mattermost/mattermost-plugin-github/commit/a20867b14bb761a232cd80139fbd4c0d33264240"),
					Message: sToP("Merge master"),
					Committer: &github.CommitAuthor{
						Name: sToP("panda"),
					},
				},
				github.PushEventCommit{
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
[panda](https://github.com/panda) just created branch [\[mattermost-plugin-github:branch\]](https://github.com/mattermost/mattermost-plugin-github/tree/branch)
`

	actual, err := renderTemplate("newCreateMessage", &github.CreateEvent{
		Repo:    &repo,
		Ref:     sToP("branch"),
		RefType: sToP("branch"),
		Sender:  &user,
	})
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestDeletedMessageTemplate(t *testing.T) {
	expected := `
[panda](https://github.com/panda) just deleted branch \[mattermost-plugin-github:branch]
`

	actual, err := renderTemplate("newDeleteMessage", &github.DeleteEvent{
		Repo:    &repo,
		Ref:     sToP("branch"),
		RefType: sToP("branch"),
		Sender:  &user,
	})
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestIssueCommentTemplate(t *testing.T) {
	t.Run("non-email body", func(t *testing.T) {
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

	t.Run("email body", func(t *testing.T) {
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
}

func TestPullRequestReviewCommentEventTemplate(t *testing.T) {
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
}

func TestCommentMentionNotificationTemplate(t *testing.T) {
	t.Run("non-email body", func(t *testing.T) {
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
	t.Run("email body", func(t *testing.T) {
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
}

func TestCommentAuthorPullRequestNotificationTemplate(t *testing.T) {
	expected := `
[panda](https://github.com/panda) commented on your pull request [mattermost-plugin-github#1](https://github.com/mattermost/mattermost-plugin-github/issues/1) - Implement git-get-head:
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
}

func TestCommentAuthorIssueNotificationTemplate(t *testing.T) {
	expected := `
[panda](https://github.com/panda) commented on your issue [mattermost-plugin-github#1](https://github.com/mattermost/mattermost-plugin-github/issues/1) - Implement git-get-head
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
[panda](https://github.com/panda) approved your pull request [mattermost-plugin-github#42](https://github.com/mattermost/mattermost-plugin-github/pull/42) - Leverage git-get-head:
>Excited to see git-get-head land!
`

		actual, err := renderTemplate("pullRequestReviewNotification", &github.PullRequestReviewEvent{
			Repo:        &repo,
			PullRequest: &pullRequest,
			Sender:      &user,
			Review: &github.PullRequestReview{
				State: sToP("approved"),
				Body:  sToP("Excited to see git-get-head land!"),
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("changes_requested", func(t *testing.T) {
		expected := `
[panda](https://github.com/panda) requested changes on your pull request [mattermost-plugin-github#42](https://github.com/mattermost/mattermost-plugin-github/pull/42) - Leverage git-get-head:
>Excited to see git-get-head land!
`

		actual, err := renderTemplate("pullRequestReviewNotification", &github.PullRequestReviewEvent{
			Repo:        &repo,
			PullRequest: &pullRequest,
			Sender:      &user,
			Review: &github.PullRequestReview{
				State: sToP("changes_requested"),
				Body:  sToP("Excited to see git-get-head land!"),
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("commented", func(t *testing.T) {
		expected := `
[panda](https://github.com/panda) commented on your pull request [mattermost-plugin-github#42](https://github.com/mattermost/mattermost-plugin-github/pull/42) - Leverage git-get-head:
>Excited to see git-get-head land!
`

		actual, err := renderTemplate("pullRequestReviewNotification", &github.PullRequestReviewEvent{
			Repo:        &repo,
			PullRequest: &pullRequest,
			Sender:      &user,
			Review: &github.PullRequestReview{
				State: sToP("commented"),
				Body:  sToP("Excited to see git-get-head land!"),
			},
		})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})
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
