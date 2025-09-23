// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package graphql

import (
	"context"
	"fmt"

	"github.com/google/go-github/v54/github"
	"github.com/pkg/errors"
	"github.com/shurcooL/githubv4"
)

const (
	queryParamReviewsCursor     = "reviewsCursor"
	queryParamAssignmentsCursor = "assignmentsCursor"
	queryParamOpenPRsCursor     = "openPrsCursor"
	queryParamMentionsCursor    = "mentionsCursor"

	queryParamOpenPRQueryArg   = "prOpenQueryArg"
	queryParamReviewPRQueryArg = "prReviewQueryArg"
	queryParamAssigneeQueryArg = "assigneeQueryArg"
	queryParamMentionsQueryArg = "prMentionsQueryArg"
)

type GithubPRDetails struct {
	*github.Issue
	Additions    *githubv4.Int `json:"additions,omitempty"`
	Deletions    *githubv4.Int `json:"deletions,omitempty"`
	ChangedFiles *githubv4.Int `json:"changed_files,omitempty"`
}

func (c *Client) GetLHSData(ctx context.Context) ([]*GithubPRDetails, []*github.Issue, []*GithubPRDetails, []*GithubPRDetails, error) {
	orgsList := c.getOrganizations()
	var resultAssignee []*github.Issue
	var resultReview, resultOpenPR, resultMentions []*GithubPRDetails

	params := map[string]interface{}{
		queryParamOpenPRQueryArg:    githubv4.String(fmt.Sprintf("author:%s is:pr is:%s archived:false", c.username, githubv4.PullRequestStateOpen)),
		queryParamReviewPRQueryArg:  githubv4.String(fmt.Sprintf("review-requested:%s is:pr is:%s archived:false", c.username, githubv4.PullRequestStateOpen)),
		queryParamAssigneeQueryArg:  githubv4.String(fmt.Sprintf("assignee:%s is:%s archived:false", c.username, githubv4.PullRequestStateOpen)),
		queryParamMentionsQueryArg:  githubv4.String(fmt.Sprintf("mentions:%s is:%s is:pr archived:false", c.username, githubv4.PullRequestStateOpen)),
		queryParamReviewsCursor:     (*githubv4.String)(nil),
		queryParamAssignmentsCursor: (*githubv4.String)(nil),
		queryParamOpenPRsCursor:     (*githubv4.String)(nil),
		queryParamMentionsCursor:    (*githubv4.String)(nil),
	}

	var err error
	for _, org := range orgsList {
		resultReview, resultAssignee, resultOpenPR, resultMentions, err = c.fetchLHSData(ctx, resultReview, resultAssignee, resultOpenPR, resultMentions, org, params)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}

	if len(orgsList) == 0 {
		return c.fetchLHSData(ctx, resultReview, resultAssignee, resultOpenPR, resultMentions, "", params)
	}

	return resultReview, resultAssignee, resultOpenPR, resultMentions, nil
}

func (c *Client) fetchLHSData(ctx context.Context, resultReview []*GithubPRDetails, resultAssignee []*github.Issue, resultOpenPR []*GithubPRDetails, resultMentions []*GithubPRDetails, org string, params map[string]interface{}) ([]*GithubPRDetails, []*github.Issue, []*GithubPRDetails, []*GithubPRDetails, error) {
	if org != "" {
		params[queryParamOpenPRQueryArg] = githubv4.String(fmt.Sprintf("org:%s %s", org, params[queryParamOpenPRQueryArg]))
		params[queryParamReviewPRQueryArg] = githubv4.String(fmt.Sprintf("org:%s %s", org, params[queryParamReviewPRQueryArg]))
		params[queryParamAssigneeQueryArg] = githubv4.String(fmt.Sprintf("org:%s %s", org, params[queryParamAssigneeQueryArg]))
		params[queryParamMentionsQueryArg] = githubv4.String(fmt.Sprintf("org:%s %s", c.org, params[queryParamMentionsQueryArg]))
	}

	allReviewRequestsFetched, allAssignmentsFetched, allOpenPRsFetched, allMentionsFetched := false, false, false, false

	for {
		if allReviewRequestsFetched && allAssignmentsFetched && allOpenPRsFetched && allMentionsFetched {
			break
		}

		if err := c.executeQuery(ctx, &mainQuery, params); err != nil {
			return nil, nil, nil, nil, errors.Wrap(err, "Not able to excute the query")
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

		if !allMentionsFetched {
			for i := range mainQuery.Mentions.Nodes {
				resp := mainQuery.Mentions.Nodes[i]
				pr := getPR(&resp)
				resultMentions = append(resultMentions, pr)
			}

			if !mainQuery.Mentions.PageInfo.HasNextPage {
				allMentionsFetched = true
			}

			params[queryParamMentionsCursor] = githubv4.NewString(mainQuery.Mentions.PageInfo.EndCursor)
		}
	}

	return resultReview, resultAssignee, resultOpenPR, resultMentions, nil
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
