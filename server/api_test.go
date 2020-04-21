package main

import (
	"github.com/mattermost/mattermost-plugin-github/server/testutils"
	"github.com/mattermost/mattermost-server/v5/model"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPlugin_ServeHTTP(t *testing.T) {
	httpTestJson := testutils.HTTPTest{
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
			httpTest: httpTestJson,
			request: testutils.Request{
				Method: "GET",
				URL:    "/api/v1/connected",
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
				Method: "GET",
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
					PluginsDirectory:        "",
					EnableCodePreview:       false,
				},
				&model.Config{
					ServiceSettings:         model.ServiceSettings{},
					TeamSettings:            model.TeamSettings{},
					ClientRequirements:      model.ClientRequirements{},
					SqlSettings:             model.SqlSettings{},
					LogSettings:             model.LogSettings{},
					NotificationLogSettings: model.NotificationLogSettings{},
					PasswordSettings:        model.PasswordSettings{},
					FileSettings:            model.FileSettings{},
					EmailSettings:           model.EmailSettings{},
					RateLimitSettings:       model.RateLimitSettings{},
					PrivacySettings:         model.PrivacySettings{},
					SupportSettings:         model.SupportSettings{},
					AnnouncementSettings:    model.AnnouncementSettings{},
					ThemeSettings:           model.ThemeSettings{},
					GitLabSettings:          model.SSOSettings{},
					GoogleSettings:          model.SSOSettings{},
					Office365Settings:       model.SSOSettings{},
					LdapSettings:            model.LdapSettings{},
					ComplianceSettings:      model.ComplianceSettings{},
					LocalizationSettings:    model.LocalizationSettings{},
					SamlSettings:            model.SamlSettings{},
					NativeAppSettings:       model.NativeAppSettings{},
					ClusterSettings:         model.ClusterSettings{},
					MetricsSettings:         model.MetricsSettings{},
					ExperimentalSettings:    model.ExperimentalSettings{},
					AnalyticsSettings:       model.AnalyticsSettings{},
					ElasticsearchSettings:   model.ElasticsearchSettings{},
					DataRetentionSettings:   model.DataRetentionSettings{},
					MessageExportSettings:   model.MessageExportSettings{},
					JobSettings:             model.JobSettings{},
					PluginSettings:          model.PluginSettings{},
					DisplaySettings:         model.DisplaySettings{},
					GuestAccountsSettings:   model.GuestAccountsSettings{},
					ImageProxySettings:      model.ImageProxySettings{},
				})
			p.initialiseAPI()

			req := tt.httpTest.CreateHTTPRequest(tt.request)
			req.Header.Add("Mattermost-User-ID", tt.userID)
			rr := httptest.NewRecorder()
			p.ServeHTTP(nil, rr, req)
			tt.httpTest.CompareHTTPResponse(rr, tt.expectedResponse)
		})
	}
}
