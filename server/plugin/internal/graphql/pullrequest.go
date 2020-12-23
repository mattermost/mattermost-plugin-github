package graphql

import (
	"fmt"

	"github.com/shurcooL/githubv4"

	"github.com/mattermost/mattermost-plugin-github/server/plugin/internal/graphql/model"
)

// PullRequestService is responsible for handling pull request related queries
type PullRequestService service

// Get retrieves pull requests and returns them as []model.PullRequest
func (p *PullRequestService) Get() ([]model.PullRequest, error) {
	params := map[string]interface{}{
		"prSearchQueryArg": githubv4.String(fmt.Sprintf("author:%s is:pr is:%s archived:false", p.client.username, githubv4.PullRequestStateOpen)),
	}
	if p.client.org != "" {
		params["prSearchQueryArg"] = githubv4.String(fmt.Sprintf("org: %s %s", p.client.org, params["prSearchQueryArg"]))
	}

	var query prSearchQuery
	var res []model.PullRequest

	if p.test.isTest {
		query = p.test.clientResponseMock.(prSearchQuery)
	} else if err := p.client.executeQuery(&query, params); err != nil {
		return res, err
	}

	for _, resp := range query.Search.Nodes {
		var requestedReviewers []string
		var reviews []model.PullRequestReview

		for _, rr := range resp.PullRequest.ReviewRequests.Nodes {
			requestedReviewers = append(requestedReviewers, string(rr.RequestedReviewer.User.Login))
		}

		for _, rw := range resp.PullRequest.Reviews.Nodes {
			review := model.PullRequestReview{
				ID:     int64(rw.DatabaseID),
				NodeID: fmt.Sprintf("%v", rw.ID),
				User: &model.User{
					ID:        int64(rw.Author.User.DatabaseID),
					Login:     string(rw.Author.User.Login),
					NodeID:    fmt.Sprintf("%v", rw.Author.User.ID),
					AvatarURL: rw.Author.User.AvatarURL.String(),
					HTMLURL:   rw.Author.User.URL.String(),
					Name:      string(rw.Author.User.Name),
				},
				Body:  string(rw.Body),
				State: string(rw.State),
				URL:   rw.URL.String(),
			}
			reviews = append(reviews, review)
		}

		pr := model.PullRequest{
			URL:                resp.PullRequest.URL.String(),
			Number:             int64(resp.PullRequest.Number),
			Title:              string(resp.PullRequest.Title),
			Status:             string(resp.PullRequest.State),
			Mergeable:          resp.PullRequest.Mergeable == githubv4.MergeableStateMergeable,
			MergeableState:     string(resp.PullRequest.Mergeable),
			RequestedReviewers: requestedReviewers,
			Reviews:            reviews,
		}

		res = append(res, pr)
	}

	return res, nil
}
