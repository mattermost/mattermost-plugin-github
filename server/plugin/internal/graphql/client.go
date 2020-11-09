package graphql

import (
	"context"
	"fmt"
	"net/url"
	"path"

	"github.com/mattermost/mattermost-plugin-github/server/plugin/internal/graphql/query"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// Client encapsulates the third party package that communicates with Github GraphQL API
type Client struct {
	client   *githubv4.Client
	org      string
	username string
}

// NewClient creates and returns Client. Third party package that queries GraphQL is initialized here.
func NewClient(token oauth2.Token, username, orgName, enterpriseBaseURL string) *Client {
	ts := oauth2.StaticTokenSource(&token)
	httpClient := oauth2.NewClient(context.Background(), ts)

	if enterpriseBaseURL == "" || orgName == "" {
		return &Client{
			client:   githubv4.NewClient(httpClient),
			username: username,
		}
	}

	baseURL, _ := url.Parse(enterpriseBaseURL)
	baseURL.Path = path.Join(baseURL.Path, "api", "graphql")

	return &Client{
		client:   githubv4.NewEnterpriseClient(baseURL.String(), httpClient),
		username: username,
		org:      orgName,
	}
}

// ExecuteQuery takes a *query.Object, transforms it to a struct and sends call to GitHub GraphQL API via helper package.
// If the call is successful, the response is converted to Result and returned.
func (c *Client) ExecuteQuery(q *query.Object) (Response, error) {
	// set org option to the main query if not empty
	if c.org != "" {
		if err := query.SetOrg(c.org)(q); err != nil {
			return nil, err
		}
	}

	builder := query.NewBuilder(q)
	qry, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("query builder error: %v", err)
	}

	err = c.client.Query(context.Background(), qry, nil)
	if err != nil {
		fmt.Println(err)
		return nil, fmt.Errorf("query execution error: %v", err)
	}

	res, err := convertToResponse(qry)
	if err != nil {
		return nil, fmt.Errorf("error converting result to map: %v", err)
	}

	return res, nil
}

// Username returns Github username
func (c *Client) Username() string {
	return c.username
}
