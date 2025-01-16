package plugin

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/mattermost/mattermost-plugin-github/server/testutils"
)

type panicHandler struct {
}

func (ph panicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	panic("bad handler")
}

func TestWithRecovery(t *testing.T) {
	defer func() {
		if x := recover(); x != nil {
			require.Fail(t, "got panic")
		}
	}()

	p := NewPlugin()
	api := &plugintest.API{}
	api.On("LogWarn",
		"Recovered from a panic",
		"url", "http://random",
		"error", "bad handler",
		"stack", mock.Anything)
	p.SetAPI(api)
	p.client = pluginapi.NewClient(p.API, p.Driver)

	ph := panicHandler{}
	handler := p.withRecovery(ph)

	req := httptest.NewRequest(http.MethodGet, "http://random", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.Body != nil {
		defer resp.Body.Close()
		_, err := io.Copy(io.Discard, resp.Body)
		require.NoError(t, err)
	}
}

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
				URL:    "/api/v1/lhs-content",
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

func TestCheckPluginRequest(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		setup      func()
		assertions func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name:    "Missing Mattermost-Plugin-ID header",
			headers: map[string]string{},
			setup:   func() {},
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, rec.Result().StatusCode)
				body, _ := io.ReadAll(rec.Body)
				assert.Equal(t, "Not authorized\n", string(body))
			},
		},
		{
			name: "Valid Mattermost-Plugin-ID header",
			headers: map[string]string{
				"Mattermost-Plugin-ID": "validPluginID",
			},
			setup: func() {},
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
				body, _ := io.ReadAll(rec.Body)
				assert.Equal(t, "Success\n", string(body))
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("Success\n"))
				assert.NoError(t, err)
			})

			handler := checkPluginRequest(nextHandler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}
			rec := httptest.NewRecorder()

			handler(rec, req)

			tc.assertions(t, rec)
		})
	}
}

func TestGetToken(t *testing.T) {
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)

	tests := []struct {
		name       string
		userID     string
		setup      func()
		assertions func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name:   "Missing userID",
			userID: "",
			setup: func() {
				mockAPI.On("LogError", "UserID not found.")
			},
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, rec.Result().StatusCode)
				body, _ := io.ReadAll(rec.Body)
				assert.Contains(t, string(body), "please provide a userID")
			},
		},
		{
			name:   "User info not found in store",
			userID: "mockUserID",
			setup: func() {
				mockAPI.On("LogError", "error occurred while getting the github user info", "UserID", MockUserID, "error", &APIErrorResponse{Message: "Unable to get user info.", StatusCode: http.StatusInternalServerError})
				mockKvStore.EXPECT().Get("mockUserID"+githubTokenKey, gomock.Any()).Return(errors.New("not found")).Times(1)
			},
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, rec.Result().StatusCode)
				body, _ := io.ReadAll(rec.Body)
				assert.Contains(t, string(body), "Unable to get user info.")
			},
		},
		{
			name:   "Successful token retrieval",
			userID: "mockUserID",
			setup: func() {
				encryptedToken, err := encrypt([]byte("dummyEncryptKey1"), MockAccessToken)
				assert.NoError(t, err)
				mockKvStore.EXPECT().Get("mockUserID"+githubTokenKey, gomock.Any()).DoAndReturn(func(key string, value **GitHubUserInfo) error {
					*value = &GitHubUserInfo{
						Token: &oauth2.Token{
							AccessToken: encryptedToken,
						},
					}
					return nil
				}).Times(1)
				p.setConfiguration(&Configuration{EncryptionKey: "dummyEncryptKey1"})
			},
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
				body, _ := io.ReadAll(rec.Body)
				assert.Contains(t, string(body), MockAccessToken)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			req := httptest.NewRequest(http.MethodGet, "/get/token?userID="+tc.userID, nil)
			rec := httptest.NewRecorder()

			p.getToken(rec, req)

			tc.assertions(t, rec)
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

	authorizedHeader := http.Header{}
	authorizedHeader.Add("Mattermost-Plugin-ID", "somePluginId")

	config := &Configuration{
		GitHubOrg:               "mockOrg",
		GitHubOAuthClientID:     "mockID",
		GitHubOAuthClientSecret: "mockSecret",
		EncryptionKey:           "mockKey",
	}

	for name, test := range map[string]struct {
		httpTest         testutils.HTTPTest
		request          testutils.Request
		expectedResponse testutils.ExpectedResponse
	}{
		"not authorized": {
			httpTest: httpTestString,
			request: testutils.Request{
				Method: http.MethodGet,
				URL:    "/api/v1/config",
				Body:   nil,
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode:   http.StatusUnauthorized,
				ResponseType: testutils.ContentTypePlain,
				Body:         "Not authorized\n",
			},
		},
		"authorized": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodGet,
				URL:    "/api/v1/config",
				Header: authorizedHeader,
				Body:   nil,
			},
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

			p.ServeHTTP(&plugin.Context{}, rr, req)

			test.httpTest.CompareHTTPResponse(rr, test.expectedResponse)
		})
	}
}

