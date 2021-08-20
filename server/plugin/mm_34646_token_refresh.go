package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/pkg/errors"
)

func (p *Plugin) forceRefreshUserTokens(ctx context.Context) error {
	config := p.getConfiguration()
	if ctx == nil {
		ctx = context.Background()
	}

	client, err := getGitHubClient(&http.Client{
		Transport: &basicAuthTransport{
			ClientID: config.GitHubOAuthClientID,
			Secret:   config.GitHubOAuthClientSecret,
		},
	}, config)
	if err != nil {
		return errors.Wrap(err, "failed to get GitHub client")
	}

	for page := 0; ; page++ {
		keys, appErr := p.API.KVList(page, 100)
		if appErr != nil {
			return appErr
		}
		if len(keys) == 0 {
			break
		}

		for _, key := range keys {
			fmt.Printf("<>/<> KEY %q\n", key)

			data, appErr := p.API.KVGet(key)
			if appErr != nil {
				p.API.LogWarn("failed to inspect key", "key", key, "error", err.Error())
				continue
			}
			info := GitHubUserInfo{}
			err = json.Unmarshal(data, &info)
			if err != nil {
				p.API.LogDebug("key failed to unmarshal as GitHubUserInfo", "key", key, "error", err.Error())
				continue
			}
			if info.Token == nil || info.Token.AccessToken == "" {
				p.API.LogDebug("skipping key with no token", "key", key)
				continue
			}

			req, err := client.NewRequest(http.MethodPatch,
				"/applications/"+config.GitHubOAuthClientID+"/token",
				map[string]string{
					"access_token": info.Token.AccessToken,
				},
			)


			bb, _ := httputil.DumpRequest(req, true)
			fmt.Printf("<>/<> REQUEST:\n%v\n", string(bb))

			if err != nil {
				p.API.LogDebug("failed to compose GitHub request", "key", key, "error", err.Error())
				continue
			}
			m := map[string]interface{}{}
			_, err = client.Do(ctx, req, &m)
			if err != nil {
				p.API.LogError("failed to refresh token", "key", key, "user_id", info.UserID, "error", err.Error())
				continue
			}

			fmt.Printf("<>/<> RESPONSE %+v\n", m)
		}
	}
	return nil
}

type basicAuthTransport struct {
	ClientID string
	Secret   string
}

func (t basicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(t.ClientID, t.Secret)
	return http.DefaultTransport.RoundTrip(req)
}
