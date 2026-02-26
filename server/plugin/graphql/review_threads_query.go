// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package graphql

import (
	"github.com/shurcooL/githubv4"
)

type (
	reactionGroupNode struct {
		Content githubv4.String
		Users   struct {
			TotalCount githubv4.Int
		}
	}

	reviewThreadCommentNode struct {
		ID             githubv4.String
		DatabaseID     githubv4.Int
		Body           githubv4.String
		Author         authorQuery
		CreatedAt      githubv4.DateTime
		UpdatedAt      githubv4.DateTime
		URL            githubv4.URI
		DiffHunk       githubv4.String
		Path           githubv4.String
		Line           githubv4.Int
		StartLine      githubv4.Int
		ReactionGroups []reactionGroupNode
	}

	reviewThreadNode struct {
		ID         githubv4.String
		IsResolved githubv4.Boolean
		ResolvedBy authorQuery
		Comments   struct {
			Nodes    []reviewThreadCommentNode
			PageInfo struct {
				EndCursor   githubv4.String
				HasNextPage bool
			}
		} `graphql:"comments(first:100)"`
	}

	reviewSummaryNode struct {
		State  githubv4.String
		Author authorQuery
	}
)

var reviewThreadsQuery struct {
	Repository struct {
		PullRequest struct {
			Reviews struct {
				Nodes []reviewSummaryNode
			} `graphql:"reviews(first:100)"`
			ReviewThreads struct {
				Nodes    []reviewThreadNode
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"reviewThreads(first:50, after:$threadsCursor)"`
		} `graphql:"pullRequest(number:$prNumber)"`
	} `graphql:"repository(owner:$owner, name:$name)"`
}

var resolveThreadMutation struct {
	ResolveReviewThread struct {
		Thread struct {
			ID         githubv4.String
			IsResolved githubv4.Boolean
		}
	} `graphql:"resolveReviewThread(input:$input)"`
}

var unresolveThreadMutation struct {
	UnresolveReviewThread struct {
		Thread struct {
			ID         githubv4.String
			IsResolved githubv4.Boolean
		}
	} `graphql:"unresolveReviewThread(input:$input)"`
}
