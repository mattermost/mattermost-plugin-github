// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-github/v54/github"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/mattermost/mattermost-plugin-github/server/mocks"
)

const (
	webhookSecret      = "whsecret"
	orgMember          = "org-member"
	orgCollaborator    = "org-collaborator"
	gitHubOrginization = "test-org"
)

func TestVerifyWebhookSignature(t *testing.T) {
	tests := []struct {
		name       string
		secret     []byte
		signature  string
		body       []byte
		assertions func(t *testing.T, valid bool, err error)
	}{
		{
			name:   "Valid signature",
			secret: []byte("test-secret"),
			signature: func() string {
				secret := []byte("test-secret")
				body := []byte("test-body")
				return generateSignature(secret, body)
			}(),
			body: []byte("test-body"),
			assertions: func(t *testing.T, valid bool, err error) {
				assert.NoError(t, err)
				assert.True(t, valid)
			},
		},
		{
			name:      "Invalid signature prefix",
			secret:    []byte("test-secret"),
			signature: "invalid-prefix=1234567890abcdef",
			body:      []byte("test-body"),
			assertions: func(t *testing.T, valid bool, err error) {
				assert.NoError(t, err)
				assert.False(t, valid)
			},
		},
		{
			name:      "Invalid signature length",
			secret:    []byte("test-secret"),
			signature: "sha1=short",
			body:      []byte("test-body"),
			assertions: func(t *testing.T, valid bool, err error) {
				assert.NoError(t, err)
				assert.False(t, valid)
			},
		},
		{
			name:      "Hex decode error",
			secret:    []byte("test-secret"),
			signature: "sha1=gggggggggggggggggggggggggggggggggggggggg",
			body:      []byte("test-body"),
			assertions: func(t *testing.T, valid bool, err error) {
				assert.Error(t, err)
				assert.False(t, valid)
			},
		},
		{
			name:      "HMAC mismatch",
			secret:    []byte("test-secret"),
			signature: "sha1=38cb0302e94c235fb349ac026084db66bc64a979",
			body:      []byte("different-body"),
			assertions: func(t *testing.T, valid bool, err error) {
				assert.NoError(t, err)
				assert.False(t, valid)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := verifyWebhookSignature(tt.secret, tt.signature, tt.body)

			tt.assertions(t, valid, err)
		})
	}
}

func TestGetEventWithRenderConfig(t *testing.T) {
	tests := []struct {
		name       string
		event      interface{}
		sub        *Subscription
		assertions func(t *testing.T, result *EventWithRenderConfig)
	}{
		{
			name:  "No Subscription",
			event: "test-event",
			sub:   nil,
			assertions: func(t *testing.T, result *EventWithRenderConfig) {
				assert.Equal(t, "test-event", result.Event)
				assert.Empty(t, result.Config.Style)
			},
		},
		{
			name:  "Subscription with RenderStyle",
			event: "test-event",
			sub: &Subscription{
				ChannelID:  "channel-1",
				CreatorID:  "creator-1",
				Repository: "repo-1",
			},
			assertions: func(t *testing.T, result *EventWithRenderConfig) {
				assert.Equal(t, "test-event", result.Event)
				assert.Empty(t, result.Config.Style)
			},
		},
		{
			name:  "Subscription with Custom RenderStyle",
			event: "test-event",
			sub: &Subscription{
				ChannelID:  "channel-1",
				CreatorID:  "creator-1",
				Flags:      SubscriptionFlags{RenderStyle: "custom-style"},
				Repository: "repo-1",
			},
			assertions: func(t *testing.T, result *EventWithRenderConfig) {
				assert.Equal(t, "test-event", result.Event)
				assert.Equal(t, "custom-style", result.Config.Style)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetEventWithRenderConfig(tt.event, tt.sub)

			tt.assertions(t, result)
		})
	}
}

func TestNewWebhookBroker(t *testing.T) {
	called := false
	mockSendGitHubPingEvent := func(event *github.PingEvent) {
		called = true
	}

	broker := NewWebhookBroker(mockSendGitHubPingEvent)

	mockSendGitHubPingEvent(nil)

	assert.NotNil(t, broker)
	assert.True(t, called, "sendGitHubPingEvent should have been called")
}

func TestSubscribePings(t *testing.T) {
	broker := &WebhookBroker{}

	ch := broker.SubscribePings()
	assert.NotNil(t, ch, "Channel should not be nil")
	assert.Len(t, broker.pingSubs, 1, "pingSubs should contain one channel")

	testCh := make(chan *github.PingEvent, 1)
	go func() {
		event := &github.PingEvent{}
		testCh <- event
	}()

	receivedEvent := <-testCh
	assert.NotNil(t, receivedEvent, "Received event should not be nil")
}

func TestUnsubscribePings(t *testing.T) {
	broker := &WebhookBroker{}
	ch := broker.SubscribePings()
	assert.NotNil(t, ch, "Channel should not be nil")
	assert.Len(t, broker.pingSubs, 1, "pingSubs should contain one channel")

	broker.UnsubscribePings(ch)

	broker.UnsubscribePings(ch)
	assert.Len(t, broker.pingSubs, 0, "pingSubs should be empty after unsubscribe")
	assert.Len(t, broker.pingSubs, 0, "pingSubs should still be empty after second unsubscribe")
}

func TestPublishPing(t *testing.T) {
	broker := &WebhookBroker{pingSubs: []chan *github.PingEvent{}}
	event := &github.PingEvent{}
	mockSendGitHubPingEvent := func(event *github.PingEvent) {}
	broker.sendGitHubPingEvent = mockSendGitHubPingEvent
	ch := broker.SubscribePings()

	go func() {
		broker.publishPing(event, false)
	}()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		receivedEvent := <-ch
		assert.NotNil(t, receivedEvent, "Received event should not be nil")
		assert.Equal(t, event, receivedEvent, "Received event should match the published event")
	}()

	wg.Wait()

	broker.closed = true
	broker.publishPing(event, false)
}

func TestClose(t *testing.T) {
	broker := &WebhookBroker{pingSubs: []chan *github.PingEvent{}}
	ch := make(chan *github.PingEvent, 1)
	broker.pingSubs = append(broker.pingSubs, ch)

	broker.Close()

	assert.True(t, broker.closed, "Broker should be marked as closed")
	select {
	case _, open := <-ch:
		assert.False(t, open, "Channel should be closed")
	default:
		t.Error("Channel should be closed")
	}
}

