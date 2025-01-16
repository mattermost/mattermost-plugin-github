package graphql

import (
	"context"
	"net/url"

	"github.com/pkg/errors"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost/server/public/pluginapi"
)

// Client encapsulates the third party package that communicates with Github GraphQL API
type Client struct {
	client           *githubv4.Client
	org              string
	username         string
	logger           pluginapi.LogService
	getOrganizations func() []string
}

// NewClient creates and returns Client. The third party package that queries GraphQL is initialized here.
func NewClient(logger pluginapi.LogService, getOrganizations func() []string, token oauth2.Token, username, orgName, enterpriseBaseURL string) *Client {
	ts := oauth2.StaticTokenSource(&token)
	httpClient := oauth2.NewClient(context.Background(), ts)
	var client Client

	if enterpriseBaseURL == "" {
		client = Client{
			username:         username,
			client:           githubv4.NewClient(httpClient),
			logger:           logger,
			org:              orgName,
			getOrganizations: getOrganizations,
		}
	} else {
		baseURL, err := url.JoinPath(enterpriseBaseURL, "api", "graphql")
		if err != nil {
			logger.Debug("Not able to parse the enterprise URL", "error", err.Error())
			return nil
		}

		client = Client{
			client:           githubv4.NewEnterpriseClient(baseURL, httpClient),
			username:         username,
			org:              orgName,
			logger:           logger,
			getOrganizations: getOrganizations,
		}
	}

	return &client
}

// executeQuery takes a query struct and sends it to Github GraphQL API via helper package.
func (c *Client) executeQuery(ctx context.Context, qry interface{}, params map[string]interface{}) error {
	if err := c.client.Query(ctx, qry, params); err != nil {
		return errors.Wrap(err, "error in executing query")
	}

	return nil
}
