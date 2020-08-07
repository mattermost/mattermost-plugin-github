package plugin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"

	"github.com/mattermost/mattermost-plugin-github/server/testutils"
)

func TestPlugin_ServeHTTP(t *testing.T) {
	httpTestJSON := testutils.HTTPTest{
		T:       t,
		Encoder: testutils.EncodeJSON,
	}

	httpTestString := testutils.HTTPTest{
		T:       t,
		Encoder: testutils.EncodeString,
	}

	for name, test := range map[string]struct {
		httpTest         testutils.HTTPTest
		request          testutils.Request
		expectedResponse testutils.ExpectedResponse
		userID           string
	}{
		"unauthorized test json": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    "/api/v1/todo",
				Body:   nil,
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode:   http.StatusUnauthorized,
				ResponseType: testutils.ContentTypeJSON,
				Body:         APIErrorResponse{ID: "", Message: "Not authorized.", StatusCode: http.StatusUnauthorized},
			},
			userID: "",
		}, "unauthorized test http": {
			httpTest: httpTestString,
			request: testutils.Request{
				Method: http.MethodGet,
				URL:    "/api/v1/reviews",
				Body:   nil,
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode:   http.StatusUnauthorized,
				ResponseType: testutils.ContentTypePlain,
				Body:         "Not authorized\n",
			},
			userID: "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := NewPlugin()
			p.setConfiguration(
				&Configuration{
					GitHubOrg:               "mockOrg",
					GitHubOAuthClientID:     "mockID",
					GitHubOAuthClientSecret: "mockSecret",
					WebhookSecret:           "",
					EnablePrivateRepo:       false,
					EncryptionKey:           "mockKey",
					EnterpriseBaseURL:       "",
					EnterpriseUploadURL:     "",
					EnableCodePreview:       "disable",
				})
			p.initializeAPI()
			p.SetAPI(&plugintest.API{})

			req := test.httpTest.CreateHTTPRequest(test.request)
			req.Header.Add("Mattermost-User-ID", test.userID)
			rr := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, rr, req)
			test.httpTest.CompareHTTPResponse(rr, test.expectedResponse)
		})
	}
}

func TestGetToken(t *testing.T) {
	httpTestString := testutils.HTTPTest{
		T:       t,
		Encoder: testutils.EncodeString,
	}

	for name, test := range map[string]struct {
		httpTest         testutils.HTTPTest
		request          testutils.Request
		context          *plugin.Context
		expectedResponse testutils.ExpectedResponse
	}{
		"not authorized": {
			httpTest: httpTestString,
			request: testutils.Request{
				Method: http.MethodGet,
				URL:    "/api/v1/token",
				Body:   nil,
			},
			context: &plugin.Context{},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode:   http.StatusUnauthorized,
				ResponseType: testutils.ContentTypePlain,
				Body:         "Not authorized\n",
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := NewPlugin()
			p.setConfiguration(
				&Configuration{
					GitHubOrg:               "mockOrg",
					GitHubOAuthClientID:     "mockID",
					GitHubOAuthClientSecret: "mockSecret",
					EncryptionKey:           "mockKey",
				})
			p.initializeAPI()

			p.SetAPI(&plugintest.API{})

			req := test.httpTest.CreateHTTPRequest(test.request)
			rr := httptest.NewRecorder()

			p.ServeHTTP(test.context, rr, req)

			test.httpTest.CompareHTTPResponse(rr, test.expectedResponse)
		})
	}
}
func TestGetConfig(t *testing.T) {
	httpTestJSON := testutils.HTTPTest{
		T:       t,
		Encoder: testutils.EncodeJSON,
	}

	httpTestString := testutils.HTTPTest{
		T:       t,
		Encoder: testutils.EncodeString,
	}

	config := &Configuration{
		GitHubOrg:               "mockOrg",
		GitHubOAuthClientID:     "mockID",
		GitHubOAuthClientSecret: "mockSecret",
		EncryptionKey:           "mockKey",
	}

	for name, test := range map[string]struct {
		httpTest         testutils.HTTPTest
		request          testutils.Request
		context          *plugin.Context
		expectedResponse testutils.ExpectedResponse
	}{
		"not authorized": {
			httpTest: httpTestString,
			request: testutils.Request{
				Method: http.MethodGet,
				URL:    "/api/v1/config",
				Body:   nil,
			},
			context: &plugin.Context{},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode:   http.StatusUnauthorized,
				ResponseType: testutils.ContentTypePlain,
				Body:         "Not authorized\n",
			},
		}, "authorized": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodGet,
				URL:    "/api/v1/config",
				Body:   nil,
			},
			context: &plugin.Context{SourcePluginId: "somePluginId"},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode:   http.StatusOK,
				ResponseType: testutils.ContentTypeJSON,
				Body:         config,
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := NewPlugin()
			p.setConfiguration(config)
			p.initializeAPI()

			p.SetAPI(&plugintest.API{})

			req := test.httpTest.CreateHTTPRequest(test.request)
			rr := httptest.NewRecorder()

			p.ServeHTTP(test.context, rr, req)

			test.httpTest.CompareHTTPResponse(rr, test.expectedResponse)
		})
	}
}
