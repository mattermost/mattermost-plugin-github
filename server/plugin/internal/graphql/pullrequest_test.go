package graphql

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/shurcooL/githubv4"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-github/server/plugin/internal/graphql/model"
)

func TestPullRequestService_Get(t *testing.T) {
	if userToken == "" || username == "" {
		t.Skipf("empty username or access token, skipping test")
	}

	tok := oauth2.Token{AccessToken: userToken}
	client := NewClient(tok, username, "", "")
	res, err := client.PullRequests.Get()

	if assert.NoError(t, err, "PullRequests.Get()") {
		t.Logf("PullRequests.Get(): %v", res)
	}
}

func TestPullRequestService_Get_response(t *testing.T) {
	url1, _ := url.Parse("https://mattermost.com")
	url2, _ := url.Parse("http://github.com/mattermost/mattermost-plugin-github")
	want := []model.PullRequest{
		{
			URL:                url1.String(),
			Number:             101,
			Title:              "First Pull Request",
			Status:             "OPEN",
			Mergeable:          true,
			MergeableState:     "MERGEABLE",
			RequestedReviewers: []string{"first_author"},
			Reviews: []model.PullRequestReview{
				{
					ID:     123,
					NodeID: "1234",
					User: &model.User{
						ID:        123,
						Login:     "first_author",
						NodeID:    "123123",
						AvatarURL: url1.String(),
						HTMLURL:   url1.String(),
						Name:      "jane doe",
					},
					Body:  "first pr review body",
					State: "APPROVED",
					URL:   url1.String(),
				},
			},
		},
		{
			URL:                url2.String(),
			Number:             102,
			Title:              "Second Pull Request",
			Status:             "OPEN",
			Mergeable:          false,
			MergeableState:     "UNKNOWN",
			RequestedReviewers: []string{"first_author", "second_author"},
			Reviews: []model.PullRequestReview{
				{
					ID:     987,
					NodeID: "987",
					User: &model.User{
						ID:        123,
						Login:     "first_author",
						NodeID:    "123123",
						AvatarURL: url1.String(),
						HTMLURL:   url1.String(),
						Name:      "jane doe",
					},
					Body:  "second pr review body here",
					State: "CHANGES_REQUESTED",
					URL:   url2.String(),
				},
			},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{
			"data": {
				"search": {
					"issueCount": 2,
					"nodes": [
						{
							"id": "1",
							"body": "Commit details of the first PR.",
							"mergeable": "MERGEABLE",
							"number": 101,
							"repository": {
								"name": "mattermost-plugin-github v1",
								"nameWithOwner": "mattermost"
							},
							"reviews": {
								"totalCount": 1,
								"nodes": [
									{
										"id": "1234",
										"databaseId": 123,
										"url": "https://mattermost.com",
										"state": "APPROVED",
										"body": "first pr review body",
										"author": {
											"id": "123123",
											"avatarUrl": "https://mattermost.com",
											"login": "first_author",
											"name": "jane doe",
											"databaseId": 123,
											"url": "https://mattermost.com"
										}
									}
								]
							},
							"reviewDecision": "REVIEW_REQUIRED",
							"reviewRequests": {
								"totalCount": 1,
								"nodes": [
									{
										"requestedReviewer": {
											"id": "123123",
											"avatarUrl": "https://mattermost.com",
											"login": "first_author",
											"name": "jane doe",
											"databaseId": 123,
											"url": "https://mattermost.com"
										}
									}
								]
							},
							"state": "OPEN",
							"title": "First Pull Request",
							"url": "https://mattermost.com"
						},
						{
							"id": "2",
							"body": "Commit details of the second PR.",
							"mergeable": "UNKNOWN",
							"number": 102,
							"repository": {
								"name": "mattermost-plugin-github v1",
								"nameWithOwner": "mattermost"
							},
							"reviews": {
								"totalCount": 1,
								"nodes": [
									{
										"id": "987",
										"databaseId": 987,
										"url": "http://github.com/mattermost/mattermost-plugin-github",
										"state": "CHANGES_REQUESTED",
										"body": "second pr review body here",
										"author": {
											"id": "123123",
											"avatarUrl": "https://mattermost.com",
											"login": "first_author",
											"name": "jane doe",
											"databaseId": 123,
											"url": "https://mattermost.com"
										}
									}
								]
							},
							"reviewDecision": "REVIEW_REQUIRED",
							"reviewRequests": {
								"totalCount": 2,
								"nodes": [
									{
										"requestedReviewer": {
											"id": "123123",
											"avatarUrl": "https://mattermost.com",
											"login": "first_author",
											"name": "jane doe",
											"databaseId": 123,
											"url": "https://mattermost.com"
										}
									},
									{
										"requestedReviewer": {
											"id": "456456",
											"avatarUrl": "http://github.com/mattermost/mattermost-plugin-github",
											"login": "second_author",
											"name": "John Doe",
											"databaseId": 456123,
											"url": "http://github.com/mattermost/mattermost-plugin-github"
										}
									}
								]
							},
							"state": "OPEN",
							"title": "Second Pull Request",
							"url": "http://github.com/mattermost/mattermost-plugin-github"
						}
					]
				}
			}
		}`)
	})

	mockServer := httptest.NewServer(mux)
	defer mockServer.Close()

	client := Client{client: githubv4.NewEnterpriseClient(mockServer.URL, mockServer.Client())}
	prService := PullRequestService{client: &client}

	got, err := prService.Get()
	if !assert.NoError(t, err, "convertPrResponse()") {
		return
	}

	assert.Equal(t, want, got)
}
