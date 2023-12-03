package graphql

import (
	"context"
	"fmt"

	"github.com/google/go-github/v41/github"
	"github.com/pkg/errors"
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

func (c *Client) GetLHSData(ctx context.Context) ([]*github.Issue, []*github.Issue, []*github.Issue, error) {
	params := map[string]interface{}{
		queryParamOpenPRQueryArg:    githubv4.String(fmt.Sprintf("author:%s is:pr is:%s archived:false", c.username, githubv4.PullRequestStateOpen)),
		queryParamReviewPRQueryArg:  githubv4.String(fmt.Sprintf("review-requested:%s is:pr is:%s archived:false", c.username, githubv4.PullRequestStateOpen)),
		queryParamAssigneeQueryArg:  githubv4.String(fmt.Sprintf("assignee:%s is:%s archived:false", c.username, githubv4.PullRequestStateOpen)),
		queryParamReviewsCursor:     (*githubv4.String)(nil),
		queryParamAssignmentsCursor: (*githubv4.String)(nil),
		queryParamOpenPRsCursor:     (*githubv4.String)(nil),
	}

	if c.org != "" {
		params[queryParamOpenPRQueryArg] = githubv4.String(fmt.Sprintf("org:%s %s", c.org, params[queryParamOpenPRQueryArg]))
		params[queryParamReviewPRQueryArg] = githubv4.String(fmt.Sprintf("org:%s %s", c.org, params[queryParamReviewPRQueryArg]))
		params[queryParamAssigneeQueryArg] = githubv4.String(fmt.Sprintf("org:%s %s", c.org, params[queryParamAssigneeQueryArg]))
	}

	var resultReview, resultAssignee, resultOpenPR []*github.Issue
	allReviewRequestsFetched, allAssignmentsFetched, allOpenPRsFetched := false, false, false

	for {
		if allReviewRequestsFetched && allAssignmentsFetched && allOpenPRsFetched {
			break
		}

		if err := c.executeQuery(ctx, &mainQuery, params); err != nil {
			return nil, nil, nil, errors.Wrap(err, "Not able to excute the query")
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
				issue := getIssue(&resp)
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

func getPR(prResp *prSearchNodes) *github.Issue {
	resp := prResp.PullRequest

	return getGithubIssue(resp.Number, resp.Title, resp.Author.Login, resp.Repository.URL, resp.URL, resp.CreatedAt, resp.UpdatedAt)
}

func getIssue(assignmentResp *assignmentSearchNodes) *github.Issue {
	resp := assignmentResp.PullRequest

	return getGithubIssue(resp.Number, resp.Title, resp.Author.Login, resp.Repository.URL, resp.URL, resp.CreatedAt, resp.UpdatedAt)
}

func getGithubIssue(prNumber githubv4.Int, title, login githubv4.String, repositoryURL, htmlURL githubv4.URI, createdAt, updatedAt githubv4.DateTime) *github.Issue {
	number := int(prNumber)
	repoURL := repositoryURL.String()
	issuetitle := string(title)
	userLogin := (string)(login)
	url := htmlURL.String()
	createdAtTime := createdAt.Time
	updatedAtTime := updatedAt.Time

	return &github.Issue{
		Number:        &number,
		RepositoryURL: &repoURL,
		Title:         &issuetitle,
		CreatedAt:     &createdAtTime,
		UpdatedAt:     &updatedAtTime,
		User: &github.User{
			Login: &userLogin,
		},
		HTMLURL: &url,
	}
}
