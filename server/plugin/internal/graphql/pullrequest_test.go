package graphql

import (
	"net/url"
	"reflect"
	"testing"

	"github.com/shurcooL/githubv4"
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

	if err != nil {
		t.Fatalf("PullRequests.Get() err: %v", err)
	}

	t.Logf("PullRequests.Get(): %v", res)
}

func TestPullRequestService_Get_response(t *testing.T) {
	url1, _ := url.Parse("https://mattermost.com")
	url2, _ := url.Parse("http://github.com/mattermost/mattermost-plugin-github")
	userJaneDoe := userQuery{
		Login:      *githubv4.NewString("first_author"),
		Name:       *githubv4.NewString("jane doe"),
		AvatarURL:  githubv4.URI{URL: url1},
		URL:        githubv4.URI{URL: url1},
		DatabaseID: *githubv4.NewInt(123),
		ID:         *githubv4.NewID(123123),
	}

	searchQuery := prSearchQuery{
		Search: struct {
			IssueCount githubv4.Int
			Nodes      []prSearchNodes
		}{
			IssueCount: *githubv4.NewInt(2),
			Nodes: []prSearchNodes{
				{
					PullRequest: struct {
						ID             githubv4.ID
						Body           githubv4.String
						Mergeable      githubv4.MergeableState
						Number         githubv4.Int
						Repository     repositoryQuery
						Reviews        reviewsQuery `graphql:"reviews(first: 10)"`
						ReviewDecision githubv4.PullRequestReviewDecision
						ReviewRequests reviewRequestsQuery `graphql:"reviewRequests(first: 10)"`
						State          githubv4.PullRequestState
						Title          githubv4.String
						URL            githubv4.URI
					}{
						ID:        *githubv4.NewID(1),
						Body:      *githubv4.NewString("Commit details of the first PR."),
						Mergeable: githubv4.MergeableStateMergeable,
						Number:    *githubv4.NewInt(101),
						Repository: repositoryQuery{
							Name:          "mattermost-plugin-github v1",
							NameWithOwner: "mattermost",
						},
						Reviews: reviewsQuery{
							TotalCount: 1,
							Nodes: []struct {
								ID         githubv4.ID
								DatabaseID githubv4.Int
								State      githubv4.PullRequestReviewState
								Body       githubv4.String
								URL        githubv4.URI
								Author     struct {
									User userQuery `graphql:"... on User"`
								}
							}{
								{
									ID:         *githubv4.NewID(1234),
									DatabaseID: *githubv4.NewInt(123),
									State:      githubv4.PullRequestReviewStateApproved,
									Body:       *githubv4.NewString("first pr review body"),
									URL:        githubv4.URI{URL: url1},
									Author: struct {
										User userQuery `graphql:"... on User"`
									}{
										User: userJaneDoe,
									},
								},
							},
						},
						ReviewDecision: githubv4.PullRequestReviewDecisionReviewRequired,
						ReviewRequests: reviewRequestsQuery{
							TotalCount: 1,
							Nodes: []struct {
								RequestedReviewer struct {
									User userQuery `graphql:"... on User"`
								}
							}{
								{
									RequestedReviewer: struct {
										User userQuery `graphql:"... on User"`
									}{
										User: userJaneDoe,
									},
								},
							},
						},
						State: githubv4.PullRequestStateOpen,
						Title: *githubv4.NewString("First Pull Request"),
						URL:   githubv4.URI{URL: url1},
					},
				},
				{
					PullRequest: struct {
						ID             githubv4.ID
						Body           githubv4.String
						Mergeable      githubv4.MergeableState
						Number         githubv4.Int
						Repository     repositoryQuery
						Reviews        reviewsQuery `graphql:"reviews(first: 10)"`
						ReviewDecision githubv4.PullRequestReviewDecision
						ReviewRequests reviewRequestsQuery `graphql:"reviewRequests(first: 10)"`
						State          githubv4.PullRequestState
						Title          githubv4.String
						URL            githubv4.URI
					}{
						ID:        *githubv4.NewID(2),
						Body:      *githubv4.NewString("Commit details of the second PR."),
						Mergeable: githubv4.MergeableStateUnknown,
						Number:    *githubv4.NewInt(102),
						Repository: repositoryQuery{
							Name:          "mattermost-plugin-github v1",
							NameWithOwner: "mattermost",
						},
						Reviews: reviewsQuery{
							TotalCount: 1,
							Nodes: []struct {
								ID         githubv4.ID
								DatabaseID githubv4.Int
								State      githubv4.PullRequestReviewState
								Body       githubv4.String
								URL        githubv4.URI
								Author     struct {
									User userQuery `graphql:"... on User"`
								}
							}{
								{
									ID:         *githubv4.NewID(987),
									DatabaseID: *githubv4.NewInt(987),
									State:      githubv4.PullRequestReviewStateChangesRequested,
									Body:       *githubv4.NewString("second pr review body here"),
									URL:        githubv4.URI{URL: url2},
									Author: struct {
										User userQuery `graphql:"... on User"`
									}{
										User: userJaneDoe,
									},
								},
							},
						},
						ReviewDecision: githubv4.PullRequestReviewDecisionReviewRequired,
						ReviewRequests: reviewRequestsQuery{
							TotalCount: 2,
							Nodes: []struct {
								RequestedReviewer struct {
									User userQuery `graphql:"... on User"`
								}
							}{
								{
									RequestedReviewer: struct {
										User userQuery `graphql:"... on User"`
									}{
										User: userJaneDoe,
									},
								},
								{
									RequestedReviewer: struct {
										User userQuery `graphql:"... on User"`
									}{
										User: userQuery{
											Login:      *githubv4.NewString("second_author"),
											Name:       "John Doe",
											AvatarURL:  githubv4.URI{URL: url2},
											URL:        githubv4.URI{URL: url2},
											DatabaseID: *githubv4.NewInt(456),
											ID:         *githubv4.NewID(456456),
										},
									},
								},
							},
						},
						State: githubv4.PullRequestStateOpen,
						Title: *githubv4.NewString("Second Pull Request"),
						URL:   githubv4.URI{URL: url2},
					},
				},
			}},
	}

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

	prService := PullRequestService{client: &Client{}}
	prService.test.isTest = true
	prService.test.clientResponseMock = searchQuery

	got, err := prService.Get()
	if err != nil {
		t.Errorf("convertPrResponse() error = %v", err)
		return
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("convertPrResponse() got = %v\n\nwant %v", got, want)
	}
}
