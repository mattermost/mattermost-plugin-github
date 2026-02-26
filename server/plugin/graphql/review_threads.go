// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package graphql

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/shurcooL/githubv4"
)

// ReactionInfo holds reaction data for a review comment.
type ReactionInfo struct {
	Content string `json:"content"`
	Count   int    `json:"count"`
}

// ReviewComment represents a single comment within a review thread.
type ReviewComment struct {
	ID             string         `json:"id"`
	DatabaseID     int            `json:"database_id"`
	Body           string         `json:"body"`
	AuthorLogin    string         `json:"author_login"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	URL            string         `json:"url"`
	DiffHunk       string         `json:"diff_hunk"`
	Path           string         `json:"path"`
	Line           int            `json:"line"`
	StartLine      int            `json:"start_line"`
	ReactionGroups []ReactionInfo `json:"reaction_groups,omitempty"`
}

// ReviewThread represents a review thread on a pull request.
type ReviewThread struct {
	ID              string          `json:"id"`
	IsResolved      bool            `json:"is_resolved"`
	ResolvedByLogin string          `json:"resolved_by_login,omitempty"`
	Comments        []ReviewComment `json:"comments"`
}

// PRReviewSummary represents a review summary (approval, changes requested, etc.)
type PRReviewSummary struct {
	State       string `json:"state"`
	AuthorLogin string `json:"author_login"`
}

// ReviewThreadsResult holds the complete result from fetching review threads.
type ReviewThreadsResult struct {
	Threads          []ReviewThread    `json:"threads"`
	ReviewSummaries  []PRReviewSummary `json:"review_summaries"`
	UnresolvedCount  int               `json:"unresolved_count"`
	TotalThreadCount int               `json:"total_thread_count"`
}

// GetReviewThreads fetches all review threads and review summaries for a pull request.
func (c *Client) GetReviewThreads(ctx context.Context, owner, name string, prNumber int) (*ReviewThreadsResult, error) {
	var allThreads []ReviewThread
	var reviewSummaries []PRReviewSummary

	params := map[string]any{
		"owner":         githubv4.String(owner),
		"name":          githubv4.String(name),
		"prNumber":      githubv4.Int(prNumber),
		"threadsCursor": (*githubv4.String)(nil),
	}

	reviewSummariesFetched := false

	for {
		if err := c.executeQuery(ctx, &reviewThreadsQuery, params); err != nil {
			return nil, errors.Wrap(err, "failed to fetch review threads")
		}

		pr := reviewThreadsQuery.Repository.PullRequest

		// Only collect review summaries on the first page (they are not paginated by threadsCursor).
		if !reviewSummariesFetched {
			for i := range pr.Reviews.Nodes {
				node := pr.Reviews.Nodes[i]
				reviewSummaries = append(reviewSummaries, PRReviewSummary{
					State:       string(node.State),
					AuthorLogin: string(node.Author.Login),
				})
			}
			reviewSummariesFetched = true
		}

		for i := range pr.ReviewThreads.Nodes {
			threadNode := pr.ReviewThreads.Nodes[i]
			thread := ReviewThread{
				ID:              string(threadNode.ID),
				IsResolved:      bool(threadNode.IsResolved),
				ResolvedByLogin: string(threadNode.ResolvedBy.Login),
			}

			for j := range threadNode.Comments.Nodes {
				commentNode := threadNode.Comments.Nodes[j]
				comment := ReviewComment{
					ID:          string(commentNode.ID),
					DatabaseID:  int(commentNode.DatabaseID),
					Body:        string(commentNode.Body),
					AuthorLogin: string(commentNode.Author.Login),
					CreatedAt:   commentNode.CreatedAt.Time,
					UpdatedAt:   commentNode.UpdatedAt.Time,
					URL:         commentNode.URL.String(),
					DiffHunk:    string(commentNode.DiffHunk),
					Path:        string(commentNode.Path),
					Line:        int(commentNode.Line),
					StartLine:   int(commentNode.StartLine),
				}

				for k := range commentNode.ReactionGroups {
					rg := commentNode.ReactionGroups[k]
					comment.ReactionGroups = append(comment.ReactionGroups, ReactionInfo{
						Content: string(rg.Content),
						Count:   int(rg.Users.TotalCount),
					})
				}

				thread.Comments = append(thread.Comments, comment)
			}

			allThreads = append(allThreads, thread)
		}

		if !pr.ReviewThreads.PageInfo.HasNextPage {
			break
		}

		params["threadsCursor"] = githubv4.NewString(pr.ReviewThreads.PageInfo.EndCursor)
	}

	unresolvedCount := 0
	for i := range allThreads {
		if !allThreads[i].IsResolved {
			unresolvedCount++
		}
	}

	return &ReviewThreadsResult{
		Threads:          allThreads,
		ReviewSummaries:  reviewSummaries,
		UnresolvedCount:  unresolvedCount,
		TotalThreadCount: len(allThreads),
	}, nil
}

// ResolveReviewThread resolves or unresolves a review thread by its GraphQL node ID.
// It returns the new isResolved state.
func (c *Client) ResolveReviewThread(ctx context.Context, threadID string, resolve bool) (bool, error) {
	if resolve {
		input := githubv4.ResolveReviewThreadInput{
			ThreadID: githubv4.ID(threadID),
		}

		if err := c.executeMutation(ctx, &resolveThreadMutation, input, nil); err != nil {
			return false, errors.Wrap(err, "failed to resolve review thread")
		}

		return bool(resolveThreadMutation.ResolveReviewThread.Thread.IsResolved), nil
	}

	input := githubv4.UnresolveReviewThreadInput{
		ThreadID: githubv4.ID(threadID),
	}

	if err := c.executeMutation(ctx, &unresolveThreadMutation, input, nil); err != nil {
		return false, errors.Wrap(err, "failed to unresolve review thread")
	}

	return bool(unresolveThreadMutation.UnresolveReviewThread.Thread.IsResolved), nil
}
