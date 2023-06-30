package graphql

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v41/github"
	"github.com/pkg/errors"
	"github.com/shurcooL/githubv4"
)

const (
	queryParamReviewCursor      = "reviewCursor"
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
		queryParamReviewCursor:      (*githubv4.String)(nil),
		queryParamAssignmentsCursor: (*githubv4.String)(nil),
		queryParamOpenPRsCursor:     (*githubv4.String)(nil),
	}

	if c.org != "" {
		params[queryParamOpenPRQueryArg] = githubv4.String(fmt.Sprintf("org:%s %s", c.org, params[queryParamOpenPRQueryArg]))
		params[queryParamReviewPRQueryArg] = githubv4.String(fmt.Sprintf("org:%s %s", c.org, params[queryParamReviewPRQueryArg]))
		params[queryParamAssigneeQueryArg] = githubv4.String(fmt.Sprintf("org:%s %s", c.org, params[queryParamAssigneeQueryArg]))
	}

	var resultPR, resultAssignee, resultOpenPR []*github.Issue
	flagPR, flagAssignee, flagOpenPR := false, false, false

	for {
		if flagPR && flagOpenPR && flagAssignee {
			break
		}

		if err := c.executeQuery(ctx, &mainQuery, params); err != nil {
			return nil, nil, nil, errors.Wrap(err, "Not able to excute the query")
		}

		if !flagPR {
			for _, resp := range mainQuery.PullRequest.Nodes {
				resp := resp
				pr := getPR(&resp)
				resultPR = append(resultPR, pr)
			}

			if !mainQuery.PullRequest.PageInfo.HasNextPage {
				flagPR = true
			}

			params[queryParamReviewCursor] = githubv4.NewString(mainQuery.PullRequest.PageInfo.EndCursor)
		}

		if !flagAssignee {
			for _, resp := range mainQuery.Assignee.Nodes {
				resp := resp
				issue := getIssue(&resp)
				resultAssignee = append(resultAssignee, issue)
			}

			if !mainQuery.Assignee.PageInfo.HasNextPage {
				flagAssignee = true
			}

			params[queryParamAssignmentsCursor] = githubv4.NewString(mainQuery.Assignee.PageInfo.EndCursor)
		}

		if !flagOpenPR {
			for _, resp := range mainQuery.OpenPullRequest.Nodes {
				resp := resp
				pr := getPR(&resp)
				resultOpenPR = append(resultOpenPR, pr)
			}

			if !mainQuery.OpenPullRequest.PageInfo.HasNextPage {
				flagOpenPR = true
			}

			params[queryParamOpenPRsCursor] = githubv4.NewString(mainQuery.OpenPullRequest.PageInfo.EndCursor)
		}
	}

	return resultPR, resultAssignee, resultOpenPR, nil
}

func getPR(prResp *prSearchNodes) *github.Issue {
	resp := prResp.PullRequest

	return getGithubIssue(int(resp.Number), resp.Repository.URL.String(), string(resp.Title), (string)(resp.Author.Login), resp.URL.String(), resp.CreatedAt.Time, resp.UpdatedAt.Time)
}

func getIssue(assignmentResp *assignmentSearchNodes) *github.Issue {
	resp := assignmentResp.PullRequest

	return getGithubIssue(int(resp.Number), resp.Repository.URL.String(), string(resp.Title), (string)(resp.Author.Login), resp.URL.String(), resp.CreatedAt.Time, resp.UpdatedAt.Time)
}

func getGithubIssue(prNumber int, repositoryURL, title, login, htmlURL string, createdAt, updatedAt time.Time) *github.Issue {
	return &github.Issue{
		Number:        &prNumber,
		RepositoryURL: &repositoryURL,
		Title:         &title,
		CreatedAt:     &createdAt,
		UpdatedAt:     &updatedAt,
		User: &github.User{
			Login: &login,
		},
		HTMLURL: &htmlURL,
	}
}
