// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package graphql

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/shurcooL/githubv4"
)

// DigestPR is a single open non-draft PR returned by GetOpenPRsWithRequestedReviewers,
// flattened from the GraphQL search response so callers don't need to know about githubv4.
type DigestPR struct {
	Owner          string
	Repo           string
	Number         int
	Title          string
	URL            string
	CreatedAt      time.Time
	RequestedUsers []string
	RequestedTeams []DigestTeamRef
}

// DigestTeamRef identifies a team review request that the caller can expand to member logins.
type DigestTeamRef struct {
	Org  string // organization login
	Slug string // team slug
}

type digestRequestedReviewer struct {
	Type githubv4.String `graphql:"__typename"`
	User struct {
		Login githubv4.String
	} `graphql:"... on User"`
	Team struct {
		Slug         githubv4.String
		Organization struct {
			Login githubv4.String
		}
	} `graphql:"... on Team"`
}

type digestPRSearchNode struct {
	PullRequest struct {
		Number     githubv4.Int
		Title      githubv4.String
		URL        githubv4.URI
		CreatedAt  githubv4.DateTime
		Repository struct {
			Name  githubv4.String
			Owner struct {
				Login githubv4.String
			}
		}
		ReviewRequests struct {
			Nodes []struct {
				RequestedReviewer digestRequestedReviewer
			}
		} `graphql:"reviewRequests(first:100)"`
	} `graphql:"... on PullRequest"`
}

// orgOpenPRsSearchQuery is the response shape for GetOpenPRsWithRequestedReviewers. Defined
// at package scope so the graphql library can reflect on its tags, but instantiated locally
// per call so concurrent callers (e.g. the digest scheduler running alongside an LHS fetch)
// don't share mutable state.
type orgOpenPRsSearchQuery struct {
	Search struct {
		Nodes    []digestPRSearchNode
		PageInfo struct {
			EndCursor   githubv4.String
			HasNextPage bool
		}
	} `graphql:"search(first:100, after:$cursor, query:$query, type:ISSUE)"`
}

// GetOpenPRsWithRequestedReviewers returns every open non-draft PR in org along with the
// users and teams currently requested for review on each one. Pages through the GitHub
// search API at 100 PRs per call.
func (c *Client) GetOpenPRsWithRequestedReviewers(ctx context.Context, org string) ([]DigestPR, error) {
	if org == "" {
		return nil, errors.New("org is required for org-wide PR search")
	}

	query := fmt.Sprintf("is:pr is:open archived:false draft:false org:%s", org)
	params := map[string]any{
		"query":  githubv4.String(query),
		"cursor": (*githubv4.String)(nil),
	}

	var orgOpenPRsQuery orgOpenPRsSearchQuery
	var out []DigestPR
	for {
		if err := c.executeQuery(ctx, &orgOpenPRsQuery, params); err != nil {
			return nil, errors.Wrapf(err, "org-wide PR search failed for org %q", org)
		}

		for _, node := range orgOpenPRsQuery.Search.Nodes {
			pr := DigestPR{
				Owner:     string(node.PullRequest.Repository.Owner.Login),
				Repo:      string(node.PullRequest.Repository.Name),
				Number:    int(node.PullRequest.Number),
				Title:     string(node.PullRequest.Title),
				URL:       node.PullRequest.URL.String(),
				CreatedAt: node.PullRequest.CreatedAt.Time,
			}
			for _, rr := range node.PullRequest.ReviewRequests.Nodes {
				switch string(rr.RequestedReviewer.Type) {
				case "User":
					if login := string(rr.RequestedReviewer.User.Login); login != "" {
						pr.RequestedUsers = append(pr.RequestedUsers, login)
					}
				case "Team":
					orgLogin := string(rr.RequestedReviewer.Team.Organization.Login)
					slug := string(rr.RequestedReviewer.Team.Slug)
					if orgLogin == "" {
						orgLogin = org
					}
					if slug != "" {
						pr.RequestedTeams = append(pr.RequestedTeams, DigestTeamRef{Org: orgLogin, Slug: slug})
					}
				}
			}
			out = append(out, pr)
		}

		if !orgOpenPRsQuery.Search.PageInfo.HasNextPage {
			break
		}
		params["cursor"] = githubv4.NewString(orgOpenPRsQuery.Search.PageInfo.EndCursor)
	}
	return out, nil
}
