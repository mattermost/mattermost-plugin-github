package client

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-github/server/plugin"
)

func TestRoundTripper(t *testing.T) {
	t.Run("Valid response", func(t *testing.T) {
		pluginAPI := &plugintest.API{}
		pluginAPI.On("PluginHTTP", mock.AnythingOfType("*http.Request")).Return(&http.Response{StatusCode: http.StatusOK})

		roundTripper := pluginAPIRoundTripper{api: pluginAPI}
		req, err := http.NewRequest(http.MethodPost, "url", nil)
		require.NoError(t, err)

		resp, err := roundTripper.RoundTrip(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Empty response", func(t *testing.T) {
		pluginAPI := &plugintest.API{}
		pluginAPI.On("PluginHTTP", mock.AnythingOfType("*http.Request")).Return(nil)
		defer pluginAPI.AssertExpectations(t)

		roundTripper := pluginAPIRoundTripper{api: pluginAPI}
		req, err := http.NewRequest(http.MethodPost, "url", nil)
		require.NoError(t, err)

		resp, err := roundTripper.RoundTrip(req)
		require.Nil(t, resp)
		require.Error(t, err)
	})
}

func TestGetConfiguration(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		config := &plugin.Configuration{
			EnterpriseBaseURL: "http://example.org",
			GitHubOrg:         "someOrg",
		}

		b := new(bytes.Buffer)
		err := json.NewEncoder(b).Encode(config)
		require.NoError(t, err)

		pluginAPI := &plugintest.API{}
		pluginAPI.On("PluginHTTP", mock.AnythingOfType("*http.Request")).Return(&http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(b)})
		defer pluginAPI.AssertExpectations(t)

		client := NewPluginClient(pluginAPI)

		rConfig, err := client.GetConfiguration()
		assert.NoError(t, err)
		assert.Equal(t, config, rConfig)
	})

	t.Run("Error", func(t *testing.T) {
		pluginAPI := &plugintest.API{}
		pluginAPI.On("PluginHTTP", mock.AnythingOfType("*http.Request")).Return(nil)
		defer pluginAPI.AssertExpectations(t)

		client := NewPluginClient(pluginAPI)

		config, err := client.GetConfiguration()
		assert.Error(t, err)
		assert.Nil(t, config)
	})
}

func TestGetToken(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		token := &oauth2.Token{
			AccessToken: "abcdef",
		}

		b := new(bytes.Buffer)
		err := json.NewEncoder(b).Encode(token)
		require.NoError(t, err)

		pluginAPI := &plugintest.API{}
		pluginAPI.On("PluginHTTP", mock.AnythingOfType("*http.Request")).Return(&http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(b)})
		defer pluginAPI.AssertExpectations(t)

		client := NewPluginClient(pluginAPI)

		rToken, err := client.GetToken("someUserID")
		assert.NoError(t, err)
		assert.Equal(t, token, rToken)
	})

	t.Run("Error", func(t *testing.T) {
		pluginAPI := &plugintest.API{}
		pluginAPI.On("PluginHTTP", mock.AnythingOfType("*http.Request")).Return(nil)
		defer pluginAPI.AssertExpectations(t)

		client := NewPluginClient(pluginAPI)

		token, err := client.GetToken("someUserID")
		assert.Error(t, err)
		assert.Nil(t, token)
	})
}
