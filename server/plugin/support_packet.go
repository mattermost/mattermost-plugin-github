// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

const (
	keysPerPage = 1000
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
		result = multierror.Append(result, errors.Wrap(err, "failed to get the number of connected users for Support Packet"))
	}

	diagnostics := SupportPacket{
		Version:            Manifest.Version,
		ConnectedUserCount: connectedUserCount,
		IsOAuthConfigured:  config.IsOAuthConfigured(),
	}
	body, err := yaml.Marshal(diagnostics)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal diagnostics")
	}

	return []*model.FileData{{
		Filename: filepath.Join(Manifest.Id, "diagnostics.yaml"),
		Body:     body,
	}}, result.ErrorOrNil()
}

func (p *Plugin) getConnectedUserCount() (int64, error) {
	checker := func(key string) (keep bool, err error) {
		return strings.HasSuffix(key, githubTokenKey), nil
	}

	var count int64

	for i := 0; ; i++ {
		keys, err := p.store.ListKeys(i, keysPerPage, pluginapi.WithChecker(checker))
		if err != nil {
			return 0, errors.Wrapf(err, "failed to list keys - page, %d", i)
		}

		count += int64(len(keys))

		if len(keys) < keysPerPage {
			break
		}
	}

	return count, nil
}
