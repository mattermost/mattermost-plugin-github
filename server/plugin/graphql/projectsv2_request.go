package graphql

import (
	"context"

	"github.com/google/go-github/v54/github"
	"github.com/pkg/errors"
	"github.com/shurcooL/githubv4"
)

const (
	queryParamOrganization = "organization"
)

func (c *Client) GetProjectsV2Data(ctx context.Context, owner string) ([]*github.ProjectsV2, error) {
	var projects []*github.ProjectsV2

	params := map[string]interface{}{
		queryParamOrganization: githubv4.String(owner),
	}

	if err := c.executeQuery(ctx, &projectsV2Query, params); err != nil {
		return nil, errors.Wrap(err, "Failed to execute the ProjectsV2 query")
	}

	for i := range projectsV2Query.Organization.ProjectsV2.Nodes {
		resp := &projectsV2Query.Organization.ProjectsV2.Nodes[i]

		nodeID := string(resp.ID)
		title := string(resp.Title)
		project := github.ProjectsV2{
			NodeID: &nodeID,
			Title:  &title,
		}

		projects = append(projects, &project)
	}

	return projects, nil
}
