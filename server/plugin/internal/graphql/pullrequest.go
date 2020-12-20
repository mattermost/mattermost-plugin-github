package graphql

import (
	"github.com/mattermost/mattermost-plugin-github/server/plugin/internal/graphql/model"
)

type PullRequestService service

func (p *PullRequestService) Get() ([]model.PullRequest, error) {
	return nil, nil
}