func TestHandleWebhookBadRequestBody(t *testing.T) {
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)

	tests := []struct {
		name            string
		signature       func([]byte) string
		body            []byte
		githubEventType string
		setup           func()
		assertions      func(t *testing.T, resp *httptest.ResponseRecorder)
	}{
		{
			name:            "failed signature verification (invalid signature)",
			body:            []byte("valid body"),
			signature:       func(body []byte) string { return "" },
			githubEventType: "",
			setup: func() {
				p.setConfiguration(&Configuration{
					WebhookSecret: MockWebhookSecret,
				})
			},
			assertions: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, resp.Code)
			},
		},
		{
			name: "Request body is not webhook content type",
			body: []byte("valid body"),
			signature: func(body []byte) string {
				return generateSignature([]byte(MockWebhookSecret), body)
			},
			githubEventType: "",
			setup: func() {
				p.setConfiguration(&Configuration{
					WebhookSecret: MockWebhookSecret,
				})
				mockAPI.On("LogDebug", "GitHub webhook content type should be set to \"application/json\"", "error", "unknown X-Github-Event in message: ").Times(1)
			},
			assertions: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, resp.Code)
			},
		},
		{
			name: "Successful handle ping event",
			body: func() []byte {
				event := GetMockPingEvent()
				body, err := json.Marshal(event)
				assert.NoError(t, err)
				return body
			}(),
			signature: func(body []byte) string {
				return generateSignature([]byte(MockWebhookSecret), body)
			},
			githubEventType: "ping",
			setup: func() {
				p.webhookBroker = NewWebhookBroker(p.sendGitHubPingEvent)
				p.setConfiguration(&Configuration{
					WebhookSecret:             MockWebhookSecret,
					EnableWebhookEventLogging: true,
				})
				mockAPI.On("LogDebug", "Webhook Event Log", "event", mock.AnythingOfType("string")).Times(1)
				mockAPI.On("PublishPluginClusterEvent", mock.AnythingOfType("model.PluginClusterEvent"), mock.AnythingOfType("model.PluginClusterEventSendOptions")).Return(nil).Times(1)
			},
			assertions: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
		{
			name: "Successful handle pull request event",
			body: func() []byte {
				event := GetMockPullRequestEvent(actionOpened, MockRepo, "", false, MockSender, MockUserLogin, "")
				body, err := json.Marshal(event)
				assert.NoError(t, err)
				return body
			}(),
			signature: func(body []byte) string {
				return generateSignature([]byte(MockWebhookSecret), body)
			},
			githubEventType: "pull_request",
			setup: func() {
				p.webhookBroker = NewWebhookBroker(p.sendGitHubPingEvent)
				p.setConfiguration(&Configuration{
					WebhookSecret:             MockWebhookSecret,
					EnableWebhookEventLogging: true,
				})
				mockAPI.On("LogDebug", "Webhook Event Log", "event", mock.AnythingOfType("string")).Times(1)
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("LogDebug", "Unhandled event action", "action", "opened").Times(1)
			},
			assertions: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
		{
			name: "Successfully handle issue event",
			body: func() []byte {
				event := GetMockIssueEvent("", "", "", "", "")
				body, err := json.Marshal(event)
				assert.NoError(t, err)
				return body
			}(),
			signature: func(body []byte) string {
				return generateSignature([]byte(MockWebhookSecret), body)
			},
			githubEventType: "issues",
			setup: func() {
				p.webhookBroker = NewWebhookBroker(p.sendGitHubPingEvent)
				p.setConfiguration(&Configuration{
					WebhookSecret:             MockWebhookSecret,
					EnableWebhookEventLogging: true,
				})
				mockAPI.On("LogDebug", "Webhook Event Log", "event", mock.AnythingOfType("string")).Times(1)
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).Return(nil).Times(1)
			},
			assertions: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
		{
			name: "Successfully handle issue comment event",
			body: func() []byte {
				event := GetMockIssueCommentEvent("", "", "")
				body, err := json.Marshal(event)
				assert.NoError(t, err)
				return body
			}(),
			signature: func(body []byte) string {
				return generateSignature([]byte(MockWebhookSecret), body)
			},
			githubEventType: "issue_comment",
			setup: func() {
				p.webhookBroker = NewWebhookBroker(p.sendGitHubPingEvent)
				p.setConfiguration(&Configuration{
					WebhookSecret:             MockWebhookSecret,
					EnableWebhookEventLogging: true,
				})
				mockAPI.On("LogDebug", "Webhook Event Log", "event", mock.AnythingOfType("string")).Times(1)
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).Return(nil).Times(1)
				mockKvStore.EXPECT().Get("issueAuthor_githubusername", gomock.Any()).Return(nil).Times(1)
			},
			assertions: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
		{
			name: "Successfully handle pull request review event",
			body: func() []byte {
				event := GetMockPullRequestReviewEvent("", "", "", true, "", "")
				body, err := json.Marshal(event)
				assert.NoError(t, err)
				return body
			}(),
			signature: func(body []byte) string {
				return generateSignature([]byte(MockWebhookSecret), body)
			},
			githubEventType: "pull_request_review",
			setup: func() {
				p.webhookBroker = NewWebhookBroker(p.sendGitHubPingEvent)
				p.setConfiguration(&Configuration{
					WebhookSecret:             MockWebhookSecret,
					EnableWebhookEventLogging: true,
				})
				mockAPI.On("LogDebug", "Webhook Event Log", "event", mock.AnythingOfType("string")).Times(1)
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).Return(nil).Times(1)
			},
			assertions: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
		{
			name: "Successfully handle pull request review comment event",
			body: func() []byte {
				event := GetMockPullRequestReviewCommentEvent()
				body, err := json.Marshal(event)
				assert.NoError(t, err)
				return body
			}(),
			signature: func(body []byte) string {
				return generateSignature([]byte(MockWebhookSecret), body)
			},
			githubEventType: "pull_request_review_comment",
			setup: func() {
				p.webhookBroker = NewWebhookBroker(p.sendGitHubPingEvent)
				p.setConfiguration(&Configuration{
					WebhookSecret:             MockWebhookSecret,
					EnableWebhookEventLogging: true,
				})
				mockAPI.On("LogDebug", "Webhook Event Log", "event", mock.AnythingOfType("string")).Times(1)
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).Return(nil).Times(1)
			},
			assertions: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
		{
			name: "Successfully handle push event",
			body: func() []byte {
				event := GetMockPushEvent()
				body, err := json.Marshal(event)
				assert.NoError(t, err)
				return body
			}(),
			signature: func(body []byte) string {
				return generateSignature([]byte(MockWebhookSecret), body)
			},
			githubEventType: "push",
			setup: func() {
				p.webhookBroker = NewWebhookBroker(p.sendGitHubPingEvent)
				p.setConfiguration(&Configuration{
					WebhookSecret:             MockWebhookSecret,
					EnableWebhookEventLogging: true,
				})
				mockAPI.On("LogDebug", "Webhook Event Log", "event", mock.AnythingOfType("string")).Times(1)
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).Return(nil).Times(1)
			},
			assertions: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
		{
			name: "Successfully handle create event",
			body: func() []byte {
				event := GetMockCreateEvent()
				body, err := json.Marshal(event)
				assert.NoError(t, err)
				return body
			}(),
			signature: func(body []byte) string {
				return generateSignature([]byte(MockWebhookSecret), body)
			},
			githubEventType: "create",
			setup: func() {
				p.webhookBroker = NewWebhookBroker(p.sendGitHubPingEvent)
				p.setConfiguration(&Configuration{
					WebhookSecret:             MockWebhookSecret,
					EnableWebhookEventLogging: true,
				})
				mockAPI.On("LogDebug", "Webhook Event Log", "event", mock.AnythingOfType("string")).Times(1)
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).Return(nil).Times(1)
			},
			assertions: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
		{
			name: "Successfully handle delete event",
			body: func() []byte {
				event := GetMockDeleteEvent()
				body, err := json.Marshal(event)
				assert.NoError(t, err)
				return body
			}(),
			signature: func(body []byte) string {
				return generateSignature([]byte(MockWebhookSecret), body)
			},
			githubEventType: "delete",
			setup: func() {
				p.webhookBroker = NewWebhookBroker(p.sendGitHubPingEvent)
				p.setConfiguration(&Configuration{
					WebhookSecret:             MockWebhookSecret,
					EnableWebhookEventLogging: true,
				})
				mockAPI.On("LogDebug", "Webhook Event Log", "event", mock.AnythingOfType("string")).Times(1)
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).Return(nil).Times(1)
			},
			assertions: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
		{
			name: "Successfully handle start event",
			body: func() []byte {
				event := GetMockStarEvent("", "", true, "")
				body, err := json.Marshal(event)
				assert.NoError(t, err)
				return body
			}(),
			signature: func(body []byte) string {
				return generateSignature([]byte(MockWebhookSecret), body)
			},
			githubEventType: "star",
			setup: func() {
				p.webhookBroker = NewWebhookBroker(p.sendGitHubPingEvent)
				p.setConfiguration(&Configuration{
					WebhookSecret:             MockWebhookSecret,
					EnableWebhookEventLogging: true,
				})
				mockAPI.On("LogDebug", "Webhook Event Log", "event", mock.AnythingOfType("string")).Times(1)
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).Return(nil).Times(1)
			},
			assertions: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
		{
			name: "Successfully handle release event",
			body: func() []byte {
				event := GetMockReleaseEvent("", "", "", "")
				body, err := json.Marshal(event)
				assert.NoError(t, err)
				return body
			}(),
			signature: func(body []byte) string {
				return generateSignature([]byte(MockWebhookSecret), body)
			},
			githubEventType: "release",
			setup: func() {
				p.webhookBroker = NewWebhookBroker(p.sendGitHubPingEvent)
				p.setConfiguration(&Configuration{
					WebhookSecret:             MockWebhookSecret,
					EnableWebhookEventLogging: true,
				})
				mockAPI.On("LogDebug", "Webhook Event Log", "event", mock.AnythingOfType("string")).Times(1)
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).Return(nil).Times(1)
			},
			assertions: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
		{
			name: "Successfully handle discussion event",
			body: func() []byte {
				event := GetMockDiscussionEvent("", "", "")
				body, err := json.Marshal(event)
				assert.NoError(t, err)
				return body
			}(),
			signature: func(body []byte) string {
				return generateSignature([]byte(MockWebhookSecret), body)
			},
			githubEventType: "discussion",
			setup: func() {
				p.webhookBroker = NewWebhookBroker(p.sendGitHubPingEvent)
				p.setConfiguration(&Configuration{
					WebhookSecret:             MockWebhookSecret,
					EnableWebhookEventLogging: true,
				})
				mockAPI.On("LogDebug", "Webhook Event Log", "event", mock.AnythingOfType("string")).Times(1)
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).Return(nil).Times(1)
			},
			assertions: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
		{
			name: "Successfully handle discussion comment event",
			body: func() []byte {
				event := GetMockDiscussionCommentEvent("", "", "", "")
				body, err := json.Marshal(event)
				assert.NoError(t, err)
				return body
			}(),
			signature: func(body []byte) string {
				return generateSignature([]byte(MockWebhookSecret), body)
			},
			githubEventType: "discussion_comment",
			setup: func() {
				p.webhookBroker = NewWebhookBroker(p.sendGitHubPingEvent)
				p.setConfiguration(&Configuration{
					WebhookSecret:             MockWebhookSecret,
					EnableWebhookEventLogging: true,
				})
				mockAPI.On("LogDebug", "Webhook Event Log", "event", mock.AnythingOfType("string")).Times(1)
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).Return(nil).Times(1)
			},
			assertions: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
		{
			name: "Successfully handle discussion comment event",
			body: func() []byte {
				event := GetMockDiscussionCommentEvent("", "", "", "")
				body, err := json.Marshal(event)
				assert.NoError(t, err)
				return body
			}(),
			signature: func(body []byte) string {
				return generateSignature([]byte(MockWebhookSecret), body)
			},
			githubEventType: "discussion_comment",
			setup: func() {
				p.webhookBroker = NewWebhookBroker(p.sendGitHubPingEvent)
				p.setConfiguration(&Configuration{
					WebhookSecret:             MockWebhookSecret,
					EnableWebhookEventLogging: true,
				})
				mockAPI.On("LogDebug", "Webhook Event Log", "event", mock.AnythingOfType("string")).Times(1)
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).Return(nil).Times(1)
			},
			assertions: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			mockAPI.On("LogInfo", "Webhook event received")

			req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Hub-Signature", tc.signature(tc.body))
			req.Header.Set("X-GitHub-Event", tc.githubEventType)
			resp := httptest.NewRecorder()

			p.handleWebhook(resp, req)

			tc.assertions(t, resp)
		})
	}
}

