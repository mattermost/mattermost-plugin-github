// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package graphql

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/v54/github"
	pkgerrors "github.com/pkg/errors"
	"github.com/shurcooL/githubv4"
)

const (
	queryParamReviewsCursor     = "reviewsCursor"
	queryParamAssignmentsCursor = "assignmentsCursor"
	queryParamOpenPRsCursor     = "openPrsCursor"

	queryParamOpenPRQueryArg   = "prOpenQueryArg"
	queryParamReviewPRQueryArg = "prReviewQueryArg"
	queryParamAssigneeQueryArg = "assigneeQueryArg"
)

type GithubPRDetails struct {
	*github.Issue
	Additions    *githubv4.Int `json:"additions,omitempty"`
	Deletions    *githubv4.Int `json:"deletions,omitempty"`
	ChangedFiles *githubv4.Int `json:"changed_files,omitempty"`
	// ReviewSLAStartAt is RFC3339 UTC time used for SLA when the plugin knows review request time (webhook); clients may fall back to created_at.
	ReviewSLAStartAt *string `json:"review_sla_start,omitempty"`
}

func (c *Client) GetLHSData(ctx context.Context) ([]*GithubPRDetails, []*github.Issue, []*GithubPRDetails, error) {
	orgsList := c.getOrganizations()
	var resultAssignee []*github.Issue
	var resultReview, resultOpenPR []*GithubPRDetails

	var firstErr error
	for _, org := range orgsList {
		var err error
		resultReview, resultAssignee, resultOpenPR, err = c.fetchLHSData(ctx, resultReview, resultAssignee, resultOpenPR, org, c.username)
		if err != nil {
			c.logger.Error("Error fetching LHS data for org", "org", org, "error", err.Error())
			firstErr = preferAuthErr(firstErr, err)
		}
	}

	if len(orgsList) == 0 {
		return c.fetchLHSData(ctx, resultReview, resultAssignee, resultOpenPR, "", c.username)
	}

	// Return partial results alongside the error so callers can detect auth failures
	// while still rendering whatever orgs succeeded.
	return resultReview, resultAssignee, resultOpenPR, firstErr
}

func preferAuthErr(existing, candidate error) error {
	if existing == nil {
		return candidate
	}
	if candidate == nil {
		return existing
	}
	if isLikelyAuthErr(candidate) && !isLikelyAuthErr(existing) {
		return candidate
	}
	return existing
}

func isLikelyAuthErr(err error) bool {
	if err == nil {
		return false
	}
	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) && ghErr.Response != nil {
		if ghErr.Response.StatusCode == http.StatusUnauthorized {
			return true
		}
		if ghErr.Response.StatusCode == http.StatusForbidden {
			if ghErr.Response.Header.Get("X-GitHub-SSO") != "" {
				return true
			}
			if strings.Contains(strings.ToLower(ghErr.Message), "saml") {
				return true
			}
		}
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "bad credentials") ||
		strings.Contains(msg, "non-200 ok status code: 401") ||
		strings.Contains(msg, "saml")
}

