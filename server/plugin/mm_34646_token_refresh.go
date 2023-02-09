package plugin

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/go-github/v48/github"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-api/cluster"

	"github.com/mattermost/mattermost-plugin-github/server/serializer"
)

const pageSize = 100
const delayBetweenUsers = 1 * time.Second
const delayToStart = 1 * time.Minute

func (p *Plugin) forceResetAllMM34646() error {
	config := p.getConfiguration()
	ctx := context.Background()

	time.Sleep(delayToStart)
	data, appErr := p.API.KVGet(mm34646DoneKey)
	if appErr != nil {
		return errors.Wrap(appErr, "failed check whether MM-34646 refresh is already done")
	}
	if len(data) > 0 {
		// Already done
		return nil
	}

	m, err := cluster.NewMutex(p.API, mm34646MutexKey)
	if err != nil {
		return errors.Wrap(err, "failed to create mutex")
	}
	m.Lock()
	defer m.Unlock()

	for page := 0; ; page++ {
		keys, appErr := p.API.KVList(page, pageSize)
		if appErr != nil {
			return appErr
		}

		for _, key := range keys {
			data, appErr := p.API.KVGet(key)
			if appErr != nil {
				p.API.LogWarn("failed to inspect key", "key", key, "error",
					appErr.Error())
				continue
			}
			tryInfo := serializer.GitHubUserInfo{}
			err := json.Unmarshal(data, &tryInfo)
			if err != nil {
				// too noisy to report
				continue
			}
			if tryInfo.MM34646ResetTokenDone {
				continue
			}
			if tryInfo.Token == nil || tryInfo.Token.AccessToken == "" {
				// too noisy to report
				continue
			}

			info, errResp := p.getGitHubUserInfo(tryInfo.UserID)
			if errResp != nil {
				p.API.LogWarn("failed to retrieve GitHubUserInfo", "key", key, "user_id", tryInfo.UserID,
					"error", errResp.Error())
				continue
			}

			_, err = p.forceResetUserTokenMM34646(ctx, config, info)
			if err != nil {
				p.API.LogWarn("failed to reset GitHub user token", "key", key, "user_id", tryInfo.UserID,
					"error", err.Error())
				continue
			}

			time.Sleep(delayBetweenUsers)
		}

		if len(keys) == 0 {
			break
		}
	}

	_ = p.API.KVSet(mm34646DoneKey, []byte("done"))
	return nil
}

func (p *Plugin) forceResetUserTokenMM34646(ctx context.Context, config *Configuration, info *serializer.GitHubUserInfo) (string, error) {
	if info.MM34646ResetTokenDone {
		return info.Token.AccessToken, nil
	}

	client, err := p.getResetUserTokenMM34646Client(config)
	if err != nil {
		return "", errors.Wrap(err, "failed to create a special GitHub client to refresh the user's token")
	}

	a, _, err := client.Authorizations.Reset(ctx, config.GitHubOAuthClientID, info.Token.AccessToken)
	if err != nil {
		return "", errors.Wrap(err, "failed to reset GitHub token")
	}
	if a.Token == nil {
		return "", errors.Wrap(err, "failed to reset GitHub token: no token received")
	}

	info.Token.AccessToken = *a.Token
	info.MM34646ResetTokenDone = true
	err = p.storeGitHubUserInfo(info)
	if err != nil {
		return "", errors.Wrap(err, "failed to store updated serializer.GitHubUserInfo")
	}
	p.API.LogDebug("Updated user access token for MM-34646", "user_id", info.UserID)

	return *a.Token, nil
}

func (p *Plugin) getResetUserTokenMM34646Client(config *Configuration) (*github.Client, error) {
	t := &github.BasicAuthTransport{
		Username: config.GitHubOAuthClientID,
		Password: config.GitHubOAuthClientSecret,
	}
	return getGitHubClient(t.Client(), config)
}
