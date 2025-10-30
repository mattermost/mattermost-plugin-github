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

type changeUserStatusMutation struct {
	ChangeUserStatus struct {
		Status struct {
			Message githubv4.String
			Emoji   githubv4.String
		}
	} `graphql:"changeUserStatus(input: $input)"`
}

func (c *Client) UpdateUserStatus(ctx context.Context, emoji, message string, busy bool) (string, error) {
	var mutation changeUserStatusMutation
	input := githubv4.ChangeUserStatusInput{
		Emoji:   githubv4.NewString(githubv4.String(emoji)),
		Message: githubv4.NewString(githubv4.String(message)),
		LimitedAvailability: githubv4.NewBoolean(githubv4.Boolean(busy)),
	}

	err := c.client.Mutate(ctx, &mutation, input, nil)
	if err != nil {
		return "", err
	}

	return string(mutation.ChangeUserStatus.Status.Message), nil
}

type getUserStatusQuery struct {
	User struct {
		Status struct {
			Message             githubv4.String
			Emoji               githubv4.String
			LimitedAvailability githubv4.Boolean
		}
	} `graphql:"user(login: $login)"`
}

func (c *Client) GetUserStatus(ctx context.Context, login string) (string, string, bool, error) {
	var query getUserStatusQuery
	variables := map[string]interface{}{
		"login": githubv4.String(login),
	}

	err := c.client.Query(ctx, &query, variables)
	if err != nil {
		return "", "", false, err
	}

	return string(query.User.Status.Message), string(query.User.Status.Emoji), bool(query.User.Status.LimitedAvailability), nil
}