func TestGetGitHubUser(t *testing.T) {
	mockKvStore, mockAPI, mockLogger, mockLoggerWith, mockContext := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)

	tests := []struct {
		name               string
		requestBody        string
		setup              func()
		expectedStatusCode int
		assertions         func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name:        "Invalid JSON Request Body",
			requestBody: "invalid-json",
			setup: func() {
				mockLogger.EXPECT().WithError(gomock.Any()).Return(mockLoggerWith).Times(1)
				mockLoggerWith.EXPECT().Warnf("Error decoding GitHubUserRequest from JSON body").Times(1)
			},
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, rec.Result().StatusCode)

				var response APIErrorResponse
				_ = json.NewDecoder(rec.Body).Decode(&response)
				assert.Contains(t, response.Message, "Please provide a JSON object.")
			},
		},
		{
			name:        "Blank user_id field",
			requestBody: `{"user_id": ""}`,
			setup:       func() {},
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, rec.Result().StatusCode)
				var response APIErrorResponse
				_ = json.NewDecoder(rec.Body).Decode(&response)
				assert.Contains(t, response.Message, "non-blank user_id field")
			},
		},
		{
			name:        "Error to getting user info",
			requestBody: `{"user_id": "mockUserID"}`,
			setup: func() {
				mockKvStore.EXPECT().Get(gomock.Any(), gomock.Any()).Return(errors.New("Error getting user details")).Times(1)
			},
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, rec.Result().StatusCode)
				var response APIErrorResponse
				_ = json.NewDecoder(rec.Body).Decode(&response)
				assert.Contains(t, response.Message, "Unable to get user info")
			},
		},
		{
			name:        "User is not connected to a GitHub account.",
			requestBody: `{"user_id": "mockUserID"}`,
			setup: func() {
				mockKvStore.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Result().StatusCode)

				var response APIErrorResponse
				_ = json.NewDecoder(rec.Body).Decode(&response)
				assert.Contains(t, response.Message, "User is not connected to a GitHub account.")
			},
		},
		{
			name:        "Successfully get github user",
			requestBody: `{"user_id": "mockUserID"}`,
			setup: func() {
				dummyUserInfo, err := GetMockGHUserInfo(p)
				assert.NoError(t, err)
				mockKvStore.EXPECT().Get("mockUserID"+githubTokenKey, gomock.Any()).DoAndReturn(func(key string, value **GitHubUserInfo) error {
					*value = dummyUserInfo
					return nil
				}).Times(1)
			},
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
				var response GitHubUserResponse
				err := json.NewDecoder(rec.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Equal(t, MockUsername, response.Username)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			req := httptest.NewRequest(http.MethodPost, "/github/user", strings.NewReader(tc.requestBody))
			rec := httptest.NewRecorder()

			p.getGitHubUser(mockContext, rec, req)

			tc.assertions(t, rec)
		})
	}
}

