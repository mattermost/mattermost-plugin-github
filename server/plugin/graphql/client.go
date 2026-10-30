// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

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
	if enterpriseBaseURL == "" {
		return &Client{
			username:         username,
			client:           githubv4.NewClient(oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&token))),
			logger:           logger,
			org:              orgName,
			getOrganizations: getOrganizations,
		}
	}

	baseURL, err := url.JoinPath(enterpriseBaseURL, "api", "graphql")
	if err != nil {
		logger.Debug("Not able to parse the enterprise URL", "error", err.Error())
		return nil
	}

	return NewClientWithURL(logger, getOrganizations, token, username, orgName, baseURL)
}

// NewClientWithURL creates a Client for the GraphQL endpoint at graphQLURL.
func NewClientWithURL(logger pluginapi.LogService, getOrganizations func() []string, token oauth2.Token, username, orgName, graphQLURL string) *Client {
	return &Client{
		client:           githubv4.NewEnterpriseClient(graphQLURL, oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&token))),
		username:         username,
		org:              orgName,
		logger:           logger,
		getOrganizations: getOrganizations,
	}
}

// executeQuery takes a query struct and sends it to Github GraphQL API via helper package.
func (c *Client) executeQuery(ctx context.Context, qry any, params map[string]any) error {
	if err := c.client.Query(ctx, qry, params); err != nil {
		return errors.Wrap(err, "error in executing query")
	}

	return nil
}