func (c *Client) fetchLHSData(
	ctx context.Context,
	resultReview []*GithubPRDetails,
	resultAssignee []*github.Issue,
	resultOpenPR []*GithubPRDetails,
	org string,
	username string,
) ([]*GithubPRDetails, []*github.Issue, []*GithubPRDetails, error) {
	baseOpenPR := fmt.Sprintf("author:%s is:pr is:%s archived:false", username, githubv4.PullRequestStateOpen)
	baseReviewPR := fmt.Sprintf("review-requested:%s is:pr is:%s archived:false", username, githubv4.PullRequestStateOpen)
	baseAssignee := fmt.Sprintf("assignee:%s is:%s archived:false", username, githubv4.PullRequestStateOpen)

	if org != "" {
		baseOpenPR = fmt.Sprintf("org:%s %s", org, baseOpenPR)
		baseReviewPR = fmt.Sprintf("org:%s %s", org, baseReviewPR)
		baseAssignee = fmt.Sprintf("org:%s %s", org, baseAssignee)
	}

	params := map[string]any{
		queryParamOpenPRQueryArg:    githubv4.String(baseOpenPR),
		queryParamReviewPRQueryArg:  githubv4.String(baseReviewPR),
		queryParamAssigneeQueryArg:  githubv4.String(baseAssignee),
		queryParamReviewsCursor:     (*githubv4.String)(nil),
		queryParamAssignmentsCursor: (*githubv4.String)(nil),
		queryParamOpenPRsCursor:     (*githubv4.String)(nil),
	}

	allReviewRequestsFetched, allAssignmentsFetched, allOpenPRsFetched := false, false, false

	for !allReviewRequestsFetched || !allAssignmentsFetched || !allOpenPRsFetched {
		if err := c.executeQuery(ctx, &mainQuery, params); err != nil {
			return nil, nil, nil, pkgerrors.Wrap(err, "Not able to execute the query")
		}

		if !allReviewRequestsFetched {
			for i := range mainQuery.ReviewRequests.Nodes {
				resp := mainQuery.ReviewRequests.Nodes[i]
				pr := getPR(&resp)
				resultReview = append(resultReview, pr)
			}

			if !mainQuery.ReviewRequests.PageInfo.HasNextPage {
				allReviewRequestsFetched = true
			}

			params[queryParamReviewsCursor] = githubv4.NewString(mainQuery.ReviewRequests.PageInfo.EndCursor)
		}

		if !allAssignmentsFetched {
			for i := range mainQuery.Assignments.Nodes {
				resp := mainQuery.Assignments.Nodes[i]
				issue := newIssueFromAssignmentResponse(&resp)
				resultAssignee = append(resultAssignee, issue)
			}

			if !mainQuery.Assignments.PageInfo.HasNextPage {
				allAssignmentsFetched = true
			}

			params[queryParamAssignmentsCursor] = githubv4.NewString(mainQuery.Assignments.PageInfo.EndCursor)
		}

		if !allOpenPRsFetched {
			for i := range mainQuery.OpenPullRequests.Nodes {
				resp := mainQuery.OpenPullRequests.Nodes[i]
				pr := getPR(&resp)
				resultOpenPR = append(resultOpenPR, pr)
			}

			if !mainQuery.OpenPullRequests.PageInfo.HasNextPage {
				allOpenPRsFetched = true
			}

			params[queryParamOpenPRsCursor] = githubv4.NewString(mainQuery.OpenPullRequests.PageInfo.EndCursor)
		}
	}

	return resultReview, resultAssignee, resultOpenPR, nil
}

func getPR(prResp *prSearchNodes) *GithubPRDetails {
	resp := prResp.PullRequest
	labels := getGithubLabels(resp.Labels.Nodes)

	number := int(resp.Number)
	repoURL := resp.Repository.URL.String()
	issuetitle := string(resp.Title)
	userLogin := string(resp.Author.Login)
	milestoneTitle := string(resp.Milestone.Title)
	url := resp.URL.String()
	createdAtTime := github.Timestamp{Time: resp.CreatedAt.Time}
	updatedAtTime := github.Timestamp{Time: resp.UpdatedAt.Time}

	return &GithubPRDetails{
		Issue: &github.Issue{
			Number:        &number,
			RepositoryURL: &repoURL,
			Title:         &issuetitle,
			CreatedAt:     &createdAtTime,
			UpdatedAt:     &updatedAtTime,
			User: &github.User{
				Login: &userLogin,
			},
			Milestone: &github.Milestone{
				Title: &milestoneTitle,
			},
			HTMLURL: &url,
			Labels:  labels,
		},
		Additions:    &resp.Additions,
		Deletions:    &resp.Deletions,
		ChangedFiles: &resp.ChangedFiles,
	}
}

func newIssueFromAssignmentResponse(assignmentResp *assignmentSearchNodes) *github.Issue {
	resp := assignmentResp.PullRequest
	labels := getGithubLabels(resp.Labels.Nodes)

	return newGithubIssue(resp.Number, resp.Title, resp.Author.Login, resp.Repository.URL, resp.URL, resp.CreatedAt, resp.UpdatedAt, labels, resp.Milestone.Title)
}

func getGithubLabels(labels []labelNode) []*github.Label {
	githubLabels := []*github.Label{}
	for _, label := range labels {
		name := (string)(label.Name)
		color := (string)(label.Color)
		githubLabels = append(githubLabels, &github.Label{
			Color: &color,
			Name:  &name,
		})
	}

	return githubLabels
}

func newGithubIssue(prNumber githubv4.Int, title, login githubv4.String, repositoryURL, htmlURL githubv4.URI, createdAt, updatedAt githubv4.DateTime, labels []*github.Label, milestone githubv4.String) *github.Issue {
	number := int(prNumber)
	repoURL := repositoryURL.String()
	issuetitle := string(title)
	userLogin := string(login)
	milestoneTitle := string(milestone)
	url := htmlURL.String()
	createdAtTime := github.Timestamp{Time: createdAt.Time}
	updatedAtTime := github.Timestamp{Time: updatedAt.Time}

	return &github.Issue{
		Number:        &number,
		RepositoryURL: &repoURL,
		Title:         &issuetitle,
		CreatedAt:     &createdAtTime,
		UpdatedAt:     &updatedAtTime,
		User: &github.User{
			Login: &userLogin,
		},
		Milestone: &github.Milestone{
			Title: &milestoneTitle,
		},
		HTMLURL: &url,
		Labels:  labels,
	}
}
