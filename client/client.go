// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/go-github/v54/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-github/server/plugin"
)

type PluginAPI interface {
	PluginHTTP(*http.Request) *http.Response
}

type Client struct {
	httpClient http.Client
}

type pluginAPIRoundTripper struct {
	api PluginAPI
}

func (p *pluginAPIRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp := p.api.PluginHTTP(req)
	if resp == nil {
		return nil, errors.Errorf("Failed to make interplugin request")
	}

	return resp, nil
}

func NewPluginClient(api PluginAPI) *Client {
	client := &Client{}
	client.httpClient.Transport = &pluginAPIRoundTripper{api}

	return client
}

func (c *Client) GetGitHubClient(userID string) (*github.Client, error) {
	token, err := c.GetToken(userID)
	if err != nil {
		return nil, err
	}

	config, err := c.GetConfiguration()
	if err != nil {
		return nil, err
	}

	client, err := plugin.GetGitHubClient(*token, config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Client) GetConfiguration() (*plugin.Configuration, error) {
	req, err := http.NewRequest(http.MethodGet, "/"+plugin.Manifest.Id+"/api/v1/config", http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("Unable to get Forgejo config. Error: %v, %v", resp.StatusCode, string(respBody))
	}

	config := &plugin.Configuration{}
	err = json.Unmarshal(respBody, config)
	config.ForgejoOrg = strings.TrimSpace(config.ForgejoOrg)
	config.ForgejoOAuthClientID = strings.TrimSpace(config.ForgejoOAuthClientID)
	config.ForgejoOAuthClientSecret = strings.TrimSpace(config.ForgejoOAuthClientSecret)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode Forgejo config")
	}

	return config, nil
}

func (c *Client) GetToken(userID string) (*oauth2.Token, error) {
	req, err := http.NewRequest(http.MethodGet, "/"+plugin.Manifest.Id+"/api/v1/token", http.NoBody)
	if err != nil {
		return nil, err
	}

	values := url.Values{}
	values.Add("userID", userID)
	req.URL.RawQuery = values.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("Unable to get Forgejo token. Error: %v, %v", resp.StatusCode, string(respBody))
	}

	token := &oauth2.Token{}
	err = json.Unmarshal(respBody, token)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode Forgejo token")
	}

	return token, nil
}
