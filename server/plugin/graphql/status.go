// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package graphql

import (
	"context"

	"github.com/pkg/errors"
	"github.com/shurcooL/githubv4"
)

type changeUserStatusMutation struct {
	ChangeUserStatus struct {
		Status struct {
			Message githubv4.String
		}
	} `graphql:"changeUserStatus(input: $input)"`
}

// SetBusyStatus sets the user's GitHub status to Busy, which excludes them from automatic PR assignment.
func (c *Client) SetBusyStatus(ctx context.Context, message, emoji string) error {
	return c.changeUserStatus(ctx, githubv4.ChangeUserStatusInput{
		LimitedAvailability: githubv4.NewBoolean(true),
		Message:             githubv4.NewString(githubv4.String(message)),
		Emoji:               githubv4.NewString(githubv4.String(emoji)),
	})
}

// ClearStatus removes the user's GitHub status, including the Busy indicator.
func (c *Client) ClearStatus(ctx context.Context) error {
	return c.changeUserStatus(ctx, githubv4.ChangeUserStatusInput{
		LimitedAvailability: githubv4.NewBoolean(false),
	})
}

func (c *Client) changeUserStatus(ctx context.Context, input githubv4.ChangeUserStatusInput) error {
	var m changeUserStatusMutation
	if err := c.client.Mutate(ctx, &m, input, nil); err != nil {
		return errors.Wrap(err, "error in changing user status")
	}
	return nil
}
