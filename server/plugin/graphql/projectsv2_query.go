package graphql

import "github.com/shurcooL/githubv4"

type (
	repositoryNode struct {
		ID   githubv4.String
		Name githubv4.String
	}

	projectV2Nodes struct {
		ID           githubv4.String
		Title        githubv4.String
		Repositories struct {
			Nodes []repositoryNode
		} `graphql:"repositories(first: 100)"`
	}
)

var projectsV2Query struct {
	Organization struct {
		ProjectsV2 struct {
			Nodes []projectV2Nodes
		} `graphql:"projectsV2(first:100)"`
	} `graphql:"organization(login: $organization)"`
}
