package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

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

	tests := []struct {
		name             string
		httpTest         testutils.HTTPTest
		request          testutils.Request
		expectedResponse testutils.ExpectedResponse
		userID           string
	}{
		{
			name:     "unauthorized test json",
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
		},
		{
			name:     "unauthorized test http",
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPlugin()
			p.setConfiguration(
				&configuration{
					GitHubOrg:               "mockOrg",
					GitHubOAuthClientID:     "mockID",
					GitHubOAuthClientSecret: "mockSecret",
					WebhookSecret:           "",
					EnablePrivateRepo:       false,
					EncryptionKey:           "mockKey",
					EnterpriseBaseURL:       "",
					EnterpriseUploadURL:     "",
					EnableCodePreview:       false,
				})
			p.initializeAPI()
			p.SetAPI(&plugintest.API{})

			req := tt.httpTest.CreateHTTPRequest(tt.request)
			req.Header.Add("Mattermost-User-ID", tt.userID)
			rr := httptest.NewRecorder()
			p.ServeHTTP(nil, rr, req)
			tt.httpTest.CompareHTTPResponse(rr, tt.expectedResponse)
		})
	}
}