func TestPostPullRequestEvent(t *testing.T) {
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)

	tests := []struct {
		name  string
		event *github.PullRequestEvent
		setup func()
	}{
		{
			name:  "No subscription for channel",
			event: GetMockPullRequestEvent(actionCreated, MockRepo, MockValidLabel, false, MockSender, MockUserID, MockUsername),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "Unsupported action",
			event: GetMockPullRequestEvent(actionCreated, MockRepo, MockValidLabel, false, MockSender, MockUserID, MockUsername),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockorg/mockrepo", "issues,label:\"validLabel\"")
					}
					return nil
				}).Times(1)
			},
		},
		{
			name:  "Valid subscription does not exist",
			event: GetMockPullRequestEvent(actionOpened, MockRepo, MockValidLabel, false, MockSender, MockUserID, MockUsername),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockorg/mockrepo", "issues,label:\"validLabel\"")
					}
					return nil
				}).Times(1)
			},
		},
		{
			name:  "PullsMerged subscription exist but PR action is not closed",
			event: GetMockPullRequestEvent(actionOpened, MockRepo, MockValidLabel, false, MockSender, MockUserID, MockUsername),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockorg/mockrepo", "pulls_merged,label:\"validLabel\"")
					}
					return nil
				}).Times(1)
			},
		},
		{
			name:  "PullsCreated subscription exist but PR action is not opened",
			event: GetMockPullRequestEvent(actionClosed, MockRepo, MockValidLabel, false, MockSender, MockUserID, MockUsername),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockorg/mockrepo", "pulls_created,label:\"validLabel\"")
					}
					return nil
				}).Times(1)
			},
		},
		{
			name:  "no valid label exists",
			event: GetMockPullRequestEvent(actionOpened, MockRepo, MockValidLabel, false, MockSender, MockUserID, MockUsername),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockorg/mockrepo", "pulls_created,label:\"invalidLabel\"")
					}
					return nil
				}).Times(1)
			},
		},
		{
			name:  "Error creating post for action labeled",
			event: GetMockPullRequestEvent(actionLabeled, MockRepo, MockValidLabel, false, MockSender, MockUserID, MockUsername),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = &Subscriptions{
							Repositories: map[string][]*Subscription{
								"mockorg/mockrepo": {
									{
										ChannelID:  MockChannelID,
										CreatorID:  MockCreatorID,
										Features:   Features("pulls,label:\"validLabel\""),
										Repository: MockRepo,
									},
								},
							},
						}
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error webhook post", "post", mock.AnythingOfType("*model.Post"), "error", "error creating post")
			},
		},
		{
			name:  "event label is not equal to subscription label",
			event: GetMockPullRequestEvent(actionLabeled, MockRepo, "invalidLabel", false, MockSender, MockUserID, MockUsername),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockorg/mockrepo", "pulls,label:\"validLabel\"")
					}
					return nil
				}).Times(1)
			},
		},
		{
			name:  "success creating post for action labeled",
			event: GetMockPullRequestEvent(actionLabeled, MockRepo, MockValidLabel, false, MockSender, MockUserID, MockUsername),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = &Subscriptions{
							Repositories: map[string][]*Subscription{
								"mockorg/mockrepo": {
									{
										ChannelID:  MockChannelID,
										CreatorID:  MockCreatorID,
										Features:   Features("pulls,label:\"validLabel\""),
										Repository: MockRepo,
									},
								},
							},
						}
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil).Times(1)
			},
		},
		{
			name:  "Success creating post for pull requeset opened",
			event: GetMockPullRequestEvent(actionOpened, MockRepo, MockValidLabel, false, MockSender, MockUserID, MockUsername),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockorg/mockrepo", "pulls_created,label:\"validLabel\"")
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil).Times(1)
			},
		},
		{
			name:  "Success creating post for pull opened",
			event: GetMockPullRequestEvent(actionReopened, MockRepo, MockValidLabel, false, MockSender, MockUserID, MockUsername),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockorg/mockrepo", "pulls,label:\"validLabel\"")
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil).Times(1)
			},
		},
		{
			name:  "Success creating post for action MarkedReadyForReview",
			event: GetMockPullRequestEvent(actionMarkedReadyForReview, MockRepo, MockValidLabel, false, MockSender, MockUserID, MockUsername),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockorg/mockrepo", "pulls,label:\"validLabel\"")
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil).Times(1)
			},
		},
		{
			name:  "Success creating post for action closed",
			event: GetMockPullRequestEvent(actionClosed, MockRepo, MockValidLabel, false, MockSender, MockUserID, MockUsername),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockorg/mockrepo", "pulls,label:\"validLabel\"")
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			p.postPullRequestEvent(tc.event)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestSanitizeDescription(t *testing.T) {
	tests := []struct {
		name        string
		description string
		expected    string
	}{
		{
			name:        "description with <details>",
			description: "description with <details>MockDetails</details> and the values",
			expected:    "description with  and the values",
		},
		{
			name:        "description without <details>",
			description: "Content without details tag.",
			expected:    "Content without details tag.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
			p := getPluginTest(mockAPI, mockKvStore)

			sanitizedDescription := p.sanitizeDescription(tt.description)

			assert.Equal(t, tt.expected, sanitizedDescription)
		})
	}
}

