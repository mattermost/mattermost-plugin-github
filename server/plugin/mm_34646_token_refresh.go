package plugin

import (
	"context"
	"time"

	"github.com/google/go-github/v54/github"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost/server/public/pluginapi/cluster"
)

const pageSize = 100
const delayBetweenUsers = 1 * time.Second
const delayToStart = 1 * time.Minute

func (p *Plugin) forceResetAllMM34646() error {
	config := p.getConfiguration()
	ctx := context.Background()

	time.Sleep(delayToStart)
	var data []byte
	err := p.store.Get(mm34646DoneKey, &data)
	if err != nil {
		return errors.Wrap(err, "failed check whether MM-34646 refresh is already done")
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
		var keys []string
		keys, err = p.store.ListKeys(page, pageSize)
		if err != nil {
			return err
		}

		for _, key := range keys {
			var tryInfo ForgejoUserInfo
			err = p.store.Get(key, &tryInfo)
			if err != nil {
				p.client.Log.Warn("failed to inspect key", "key", key, "error",
					err.Error())
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
				p.client.Log.Warn("failed to retrieve ForgejoUserInfo", "key", key, "user_id", tryInfo.UserID,
					"error", errResp.Error())
				continue
			}

			_, err = p.forceResetUserTokenMM34646(ctx, config, info)
			if err != nil {
				p.client.Log.Warn("failed to reset Forgejo user token", "key", key, "user_id", tryInfo.UserID,
					"error", err.Error())
				continue
			}

			time.Sleep(delayBetweenUsers)
		}

		if len(keys) == 0 {
			break
		}
	}

	_, err = p.store.Set(mm34646DoneKey, []byte("done"))
	if err != nil {
		return err
	}

	return nil
}

func (p *Plugin) forceResetUserTokenMM34646(ctx context.Context, config *Configuration, info *ForgejoUserInfo) (string, error) {
	if info.MM34646ResetTokenDone {
		return info.Token.AccessToken, nil
	}

	client, err := p.getResetUserTokenMM34646Client(config)
	if err != nil {
		return "", errors.Wrap(err, "failed to create a special Forgejo client to refresh the user's token")
	}

	a, _, err := client.Authorizations.Reset(ctx, config.ForgejoOAuthClientID, info.Token.AccessToken)
	if err != nil {
		return "", errors.Wrap(err, "failed to reset Forgejo token")
	}
	if a.Token == nil {
		return "", errors.Wrap(err, "failed to reset Forgejo token: no token received")
	}

	info.Token.AccessToken = *a.Token
	info.MM34646ResetTokenDone = true
	err = p.storeGitHubUserInfo(info)
	if err != nil {
		return "", errors.Wrap(err, "failed to store updated ForgejoUserInfo")
	}
	p.client.Log.Debug("Updated user access token for MM-34646", "user_id", info.UserID)

	return *a.Token, nil
}

func (p *Plugin) getResetUserTokenMM34646Client(config *Configuration) (*github.Client, error) {
	t := &github.BasicAuthTransport{
		Username: config.ForgejoOAuthClientID,
		Password: config.ForgejoOAuthClientSecret,
	}
	return getGitHubClient(t.Client(), config)
}
