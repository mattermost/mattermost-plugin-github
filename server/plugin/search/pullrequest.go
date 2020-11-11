package search

import (
	"fmt"

	"github.com/mattermost/mattermost-plugin-github/server/plugin/internal/graphql"
	"github.com/mattermost/mattermost-plugin-github/server/plugin/search/model"
)

func GetPRDetail(graphQLClient *graphql.Client) ([]model.PullRequest, error) {
	var res []model.PullRequest
	q, err := getPrsSearchQuery(graphQLClient.Username())
	if err != nil {
		return nil, err
	}

	response, err := graphQLClient.ExecuteQuery(q)
	if err != nil {
		return res, err
	}

	searchResult := response.GetResponseObject("Search")
	if searchResult == nil {
		return res, fmt.Errorf("field not found: Search")
	}

	if searchResult.GetChildType("Nodes") != graphql.TypeResponseSlice {
		return res, fmt.Errorf("unexpected field type: Nodes")
	}

	prNodes, _ := searchResult.Get("Nodes")
	for _, node := range prNodes.([]graphql.Response) {
		var requestedReviewers []string
		pr := node.GetResponseObject("PullRequest")

		if reviewRequests, err := pr.GetResponseObject("ReviewRequests").Get("Nodes"); err == nil {
			for _, rr := range reviewRequests.([]graphql.Response) {
				user := rr.GetResponseObject("RequestedReviewer").GetResponseObject("User")
				requestedReviewers = append(requestedReviewers, user.GetString("Login"))
			}
		}

		var reviews []model.PullRequestReview
		if result, err := pr.GetResponseObject("Reviews").Get("Nodes"); err == nil {
			for _, r := range result.([]graphql.Response) {
				author := r.GetResponseObject("Author").GetResponseObject("User")
				user := model.User{
					ID:        author.GetInt64("DatabaseID"),
					Login:     author.GetString("Login"),
					NodeID:    author.GetString("ID"),
					AvatarURL: author.GetString("AvatarURL"),
					HTMLURL:   author.GetString("URL"),
					Name:      author.GetString("Name"),
				}

				review := model.PullRequestReview{
					ID:     r.GetInt64("DatabaseID"),
					NodeID: r.GetString("ID"),
					User:   &user,
					Body:   r.GetString("Body"),
					State:  r.GetString("State"),
					URL:    r.GetString("URL"),
				}
				reviews = append(reviews, review)
			}
		}

		mergeableState := pr.GetString("Mergeable")

		v := model.PullRequest{
			URL:                pr.GetString("URL"),
			Number:             pr.GetInt64("Number"),
			Status:             pr.GetString("State"),
			Mergeable:          mergeableState == "MERGEABLE",
			MergeableState:     mergeableState,
			RequestedReviewers: requestedReviewers,
			Reviews:            reviews,
		}

		res = append(res, v)
	}

	return res, nil
}
