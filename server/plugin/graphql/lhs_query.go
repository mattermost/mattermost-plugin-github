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

	labelNode struct {
		Name  githubv4.String
		Color githubv4.String
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
			Labels            struct {
				Nodes []labelNode
			} `graphql:"labels(first:100)"`
			Milestone struct {
				Title githubv4.String
			}
			Additions    githubv4.Int
			Deletions    githubv4.Int
			ChangedFiles githubv4.Int
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
			Labels            struct {
				Nodes []labelNode
			} `graphql:"labels(first:100)"`
			Milestone struct {
				Title githubv4.String
			}
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
			Labels            struct {
				Nodes []labelNode
			} `graphql:"labels(first:100)"`
			Milestone struct {
				Title githubv4.String
			}
			Additions    githubv4.Int
			Deletions    githubv4.Int
			ChangedFiles githubv4.Int
		} `graphql:"... on PullRequest"`
	}
)

var mainQuery struct {
	ReviewRequests struct {
		IssueCount int
		Nodes      []prSearchNodes
		PageInfo   struct {
			EndCursor   githubv4.String
			HasNextPage bool
		}
	} `graphql:"pullRequest: search(first:100, after:$reviewsCursor, query: $prReviewQueryArg, type: ISSUE)"`

	Assignments struct {
		IssueCount int
		Nodes      []assignmentSearchNodes
		PageInfo   struct {
			EndCursor   githubv4.String
			HasNextPage bool
		}
	} `graphql:"assignee: search(first:100, after:$assignmentsCursor, query: $assigneeQueryArg, type: ISSUE)"`

	OpenPullRequests struct {
		IssueCount int
		Nodes      []prSearchNodes
		PageInfo   struct {
			EndCursor   githubv4.String
			HasNextPage bool
		}
	} `graphql:"graphql: search(first:100, after:$openPrsCursor, query: $prOpenQueryArg, type: ISSUE)"`
}
