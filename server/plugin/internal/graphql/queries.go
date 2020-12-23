package graphql

import (
	"github.com/shurcooL/githubv4"
)

type (
	userQuery struct {
		Login      githubv4.String
		Name       githubv4.String
		AvatarURL  githubv4.URI
		URL        githubv4.URI
		DatabaseID githubv4.Int
		ID         githubv4.ID
	}

	reviewsQuery struct {
		TotalCount githubv4.Int
		Nodes      []struct {
			ID         githubv4.ID
			DatabaseID githubv4.Int
			State      githubv4.PullRequestReviewState
			Body       githubv4.String
			URL        githubv4.URI
			Author     struct {
				User userQuery `graphql:"... on User"`
			}
		}
	}

	reviewRequestsQuery struct {
		TotalCount githubv4.Int
		Nodes      []struct {
			RequestedReviewer struct {
				User userQuery `graphql:"... on User"`
			}
		}
	}

	repositoryQuery struct {
		Name          githubv4.String
		NameWithOwner githubv4.String
	}

	prSearchNodes struct {
		PullRequest struct {
			ID             githubv4.ID
			Body           githubv4.String
			Mergeable      githubv4.MergeableState
			Number         githubv4.Int
			Repository     repositoryQuery
			Reviews        reviewsQuery `graphql:"reviews(first: 10)"`
			ReviewDecision githubv4.PullRequestReviewDecision
			ReviewRequests reviewRequestsQuery `graphql:"reviewRequests(first: 10)"`
			State          githubv4.PullRequestState
			Title          githubv4.String
			URL            githubv4.URI
		} `graphql:"... on PullRequest"`
	}

	prSearchQuery struct {
		Search struct {
			IssueCount githubv4.Int
			Nodes      []prSearchNodes
		} `graphql:"search(first: 100, query: $prSearchQueryArg, type: ISSUE)"`
	}
)