func TestParseRepo(t *testing.T) {
	tests := []struct {
		name       string
		repoParam  string
		setup      func()
		assertions func(t *testing.T, owner, repo string, err error)
	}{
		{
			name:      "Empty repository parameter",
			repoParam: "",
			setup:     func() {},
			assertions: func(t *testing.T, owner, repo string, err error) {
				assert.Equal(t, "", owner)
				assert.Equal(t, "", repo)
				assert.EqualError(t, err, "repository cannot be blank")
			},
		},
		{
			name:      "Invalid repository format",
			repoParam: "owner",
			setup:     func() {},
			assertions: func(t *testing.T, owner, repo string, err error) {
				assert.Equal(t, "", owner)
				assert.Equal(t, "", repo)
				assert.EqualError(t, err, "invalid repository")
			},
		},
		{
			name:      "Valid repository format",
			repoParam: "owner/repo",
			setup:     func() {},
			assertions: func(t *testing.T, owner, repo string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "owner", owner)
				assert.Equal(t, "repo", repo)
			},
		},
		{
			name:      "Extra slashes in repository parameter",
			repoParam: "owner/repo/",
			setup:     func() {},
			assertions: func(t *testing.T, owner, repo string, err error) {
				assert.Equal(t, "", owner)
				assert.Equal(t, "", repo)
				assert.EqualError(t, err, "invalid repository")
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			owner, repo, err := parseRepo(tc.repoParam)

			tc.assertions(t, owner, repo, err)
		})
	}
}

func TestUpdateSettings(t *testing.T) {
	mockKvStore, mockAPI, mockLogger, mockLoggerWith, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)
	mockGHContext, err := GetMockUserContext(p, mockLogger)
	assert.NoError(t, err)

	tests := []struct {
		name               string
		requestBody        string
		setup              func()
		expectedStatusCode int
		assertions         func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name:        "Invalid JSON Request Body",
			requestBody: "invalid-json",
			setup: func() {
				mockLogger.EXPECT().WithError(gomock.Any()).Return(mockLoggerWith).Times(1)
				mockLoggerWith.EXPECT().Warnf("Error decoding settings from JSON body").Times(1)
			},
			expectedStatusCode: http.StatusBadRequest,
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, rec.Result().StatusCode)
			},
		},
		{
			name:        "Error Storing User Info",
			requestBody: `{"access_token": "mockAccessToken"}`,
			setup: func() {
				p.setConfiguration(&Configuration{
					EncryptionKey: "dummyEncryptKey1",
				})
				mockKvStore.EXPECT().Set(gomock.Any(), gomock.Any()).Return(false, errors.New("store error")).Times(1)
				mockLogger.EXPECT().WithError(gomock.Any()).Return(mockLoggerWith).Times(1)
				mockLoggerWith.EXPECT().Warnf("Failed to store GitHub user info").Times(1)
			},
			expectedStatusCode: http.StatusInternalServerError,
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, rec.Result().StatusCode)
			},
		},
		{
			name:        "Successful Update",
			requestBody: `{"access_token": "mockAccessToken"}`,
			setup: func() {
				mockKvStore.EXPECT().Set(gomock.Any(), gomock.Any()).Return(true, nil).Times(1)
			},
			expectedStatusCode: http.StatusOK,
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
				var settings UserSettings
				err := json.NewDecoder(rec.Body).Decode(&settings)
				assert.NoError(t, err)
				assert.Equal(t, mockGHContext.GHInfo.Settings, &settings)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			req := httptest.NewRequest(http.MethodPost, "/update/settings", strings.NewReader(tc.requestBody))
			rec := httptest.NewRecorder()

			p.updateSettings(mockGHContext, rec, req)

			tc.assertions(t, rec)
		})
	}
}
