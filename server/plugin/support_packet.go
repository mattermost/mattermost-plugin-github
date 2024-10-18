package plugin

import (
	"path"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

type SupportPacket struct {
	Version string `yaml:"version"`

	ConnectedUserCount int64 `yaml:"connected_user_count"`
	IsOAuthConfigured  bool  `yaml:"is_oauth_configured"`
}

func (p *Plugin) GenerateSupportData(_ *plugin.Context) ([]*model.FileData, error) {
	var result *multierror.Error

	config := p.getConfiguration()

	connectedUserCount, err := p.getConnectedUserCount()
	if err != nil {
		result = multierror.Append(result, errors.Wrap(err, "Failed to get the number of connected users for Support Packet"))
	}

	diagnostics := SupportPacket{
		Version:            Manifest.Version,
		ConnectedUserCount: connectedUserCount,
		IsOAuthConfigured:  config.IsOAuthConfigured(),
	}
	b, err := yaml.Marshal(diagnostics)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to marshal diagnostics")
	}

	return []*model.FileData{{
		Filename: path.Join(Manifest.Id, "diagnostics.yaml"),
		Body:     b,
	}}, result.ErrorOrNil()
}
