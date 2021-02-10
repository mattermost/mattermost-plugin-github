package graphql

import (
	"context"
	"net/url"
	"path"

	"github.com/pkg/errors"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type service struct {
	client *Client
}

// Client encapsulates the third party package that communicates with Github GraphQL API
type Client struct {
	client   *githubv4.Client
	org      string
	username string

	PullRequests *PullRequestService
}

// NewClient creates and returns Client. Third party package that queries GraphQL is initialized here.
func NewClient(token oauth2.Token, username, orgName, enterpriseBaseURL string) *Client {
	ts := oauth2.StaticTokenSource(&token)
	httpClient := oauth2.NewClient(context.Background(), ts)
	var client Client

	if enterpriseBaseURL == "" || orgName == "" {
		client = Client{
			username: username,
			client:   githubv4.NewClient(httpClient),
		}
	} else {
		baseURL, _ := url.Parse(enterpriseBaseURL)
		baseURL.Path = path.Join(baseURL.Path, "api", "graphql")

		client = Client{
			client:   githubv4.NewEnterpriseClient(baseURL.String(), httpClient),
			username: username,
			org:      orgName,
		}
	}

	// For reuse
	common := service{client: &client}
	// Set services
	client.PullRequests = (*PullRequestService)(&common)

	return &client
}

// executeQuery takes a query struct and sends it to GitHub GraphQL API via helper package.
func (c *Client) executeQuery(qry interface{}, params map[string]interface{}) error {
	if err := c.client.Query(context.Background(), qry, params); err != nil {
		return errors.Wrap(err, "query execution error")
	}
	return nil
}