func TestHandlePRDescriptionMentionNotification(t *testing.T) {
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)

	tests := []struct {
		name  string
		event *github.PullRequestEvent
		setup func()
	}{
		{
			name:  "action other than opened",
			event: GetMockPRDescriptionEvent(MockRepo, MockOrg, MockSender, MockSender, actionClosed, ""),
			setup: func() {},
		},
		{
			name:  "no mentioned users in PR description",
			event: GetMockPRDescriptionEvent(MockRepo, MockOrg, MockSender, MockSender, actionOpened, ""),
			setup: func() {},
		},
		{
			name:  "PR description mentions a user but they are the PR author",
			event: GetMockPRDescriptionEvent(MockRepo, MockOrg, MockSender, MockSender, actionOpened, fmt.Sprintf("@%s", MockSender)),
			setup: func() {
				mockKvStore.EXPECT().Get("prAuthor_githubusername", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "Skip notification for pull request",
			event: GetMockPRDescriptionEvent(MockRepo, MockOrg, "mockSender2", MockSender, actionOpened, fmt.Sprintf("@%s", MockSender)),
			setup: func() {
				mockKvStore.EXPECT().Get("prAuthor_githubusername", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "user id not mapped with github",
			event: GetMockPRDescriptionEvent(MockRepo, MockOrg, MockSender, MockSender, actionOpened, MockProfileUsername),
			setup: func() {
				mockKvStore.EXPECT().Get("username_githubusername", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "Error getting channel",
			event: GetMockPRDescriptionEvent(MockRepo, MockOrg, MockSender, MockSender, actionOpened, MockProfileUsername),
			setup: func() {
				mockKvStore.EXPECT().Get("username_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte(MockUserID)
					}
					return nil
				}).Times(1)
				mockKvStore.EXPECT().Get("mockUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("GetDirectChannel", MockUserID, p.BotUserID).Return(nil, &model.AppError{Message: "error getting direct channel"}).Times(1)
			},
		},
		{
			name:  "PR description mentions a user, post created",
			event: GetMockPRDescriptionEvent(MockRepo, MockOrg, MockSender, MockSender, actionOpened, MockProfileUsername),
			setup: func() {
				mockKvStore.EXPECT().Get("username_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte(MockUserID)
					}
					return nil
				}).Times(1)
				mockKvStore.EXPECT().Get("mockUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("GetDirectChannel", MockUserID, p.BotUserID).Return(&model.Channel{Id: MockChannelID}, nil).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{Id: MockPostID}, nil).Times(1)
				mockAPI.On("LogWarn", "Failed to get github user info", "error", "Must connect user account to GitHub first.")
			},
		},
		{
			name:  "Error creating post",
			event: GetMockPRDescriptionEvent(MockRepo, MockOrg, MockSender, MockSender, actionOpened, MockProfileUsername),
			setup: func() {
				mockKvStore.EXPECT().Get("username_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte(MockUserID)
					}
					return nil
				}).Times(1)
				mockKvStore.EXPECT().Get("mockUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("GetDirectChannel", MockUserID, p.BotUserID).Return(&model.Channel{Id: MockChannelID}, nil).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Failed to get github user info", "error", "Must connect user account to GitHub first.")
				mockAPI.On("LogWarn", "Error webhook post", "post", mock.AnythingOfType("*model.Post"), "error", "error creating post")
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			p.handlePRDescriptionMentionNotification(tc.event)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestPostIssueEvent(t *testing.T) {
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)

	tests := []struct {
		name  string
		event *github.IssuesEvent
		setup func()
	}{
		{
			name:  "no subscribed channels for repository",
			event: GetMockIssueEvent(MockRepo, MockOrg, MockSender, actionOpened, MockLabel),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "issue labeled but recently created, no post sent",
			event: GetMockIssueEventWithTimeDiff(MockRepo, MockOrg, MockSender, actionLabeled, MockLabel, -2*time.Second),
			setup: func() {},
		},
		{
			name:  "issue labeled with matching label",
			event: GetMockIssueEventWithTimeDiff(MockRepo, MockOrg, MockSender, actionLabeled, MockValidLabel, -5*time.Second),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockrepo/mockorg", "issues,label:\"validLabel\"")
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil).Times(1)
			},
		},
		{
			name:  "error creating post",
			event: GetMockIssueEventWithTimeDiff(MockRepo, MockOrg, MockSender, actionLabeled, MockValidLabel, -5*time.Second),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockrepo/mockorg", "issues,label:\"validLabel\"")
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error webhook post", "post", mock.AnythingOfType("*model.Post"), "error", "error creating post")
			},
		},
		{
			name:  "issue creation skipped due to unsupported action",
			event: GetMockIssueEventWithTimeDiff(MockRepo, MockOrg, MockSender, actionClosed, MockLabel, -5*time.Second),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockorg/mockrepo", featureIssueCreation)
					}
					return nil
				}).Times(1)
			},
		},
		{
			name:  "issue skipped due to unmatched label",
			event: GetMockIssueEventWithTimeDiff(MockRepo, MockOrg, MockSender, actionLabeled, "nonMatchingLabel", -5*time.Second),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockorg/mockrepo", "issues,label:\"validLabel\"")
					}
					return nil
				}).Times(1)
			},
		},
		{
			name: "issue skipped due to mismatched event label",
			event: func() *github.IssuesEvent {
				event := GetMockIssueEventWithTimeDiff(MockRepo, MockOrg, MockSender, actionLabeled, "eventLabel", -5*time.Second)
				event.GetIssue().Labels = []*github.Label{{Name: github.String("subscriptionLabel")}}
				return event
			}(),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockorg/mockrepo", "issues,label:\"subscriptionLabel\"")
					}
					return nil
				}).Times(1)
			},
		},
		{
			name:  "success creating post for issue opened",
			event: GetMockIssueEvent(MockRepo, MockOrg, MockSender, actionOpened, MockLabel),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockrepo/mockorg", featureIssueCreation)
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil).Times(1)
			},
		},
		{
			name:  "success creating post for issue closed",
			event: GetMockIssueEvent(MockRepo, MockOrg, MockSender, actionOpened, MockLabel),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockrepo/mockorg", featureIssueCreation)
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil).Times(1)
			},
		},
		{
			name:  "success creating post for issue reopened",
			event: GetMockIssueEvent(MockRepo, MockOrg, MockSender, actionReopened, MockLabel),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockrepo/mockorg", featureIssueCreation)
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil).Times(1)
			},
		},
		{
			name:  "unsupported action",
			event: GetMockIssueEvent(MockRepo, MockOrg, MockSender, actionDeleted, MockLabel),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockorg/mockrepo", featureIssueCreation)
					}
					return nil
				}).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			p.postIssueEvent(tc.event)

			mockAPI.AssertExpectations(t)
			mockAPI.ExpectedCalls = nil
		})
	}
}

