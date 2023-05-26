package graphql

import (
	"github.com/shurcooL/githubv4"
)

type (
	repositoryQuery struct {
		Name          githubv4.String
		NameWithOwner githubv4.String
		URL           githubv4.URI
	}

	authorQuery struct {
		Login githubv4.String
	}

	prSearchNodes struct {
		PullRequest struct {
			Body              githubv4.String
			Number            githubv4.Int
			AuthorAssociation githubv4.String
			CreatedAt         githubv4.DateTime
			UpdatedAt         githubv4.DateTime
			Repository        repositoryQuery
			State             githubv4.String
			Title             githubv4.String
			Author            authorQuery
			URL               githubv4.URI
		} `graphql:"... on PullRequest"`
	}
)

type (
	assignmentSearchNodes struct {
		Issue struct {
			Body              githubv4.String
			Number            githubv4.Int
			AuthorAssociation githubv4.String
			CreatedAt         githubv4.DateTime
			UpdatedAt         githubv4.DateTime
			Repository        repositoryQuery
			State             githubv4.String
			Title             githubv4.String
			Author            authorQuery
			URL               githubv4.URI
		} `graphql:"... on Issue"`

		PullRequest struct {
			Body              githubv4.String
			Number            githubv4.Int
			AuthorAssociation githubv4.String
			CreatedAt         githubv4.DateTime
			UpdatedAt         githubv4.DateTime
			Repository        repositoryQuery
			State             githubv4.String
			Title             githubv4.String
			Author            authorQuery
			URL               githubv4.URI
		} `graphql:"... on PullRequest"`
	}
)

var mainQuery struct {
	PullRequest struct {
		IssueCount int
		Nodes      []prSearchNodes
		PageInfo   struct {
			EndCursor   githubv4.String
			HasNextPage bool
		}
	} `graphql:"pullRequest: search(first:100, after:$reviewCursor, query: $prReviewQueryArg, type: ISSUE)"`

	Assignee struct {
		IssueCount int
		Nodes      []assignmentSearchNodes
		PageInfo   struct {
			EndCursor   githubv4.String
			HasNextPage bool
		}
	} `graphql:"assignee: search(first:100, after:$assignmentsCursor, query: $assigneeQueryArg, type: ISSUE)"`

	OpenPullRequest struct {
		IssueCount int
		Nodes      []prSearchNodes
		PageInfo   struct {
			EndCursor   githubv4.String
			HasNextPage bool
		}
	} `graphql:"graphql: search(first:100, after:$openPrsCursor, query: $prOpenQueryArg, type: ISSUE)"`
}