func TestPostPushEvent(t *testing.T) {
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)

	tests := []struct {
		name      string
		pushEvent *github.PushEvent
		setup     func()
	}{
		{
			name:      "no subscription found",
			pushEvent: GetMockPushEvent(),
			setup: func() {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:      "no commits found in event",
			pushEvent: GetMockPushEventWithoutCommit(),
			setup: func() {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptions()
					}
					return nil
				}).Times(1)
			},
		},
		{
			name:      "Error creating post",
			pushEvent: GetMockPushEvent(),
			setup: func() {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptions()
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error webhook post", "post", mock.AnythingOfType("*model.Post"), "error", "error creating post")
			},
		},
		{
			name:      "Successful handle post push event",
			pushEvent: GetMockPushEvent(),
			setup: func() {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptions()
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			p.postPushEvent(tc.pushEvent)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestPostCreateEvent(t *testing.T) {
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)

	tests := []struct {
		name        string
		createEvent *github.CreateEvent
		setup       func()
	}{
		{
			name:        "no subscription found",
			createEvent: GetMockCreateEvent(),
			setup: func() {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:        "unsupported ref type",
			createEvent: GetMockCreateEventWithUnsupportedRefType(),
			setup: func() {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptions()
					}
					return nil
				}).Times(1)
			},
		},
		{
			name:        "Error creating post",
			createEvent: GetMockCreateEvent(),
			setup: func() {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptions()
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error webhook post", "post", mock.AnythingOfType("*model.Post"), "error", "error creating post")
			},
		},
		{
			name:        "Successfully handle post create event",
			createEvent: GetMockCreateEvent(),
			setup: func() {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptions()
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			p.postCreateEvent(tc.createEvent)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestPostDeleteEvent(t *testing.T) {
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)

	tests := []struct {
		name        string
		deleteEvent *github.DeleteEvent
		setup       func()
	}{
		{
			name:        "no subscription found",
			deleteEvent: GetMockDeleteEvent(),
			setup: func() {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:        "non-tag and non-branch event",
			deleteEvent: GetMockDeleteEventWithInvalidType(),
			setup: func() {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptions()
					}
					return nil
				}).Times(1)
			},
		},
		{
			name:        "Error creating post",
			deleteEvent: GetMockDeleteEvent(),
			setup: func() {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptions()
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error webhook post", "post", mock.AnythingOfType("*model.Post"), "error", "error creating post")
			},
		},
		{
			name:        "Successful handle post delete event",
			deleteEvent: GetMockDeleteEvent(),
			setup: func() {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptions()
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil).Times(1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			p.postDeleteEvent(tc.deleteEvent)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestPostIssueCommentEvent(t *testing.T) {
	tests := []struct {
		name        string
		event       *github.IssueCommentEvent
		setup       func(*plugintest.API, *mocks.MockKvStore)
		expectedErr string
	}{
		{
			name:  "no subscriptions found",
			event: GetMockIssueCommentEvent(actionCreated, "mockBody", "mockUser"),
			setup: func(_ *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "event action is not created",
			event: GetMockIssueCommentEvent("edited", "mockBody", "mockUser"),
			setup: func(_ *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptions()
					}
					return nil
				}).Times(1)
			},
		},
		{
			name:  "successful event handling with no label filtering",
			event: GetMockIssueCommentEvent(actionCreated, "mockBody", "mockUser"),
			setup: func(mockAPI *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptions()
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil).Times(1)
			},
		},
		{
			name:  "error creating post",
			event: GetMockIssueCommentEvent(actionCreated, "mockBody", "mockUser"),
			setup: func(mockAPI *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptions()
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error webhook post", "post", mock.AnythingOfType("*model.Post"), "error", "error creating post").Times(1)
			},
		},
		{
			name:  "successful handle post issue comment event",
			event: GetMockIssueCommentEvent(actionCreated, "mockBody", "mockUser"),
			setup: func(mockAPI *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptions()
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil).Times(1)
			},
		},
	}

	for _, tc := range tests {
		mockKVStore, mockAPI, _, _, _ := GetTestSetup(t)
		p := getPluginTest(mockAPI, mockKVStore)

		t.Run(tc.name, func(t *testing.T) {
			mockAPI.ExpectedCalls = nil
			tc.setup(mockAPI, mockKVStore)

			p.postIssueCommentEvent(tc.event)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestSenderMutedByReceiver(t *testing.T) {
	mockStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockStore)

	tests := []struct {
		name   string
		userID string
		sender string
		setup  func()
		assert func(t *testing.T, muted bool)
	}{
		{
			name:   "sender is muted",
			userID: "user1",
			sender: "sender1",
			setup: func() {
				mockStore.EXPECT().Get("user1-muted-users", gomock.Any()).Return(nil).Do(func(key string, value interface{}) {
					*value.(*[]byte) = []byte("sender1,sender2")
				}).Times(1)
			},
			assert: func(t *testing.T, muted bool) {
				assert.True(t, muted, "Expected sender to be muted")
			},
		},
		{
			name:   "sender is not muted",
			userID: "user1",
			sender: "sender3",
			setup: func() {
				mockStore.EXPECT().Get("user1-muted-users", gomock.Any()).Return(nil).Do(func(key string, value interface{}) {
					*value.(*[]byte) = []byte("sender1,sender2")
				}).Times(1)
			},
			assert: func(t *testing.T, muted bool) {
				assert.False(t, muted, "Expected sender to not be muted")
			},
		},
		{
			name:   "error fetching muted users",
			userID: "user1",
			sender: "sender1",
			setup: func() {
				mockStore.EXPECT().Get("user1-muted-users", gomock.Any()).Return(errors.New("store error")).Times(1)
				mockAPI.On("LogWarn", "Failed to get muted users", "userID", "user1").Times(1)
			},
			assert: func(t *testing.T, muted bool) {
				assert.False(t, muted, "Expected sender to not be muted due to store error")
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			muted := p.senderMutedByReceiver(tc.userID, tc.sender)

			tc.assert(t, muted)
			mockAPI.AssertExpectations(t)
		})
	}
}

func TestPostPullRequestReviewEvent(t *testing.T) {
	tests := []struct {
		name  string
		event *github.PullRequestReviewEvent
		setup func(*plugintest.API, *mocks.MockKvStore)
	}{
		{
			name:  "no subscriptions found",
			event: GetMockPullRequestReviewEvent("submitted", "approved", MockRepoName, false, MockUserLogin, MockIssueAuthor),
			setup: func(_ *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "unsupported action in event",
			event: GetMockPullRequestReviewEvent("deleted", "approved", MockRepoName, false, MockUserLogin, MockIssueAuthor),
			setup: func(_ *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptions()
					}
					return nil
				}).Times(1)
			},
		},
		{
			name:  "unsupported review state",
			event: GetMockPullRequestReviewEvent("submitted", "canceled", MockRepoName, false, MockUserLogin, MockIssueAuthor),
			setup: func(mockAPI *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptions()
					}
					return nil
				}).Times(1)
				mockAPI.On("LogDebug", "Unhandled review state", "state", "canceled").Times(1)
			},
		},
		{
			name:  "error creating post",
			event: GetMockPullRequestReviewEvent("submitted", "approved", MockRepoName, false, MockUserLogin, MockIssueAuthor),
			setup: func(mockAPI *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptions()
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error webhook post", "post", mock.AnythingOfType("*model.Post"), "error", "error creating post").Times(1)
			},
		},
		{
			name:  "successful handling of pull request review event",
			event: GetMockPullRequestReviewEvent("submitted", "approved", MockRepoName, false, MockUserLogin, MockIssueAuthor),
			setup: func(mockAPI *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptions()
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil).Times(1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockKVStore, mockAPI, _, _, _ := GetTestSetup(t)
			p := getPluginTest(mockAPI, mockKVStore)

			mockAPI.ExpectedCalls = nil
			tc.setup(mockAPI, mockKVStore)

			p.postPullRequestReviewEvent(tc.event)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestPostPullRequestReviewCommentEvent(t *testing.T) {
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)

	tests := []struct {
		name  string
		event *github.PullRequestReviewCommentEvent
		setup func()
	}{
		{
			name:  "no subscriptions found",
			event: GetMockPullRequestReviewCommentEvent(),
			setup: func() {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "error creating post",
			event: GetMockPullRequestReviewCommentEvent(),
			setup: func() {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptions()
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error webhook post", "post", mock.AnythingOfType("*model.Post"), "error", "error creating post").Times(1)
			},
		},
		{
			name:  "successful handling of pull request review comment event",
			event: GetMockPullRequestReviewCommentEvent(),
			setup: func() {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptions()
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			p.postPullRequestReviewCommentEvent(tc.event)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestHandleCommentMentionNotification(t *testing.T) {
	tests := []struct {
		name  string
		event *github.IssueCommentEvent
		setup func(*plugintest.API, *mocks.MockKvStore)
	}{
		{
			name:  "unsupported action",
			event: GetMockIssueCommentEvent(actionEdited, "mockBody", "mockUser"),
			setup: func(*plugintest.API, *mocks.MockKvStore) {},
		},
		{
			name:  "commenter is the same as mentioned user",
			event: GetMockIssueCommentEvent(actionCreated, "mention @mockUser", "mockUser"),
			setup: func(*plugintest.API, *mocks.MockKvStore) {},
		},
		{
			name:  "comment mentions issue author",
			event: GetMockIssueCommentEvent(actionCreated, "mention @issueAuthor", "mockUser"),
			setup: func(*plugintest.API, *mocks.MockKvStore) {},
		},
		{
			name:  "error getting channel details",
			event: GetMockIssueCommentEvent(actionCreated, "mention @otherUser", "mockUser"),
			setup: func(_ *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get("otherUser_githubusername", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "error getting channel details",
			event: GetMockIssueCommentEvent(actionCreated, "mention @otherUser", "mockUser"),
			setup: func(mockAPI *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get("otherUser_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("otherUserID")
					}
					return nil
				}).Times(1)
				mockKvStore.EXPECT().Get("otherUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("GetDirectChannel", "otherUserID", "mockBotID").Return(nil, &model.AppError{Message: "error getting channel"}).Times(1)
			},
		},
		{
			name:  "error creating post",
			event: GetMockIssueCommentEvent(actionCreated, "mention @otherUser", "mockUser"),
			setup: func(mockAPI *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get("otherUser_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("otherUserID")
					}
					return nil
				}).Times(1)
				mockKvStore.EXPECT().Get("otherUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("GetDirectChannel", "otherUserID", "mockBotID").Return(&model.Channel{Id: "mockChannelID"}, nil).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error creating mention post", "error", "error creating post").Times(1)
				mockAPI.On("LogWarn", "Failed to get github user info", "error", "Must connect user account to GitHub first.").Times(1)
			},
		},
		{
			name:  "successful mention notification",
			event: GetMockIssueCommentEvent(actionCreated, "mention @otherUser", "mockUser"),
			setup: func(mockAPI *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get("otherUser_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("otherUserID")
					}
					return nil
				}).Times(1)
				mockKvStore.EXPECT().Get("otherUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("GetDirectChannel", "otherUserID", "mockBotID").Return(&model.Channel{Id: "mockChannelID"}, nil).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil).Times(1)
				mockAPI.On("LogWarn", "Failed to get github user info", "error", "Must connect user account to GitHub first.")
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockKVStore, mockAPI, _, _, _ := GetTestSetup(t)
			p := getPluginTest(mockAPI, mockKVStore)

			mockAPI.ExpectedCalls = nil
			tc.setup(mockAPI, mockKVStore)

			p.handleCommentMentionNotification(tc.event)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestHandleCommentAuthorNotification(t *testing.T) {
	tests := []struct {
		name  string
		event *github.IssueCommentEvent
		setup func(*mocks.MockKvStore, *plugintest.API)
	}{
		{
			name:  "author is the commenter",
			event: GetMockIssueCommentEvent(actionCreated, "mockBody", "issueAuthor"),
			setup: func(_ *mocks.MockKvStore, _ *plugintest.API) {},
		},
		{
			name:  "unsupported action",
			event: GetMockIssueCommentEvent(actionEdited, "mockBody", "mockUser"),
			setup: func(_ *mocks.MockKvStore, _ *plugintest.API) {},
		},
		{
			name:  "author not mapped to user ID",
			event: GetMockIssueCommentEvent(actionCreated, "mockBody", "mockUser"),
			setup: func(mockKvStore *mocks.MockKvStore, _ *plugintest.API) {
				mockKvStore.EXPECT().Get("issueAuthor_githubusername", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "author has no permission to repo",
			event: GetMockIssueCommentEvent(actionCreated, "mockBody", "mockUser"),
			setup: func(mockKvStore *mocks.MockKvStore, _ *plugintest.API) {
				mockKvStore.EXPECT().Get("issueAuthor_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("authorUserID")
					}
					return nil
				}).Times(1)
			},
		},
		{
			name:  "unhandled issue type",
			event: GetMockIssueCommentEventWithURL(actionCreated, "mockBody", "mockUser", "https://mockurl.com/unhandledType/123"),
			setup: func(mockKvStore *mocks.MockKvStore, mockAPI *plugintest.API) {
				mockKvStore.EXPECT().Get("issueAuthor_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("authorUserID")
					}
					return nil
				}).Times(1)
				mockAPI.On("LogDebug", "Unhandled issue type", "type", "unhandledType").Times(1)
			},
		},
		{
			name:  "error creating post",
			event: GetMockIssueCommentEventWithURL(actionCreated, "mockBody", "mockUser", "https://mockurl.com/issues/123"),
			setup: func(mockKvStore *mocks.MockKvStore, mockAPI *plugintest.API) {
				mockKvStore.EXPECT().Get("issueAuthor_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("authorUserID")
					}
					return nil
				}).Times(1)
				mockKvStore.EXPECT().Get("authorUserID-muted-users", gomock.Any()).Return(nil).Times(1)
				mockKvStore.EXPECT().Get("authorUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("GetDirectChannel", "authorUserID", "mockBotID").Return(&model.Channel{Id: "mockChannelID"}, nil).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Failed to get github user info", "error", "Must connect user account to GitHub first.").Times(1)
			},
		},
		{
			name:  "successful notification",
			event: GetMockIssueCommentEventWithURL(actionCreated, "mockBody", "mockUser", "https://mockurl.com/issues/123"),
			setup: func(mockKvStore *mocks.MockKvStore, mockAPI *plugintest.API) {
				mockKvStore.EXPECT().Get("issueAuthor_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("authorUserID")
					}
					return nil
				}).Times(1)
				mockKvStore.EXPECT().Get("authorUserID-muted-users", gomock.Any()).Return(nil).Times(1)
				mockKvStore.EXPECT().Get("authorUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("LogWarn", "Failed to get github user info", "error", "Must connect user account to GitHub first.").Times(1)
				mockAPI.On("GetDirectChannel", "authorUserID", "mockBotID").Return(&model.Channel{Id: "mockChannelID"}, nil).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockKVStore, mockAPI, _, _, _ := GetTestSetup(t)
			p := getPluginTest(mockAPI, mockKVStore)

			mockAPI.ExpectedCalls = nil
			tc.setup(mockKVStore, mockAPI)

			p.handleCommentAuthorNotification(tc.event)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestHandleCommentAssigneeNotification(t *testing.T) {
	tests := []struct {
		name  string
		event *github.IssueCommentEvent
		setup func(*plugintest.API, *mocks.MockKvStore)
	}{
		{
			name:  "unsupported issue type",
			event: GetMockIssueCommentEventWithAssignees("mockType", actionCreated, "mockBody", "mockUser", []string{"assigneeUser"}),
			setup: func(mockAPI *plugintest.API, _ *mocks.MockKvStore) {
				mockAPI.On("LogDebug", "Unhandled issue type", "Type", "mockType")
			},
		},
		{
			name:  "assignee is the author",
			event: GetMockIssueCommentEventWithAssignees("issues", actionCreated, "mockBody", "assigneeUser", []string{"assigneeUser"}),
			setup: func(_ *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get("assigneeUser_githubusername", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "issue author is assignee",
			event: GetMockIssueCommentEventWithAssignees("issues", actionCreated, "mockBody", "assigneeUser", []string{"issueAuthor"}),
			setup: func(_ *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get("issueAuthor_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("issueAuthor")
					}
					return nil
				}).Times(1)
			},
		},
		{
			name:  "assignee is the sender",
			event: GetMockIssueCommentEventWithAssignees("issues", actionCreated, "mockBody", "mockUser", []string{"mockUser"}),
			setup: func(_ *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get("mockUser_githubusername", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "comment mentions assignee (self-mention)",
			event: GetMockIssueCommentEventWithAssignees("issues", actionCreated, "mention @assigneeUser", "mockUser", []string{"assigneeUser"}),
			setup: func(_ *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get("assigneeUser_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("assigneeUserID")
					}
					return nil
				}).Times(1)
				mockKvStore.EXPECT().Get("assigneeUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "no permission to the repo",
			event: GetMockIssueCommentEventWithAssignees("issues", actionCreated, "mockBody", "mockUser", []string{"assigneeUser"}),
			setup: func(_ *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get("assigneeUser_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("assigneeUserID")
					}
					return nil
				}).Times(1)
				mockKvStore.EXPECT().Get("assigneeUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockKVStore, mockAPI, _, _, _ := GetTestSetup(t)
			p := getPluginTest(mockAPI, mockKVStore)

			mockAPI.ExpectedCalls = nil
			tc.setup(mockAPI, mockKVStore)

			p.handleCommentAssigneeNotification(tc.event)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestHandlePullRequestNotification(t *testing.T) {
	tests := []struct {
		name  string
		event *github.PullRequestEvent
		setup func(*plugintest.API, *mocks.MockKvStore)
	}{
		{
			name:  "review requested by sender",
			event: GetMockPullRequestEvent("review_requested", "mockRepo", MockValidLabel, false, "senderUser", "senderUser", ""),
			setup: func(*plugintest.API, *mocks.MockKvStore) {},
		},
		{
			name:  "review requested with no repo permission",
			event: GetMockPullRequestEvent("review_requested", "mockRepo", MockValidLabel, true, "senderUser", "requestedReviewer", ""),
			setup: func(_ *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get("requestedReviewer_githubusername", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "pull request closed by author",
			event: GetMockPullRequestEvent(actionClosed, "mockRepo", MockValidLabel, false, "authorUser", "authorUser", ""),
			setup: func(*plugintest.API, *mocks.MockKvStore) {},
		},
		{
			name:  "pull request closed successfully",
			event: GetMockPullRequestEvent(actionClosed, "mockRepo", MockValidLabel, false, "authorUser", "senderUser", ""),
			setup: func(mockAPI *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get("senderUser_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("authorUserID")
					}
					return nil
				}).Times(1)
				mockKvStore.EXPECT().Get("authorUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("GetDirectChannel", "authorUserID", "mockBotID").Return(&model.Channel{Id: "mockChannelID"}, nil)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil).Times(1)
				mockAPI.On("LogWarn", "Failed to get github user info", "error", "Must connect user account to GitHub first.").Times(1)
			},
		},
		{
			name:  "pull request reopened with no repo permission",
			event: GetMockPullRequestEvent(actionReopened, "mockRepo", MockValidLabel, true, "authorUser", "senderUser", ""),
			setup: func(_ *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get("senderUser_githubusername", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "pull request assigned to self",
			event: GetMockPullRequestEvent(actionAssigned, "mockRepo", MockValidLabel, false, "assigneeUser", "assigneeUser", "assigneeUser"),
			setup: func(*plugintest.API, *mocks.MockKvStore) {},
		},
		{
			name:  "pull request assigned successfully",
			event: GetMockPullRequestEvent(actionAssigned, "mockRepo", MockValidLabel, false, "senderUser", "assigneeUser", "assigneeUser"),
			setup: func(mockAPI *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get("assigneeUser_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("assigneeUserID")
					}
					return nil
				}).Times(1)
				mockAPI.On("GetDirectChannel", "assigneeUserID", "mockBotID").Return(&model.Channel{Id: "mockChannelID"}, nil)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil).Times(1)
				mockKvStore.EXPECT().Get("assigneeUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("LogWarn", "Failed to get github user info", "error", "Must connect user account to GitHub first.").Times(1)
			},
		},
		{
			name:  "review requested with valid user ID",
			event: GetMockPullRequestEvent("review_requested", "mockRepo", MockValidLabel, false, "senderUser", "requestedReviewer", ""),
			setup: func(mockAPI *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get("requestedReviewer_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("requestedUserID")
					}
					return nil
				}).Times(1)
				mockAPI.On("GetDirectChannel", "requestedUserID", "mockBotID").Return(&model.Channel{Id: "mockChannelID"}, nil)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil).Times(1)
				mockKvStore.EXPECT().Get("requestedUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("LogWarn", "Failed to get github user info", "error", "Must connect user account to GitHub first.").Times(1)
			},
		},
		{
			name: "unhandled event action",
			event: GetMockPullRequestEvent(
				"unsupported_action", "mockRepo", MockValidLabel, false, "senderUser", "", ""),
			setup: func(mockAPI *plugintest.API, _ *mocks.MockKvStore) {
				mockAPI.On("LogDebug", "Unhandled event action", "action", "unsupported_action").Return(nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockKVStore, mockAPI, _, _, _ := GetTestSetup(t)
			p := getPluginTest(mockAPI, mockKVStore)

			mockAPI.ExpectedCalls = nil
			tc.setup(mockAPI, mockKVStore)

			p.handlePullRequestNotification(tc.event)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestHandleIssueNotification(t *testing.T) {
	tests := []struct {
		name  string
		event *github.IssuesEvent
		setup func(*plugintest.API, *mocks.MockKvStore)
	}{
		{
			name:  "issue closed by author",
			event: GetMockIssuesEvent(actionClosed, MockRepo, false, "authorUser", "authorUser", ""),
			setup: func(*plugintest.API, *mocks.MockKvStore) {},
		},
		{
			name:  "issue closed successfully",
			event: GetMockIssuesEvent(actionClosed, MockRepo, true, "authorUser", "senderUser", ""),
			setup: func(_ *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get("authorUser_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("authorUserID")
					}
					return nil
				}).Times(1)
				mockKvStore.EXPECT().Get("authorUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "issue reopened with no repo permission",
			event: GetMockIssuesEvent(actionReopened, MockRepo, true, "authorUser", "senderUser", ""),
			setup: func(_ *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get("authorUser_githubusername", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "issue assigned to self",
			event: GetMockIssuesEvent(actionAssigned, MockRepo, false, "assigneeUser", "assigneeUser", "assigneeUser"),
			setup: func(*plugintest.API, *mocks.MockKvStore) {},
		},
		{
			name:  "issue assigned successfully",
			event: GetMockIssuesEvent(actionAssigned, MockRepo, false, "senderUser", "assigneeUser", "assigneeUser"),
			setup: func(_ *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get("assigneeUser_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("assigneeUserID")
					}
					return nil
				}).Times(1)
			},
		},
		{
			name:  "issue assigned with no repo permission for assignee",
			event: GetMockIssuesEvent(actionAssigned, MockRepo, true, "senderUser", "demoassigneeUser", "assigneeUser"),
			setup: func(_ *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get("assigneeUser_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("assigneeUserID")
					}
					return nil
				}).Times(1)
			},
		},
		{
			name:  "unhandled event action",
			event: GetMockIssuesEvent("unsupported_action", MockRepo, false, "senderUser", "", ""),
			setup: func(mockAPI *plugintest.API, _ *mocks.MockKvStore) {
				mockAPI.On("LogDebug", "Unhandled event action", "action", "unsupported_action").Return(nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		mockKVStore, mockAPI, _, _, _ := GetTestSetup(t)
		p := getPluginTest(mockAPI, mockKVStore)

		t.Run(tc.name, func(t *testing.T) {
			tc.setup(mockAPI, mockKVStore)

			p.handleIssueNotification(tc.event)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestHandlePullRequestReviewNotification(t *testing.T) {
	tests := []struct {
		name  string
		event *github.PullRequestReviewEvent
		setup func(*plugintest.API, *mocks.MockKvStore)
	}{
		{
			name:  "review submitted by author",
			event: GetMockPullRequestReviewEvent(actionSubmitted, "approved", MockRepo, false, "authorUser", "authorUser"),
			setup: func(_ *plugintest.API, _ *mocks.MockKvStore) {},
		},
		{
			name:  "review action not submitted",
			event: GetMockPullRequestReviewEvent("dismissed", "approved", MockRepo, false, "authorUser", "reviewerUser"),
			setup: func(_ *plugintest.API, _ *mocks.MockKvStore) {},
		},
		{
			name:  "review with author not mapped to user ID",
			event: GetMockPullRequestReviewEvent(actionSubmitted, "approved", MockRepo, false, "unknownAuthor", "reviewerUser"),
			setup: func(_ *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get("reviewerUser_githubusername", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "private repo, no permission for author",
			event: GetMockPullRequestReviewEvent(actionSubmitted, "approved", MockRepo, true, "authorUser", "reviewerUser"),
			setup: func(_ *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get("reviewerUser_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("authorUserID")
					}
					return nil
				}).Times(1)
				mockKvStore.EXPECT().Get("authorUserID_githubtoken", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("authorUserID")
					}
					return nil
				}).Times(1)
			},
		},
		{
			name:  "successful review notification",
			event: GetMockPullRequestReviewEvent(actionSubmitted, "approved", MockRepo, false, "authorUser", "reviewerUser"),
			setup: func(mockAPI *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get("reviewerUser_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("authorUserID")
					}
					return nil
				}).Times(1)
				mockAPI.On("GetDirectChannel", "authorUserID", "mockBotID").Return(nil, &model.AppError{Message: "error getting channel"}).Times(1)
				mockAPI.On("LogWarn", "Couldn't get bot's DM channel", "userID", "authorUserID", "error", "error getting channel")
				mockKvStore.EXPECT().Get("authorUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("LogWarn", "Failed to get github user info", "error", "Must connect user account to GitHub first.")
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockKVStore, mockAPI, _, _, _ := GetTestSetup(t)
			p := getPluginTest(mockAPI, mockKVStore)

			tc.setup(mockAPI, mockKVStore)

			p.handlePullRequestReviewNotification(tc.event)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestPostStarEvent(t *testing.T) {
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)

	tests := []struct {
		name  string
		event *github.StarEvent
		setup func()
	}{
		{
			name:  "no subscribed channels for repository",
			event: GetMockStarEvent(MockRepo, MockOrg, false, MockSender),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "error creating post",
			event: GetMockStarEvent(MockRepo, MockOrg, false, MockSender),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockrepo/mockorg", featureStars)
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error webhook post", "post", mock.AnythingOfType("*model.Post"), "error", "error creating post")
			},
		},
		{
			name:  "successful star event notification",
			event: GetMockStarEvent(MockRepo, MockOrg, false, MockSender),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockrepo/mockorg", featureStars)
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			p.postStarEvent(tc.event)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestPostReleaseEvent(t *testing.T) {
	tests := []struct {
		name  string
		event *github.ReleaseEvent
		setup func(*plugintest.API, *mocks.MockKvStore)
	}{
		{
			name:  "no subscribed channels for repository",
			event: GetMockReleaseEvent(MockRepo, MockOrg, "created", MockSender),
			setup: func(_ *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "unsupported action",
			event: GetMockReleaseEvent(MockRepo, MockOrg, "edited", MockSender),
			setup: func(mockAPI *plugintest.API, _ *mocks.MockKvStore) {},
		},
		{
			name:  "error creating post",
			event: GetMockReleaseEvent(MockRepo, MockOrg, "created", MockSender),
			setup: func(mockAPI *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockrepo/mockorg", featureReleases)
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error webhook post", "Post", mock.AnythingOfType("*model.Post"), "Error", "error creating post")
			},
		},
		{
			name:  "successful release event notification",
			event: GetMockReleaseEvent(MockRepo, MockOrg, "created", MockSender),
			setup: func(mockAPI *plugintest.API, mockKvStore *mocks.MockKvStore) {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockrepo/mockorg", featureReleases)
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockKVStore, mockAPI, _, _, _ := GetTestSetup(t)
			p := getPluginTest(mockAPI, mockKVStore)

			tc.setup(mockAPI, mockKVStore)

			p.postReleaseEvent(tc.event)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestPostDiscussionEvent(t *testing.T) {
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)

	tests := []struct {
		name  string
		event *github.DiscussionEvent
		setup func()
	}{
		{
			name:  "no subscribed channels for repository",
			event: GetMockDiscussionEvent(MockRepo, MockOrg, MockSender),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "error creating discussion post",
			event: GetMockDiscussionEvent(MockRepo, MockOrg, MockSender),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockrepo/mockorg", featureDiscussions)
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error creating discussion notification post", "Post", mock.AnythingOfType("*model.Post"), "Error", "error creating post")
			},
		},
		{
			name:  "successful discussion notification",
			event: GetMockDiscussionEvent(MockRepo, MockOrg, MockSender),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockrepo/mockorg", featureDiscussions)
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			p.postDiscussionEvent(tc.event)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestPostDiscussionCommentEvent(t *testing.T) {
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)

	tests := []struct {
		name  string
		event *github.DiscussionCommentEvent
		setup func()
	}{
		{
			name:  "no subscribed channels for repository",
			event: GetMockDiscussionCommentEvent(MockRepo, MockOrg, "created", MockSender),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "error creating discussion comment post",
			event: GetMockDiscussionCommentEvent(MockRepo, MockOrg, "created", MockSender),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockrepo/mockorg", featureDiscussionComments)
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error creating discussion comment post", "Post", mock.AnythingOfType("*model.Post"), "Error", "error creating post")
			},
		},
		{
			name:  "successful discussion comment notification",
			event: GetMockDiscussionCommentEvent(MockRepo, MockOrg, "created", MockSender),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptionWithLabel("mockrepo/mockorg", featureDiscussionComments)
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockAPI.ExpectedCalls = nil
			tc.setup()

			p.postDiscussionCommentEvent(tc.event)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestIncludeOnlyOrgMembers(t *testing.T) {
	tests := []struct {
		name         string
		user         github.User
		subscription Subscription
		expectWarn   bool
		want         bool
	}{
		{
			name: "IncludeOnlyOrgMembers flag is false",
			user: github.User{
				Login: github.String(orgMember),
			},
			subscription: Subscription{
				Flags: SubscriptionFlags{IncludeOnlyOrgMembers: false},
			},
			expectWarn: false,
			want:       false,
		},
		{
			name: "Failed to get GitHub Client",
			user: github.User{
				Login: github.String(orgMember),
			},
			subscription: Subscription{
				CreatorID: model.NewId(),
				Flags:     SubscriptionFlags{IncludeOnlyOrgMembers: true},
			},
			expectWarn: true,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			user := *tt.user.Login
			server := mockGitHubServer(user)
			gitHubPlugin := newPlugin(model.NewId(), server.URL)
			api := plugintest.NewAPI(t)
			if tt.expectWarn {
				api.On("LogWarn", mock.AnythingOfType("string"), "error", mock.AnythingOfType("string")).Return(nil)
			}
			gitHubPlugin.SetAPI(api)
			gitHubPlugin.client = pluginapi.NewClient(gitHubPlugin.API, gitHubPlugin.Driver)

			got := gitHubPlugin.shouldDenyEventDueToNotOrgMember(&tt.user, &tt.subscription)
			assert.Equal(t, tt.want, got)
		})
	}
}

func mockSubscription(mockKVStore *mocks.MockKvStore) {
	mockKVStore.EXPECT().Get(SubscriptionsKey, mock.MatchedBy(func(val interface{}) bool {
		_, ok := val.(**Subscriptions)
		return ok
	})).DoAndReturn(func(key string, value interface{}) error {
		if v, ok := value.(**Subscriptions); ok {
			*v = GetMockSubscriptions()
		}
		return nil
	}).Times(1)
}

func setupMockSubscriptions(subs map[string][]*Subscription) func(string, interface{}) error {
	return func(_ string, value interface{}) error {
		if v, ok := value.(**Subscriptions); ok {
			*v = &Subscriptions{
				Repositories: subs,
			}
		}
		return nil
	}
}

func setByteValue(data string) func(key string, value interface{}) error {
	return func(key string, value interface{}) error {
		if v, ok := value.(*[]byte); ok {
			*v = []byte(data)
		}
		return nil
	}
}

func newPlugin(userID string, gitHubURL string) *Plugin {
	p := NewPlugin()
	p.initializeAPI()
	p.SetDriver(&plugintest.Driver{})
	p.store = &pluginapi.MemoryStore{}
	token, _ := generateSecret()
	encryptionKey, _ := generateSecret()
	encryptedToken, _ := encrypt([]byte(encryptionKey), token)
	_, _ = p.store.Set(userID+githubTokenKey, GitHubUserInfo{
		UserID: userID,
		Token: &oauth2.Token{
			AccessToken: encryptedToken,
		},
	})
	p.setConfiguration(&Configuration{
		EncryptionKey:       encryptionKey,
		GitHubOrg:           gitHubOrginization,
		WebhookSecret:       webhookSecret,
		EnterpriseBaseURL:   gitHubURL,
		EnterpriseUploadURL: gitHubURL,
	})

	_ = p.AddSubscription(
		gitHubOrginization+"/test-repo",
		&Subscription{
			ChannelID: "1",
			CreatorID: userID,
			Features:  Features(strings.Join([]string{featureIssues, featureIssueCreation}, ",")),
			Flags:     SubscriptionFlags{IncludeOnlyOrgMembers: true},
		},
	)

	return p
}

func mockGitHubServer(user string) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != fmt.Sprintf("/api/v3/orgs/%v/members/%v", gitHubOrginization, user) {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		if user == orgMember {
			w.WriteHeader(http.StatusNoContent)
		} else if user == orgCollaborator {
			w.WriteHeader(http.StatusFound)
		}
	}))

	return ts
}
