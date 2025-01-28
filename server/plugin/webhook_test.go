package plugin

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-github/v54/github"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"

	"github.com/mattermost/mattermost-plugin-github/server/mocks"
)

func TestPostPushEvent(t *testing.T) {
	tests := []struct {
		name      string
		pushEvent *github.PushEvent
		setup     func(*plugintest.API, *mocks.MockKvStore)
	}{
		{
			name:      "No subscription found",
			pushEvent: GetMockPushEvent(),
			setup: func(_ *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockKVStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:      "No commits found in event",
			pushEvent: GetMockPushEventWithoutCommit(),
			setup: func(_ *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockSubscription(mockKVStore)
			},
		},
		{
			name:      "Error creating post",
			pushEvent: GetMockPushEvent(),
			setup: func(mockAPI *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockSubscription(mockKVStore)
				mockAPI.On("CreatePost", mock.Anything).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error webhook post", "post", mock.Anything, "error", "error creating post")
			},
		},
		{
			name:      "Successful handle post push event",
			pushEvent: GetMockPushEvent(),
			setup: func(mockAPI *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockSubscription(mockKVStore)
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		mockKVStore, mockAPI, _, _, _ := GetTestSetup(t)
		p := getPluginTest(mockAPI, mockKVStore)

		t.Run(tc.name, func(t *testing.T) {
			tc.setup(mockAPI, mockKVStore)

			p.postPushEvent(tc.pushEvent)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestPostCreateEvent(t *testing.T) {
	tests := []struct {
		name        string
		createEvent *github.CreateEvent
		setup       func(*plugintest.API, *mocks.MockKvStore)
	}{
		{
			name:        "No subscription found",
			createEvent: GetMockCreateEvent(),
			setup: func(_ *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockKVStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:        "Unsupported ref type",
			createEvent: GetMockCreateEventWithUnsupportedRefType(),
			setup: func(_ *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockSubscription(mockKVStore)
			},
		},
		{
			name:        "Error creating post",
			createEvent: GetMockCreateEvent(),
			setup: func(mockAPI *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockSubscription(mockKVStore)
				mockAPI.On("CreatePost", mock.Anything).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error webhook post", "post", mock.Anything, "error", "error creating post")
			},
		},
		{
			name:        "Successfully handle post create event",
			createEvent: GetMockCreateEvent(),
			setup: func(mockAPI *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockSubscription(mockKVStore)
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockKVStore, mockAPI, _, _, _ := GetTestSetup(t)
			p := getPluginTest(mockAPI, mockKVStore)

			tc.setup(mockAPI, mockKVStore)

			p.postCreateEvent(tc.createEvent)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestPostDeleteEvent(t *testing.T) {
	tests := []struct {
		name        string
		deleteEvent *github.DeleteEvent
		setup       func(*plugintest.API, *mocks.MockKvStore)
	}{
		{
			name:        "No subscription found",
			deleteEvent: GetMockDeleteEvent(),
			setup: func(_ *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockKVStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:        "Non-tag and non-branch event",
			deleteEvent: GetMockDeleteEventWithInvalidType(),
			setup: func(_ *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockSubscription(mockKVStore)
			},
		},
		{
			name:        "Error creating post",
			deleteEvent: GetMockDeleteEvent(),
			setup: func(mockAPI *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockSubscription(mockKVStore)
				mockAPI.On("CreatePost", mock.Anything).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error webhook post", "post", mock.Anything, "error", "error creating post")
			},
		},
		{
			name:        "Successful handle post delete event",
			deleteEvent: GetMockDeleteEvent(),
			setup: func(mockAPI *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockSubscription(mockKVStore)
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockKVStore, mockAPI, _, _, _ := GetTestSetup(t)
			p := getPluginTest(mockAPI, mockKVStore)

			tc.setup(mockAPI, mockKVStore)

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
			name:  "No subscriptions found",
			event: GetMockIssueCommentEvent(actionCreated, "mockBody", "mockUser"),
			setup: func(_ *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockKVStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "Event action is not created",
			event: GetMockIssueCommentEvent("edited", "mockBody", "mockUser"),
			setup: func(_ *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockSubscription(mockKVStore)
			},
		},
		{
			name:  "Successful event handling with no label filtering",
			event: GetMockIssueCommentEvent(actionCreated, "mockBody", "mockUser"),
			setup: func(mockAPI *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockSubscription(mockKVStore)
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
			},
		},
		{
			name:  "Error creating post",
			event: GetMockIssueCommentEvent(actionCreated, "mockBody", "mockUser"),
			setup: func(mockAPI *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockSubscription(mockKVStore)
				mockAPI.On("CreatePost", mock.Anything).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error webhook post", "post", mock.Anything, "error", "error creating post").Times(1)
			},
		},
		{
			name:  "Successful handle post issue comment event",
			event: GetMockIssueCommentEvent(actionCreated, "mockBody", "mockUser"),
			setup: func(mockAPI *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockSubscription(mockKVStore)
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockKVStore, mockAPI, _, _, _ := GetTestSetup(t)
			p := getPluginTest(mockAPI, mockKVStore)

			tc.setup(mockAPI, mockKVStore)

			p.postIssueCommentEvent(tc.event)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestSenderMutedByReceiver(t *testing.T) {
	tests := []struct {
		name   string
		userID string
		sender string
		setup  func(*mocks.MockKvStore, *plugintest.API)
		assert func(t *testing.T, muted bool)
	}{
		{
			name:   "Sender is muted",
			userID: "user1",
			sender: "sender1",
			setup: func(mockKVStore *mocks.MockKvStore, _ *plugintest.API) {
				mockKVStore.EXPECT().Get("user1-muted-users", gomock.Any()).Return(nil).Do(func(key string, value interface{}) {
					*value.(*[]byte) = []byte("sender1,sender2")
				}).Times(1)
			},
			assert: func(t *testing.T, muted bool) {
				assert.True(t, muted, "Expected sender to be muted")
			},
		},
		{
			name:   "Sender is not muted",
			userID: "user1",
			sender: "sender3",
			setup: func(mockKVStore *mocks.MockKvStore, _ *plugintest.API) {
				mockKVStore.EXPECT().Get("user1-muted-users", gomock.Any()).Return(nil).Do(func(key string, value interface{}) {
					*value.(*[]byte) = []byte("sender1,sender2")
				}).Times(1)
			},
			assert: func(t *testing.T, muted bool) {
				assert.False(t, muted, "Expected sender to not be muted")
			},
		},
		{
			name:   "Error fetching muted users",
			userID: "user1",
			sender: "sender1",
			setup: func(mockKVStore *mocks.MockKvStore, mockAPI *plugintest.API) {
				mockKVStore.EXPECT().Get("user1-muted-users", gomock.Any()).Return(errors.New("store error")).Times(1)
				mockAPI.On("LogWarn", "Failed to get muted users", "userID", "user1").Times(1)
			},
			assert: func(t *testing.T, muted bool) {
				assert.False(t, muted, "Expected sender to not be muted due to store error")
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockKVStore, mockAPI, _, _, _ := GetTestSetup(t)
			p := getPluginTest(mockAPI, mockKVStore)

			tc.setup(mockKVStore, mockAPI)

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
			name:  "No subscriptions found",
			event: GetMockPullRequestReviewEvent("submitted", "approved"),
			setup: func(_ *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockKVStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "Unsupported action in event",
			event: GetMockPullRequestReviewEvent("deleted", "approved"),
			setup: func(_ *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockSubscription(mockKVStore)
			},
		},
		{
			name:  "Unsupported review state",
			event: GetMockPullRequestReviewEvent("submitted", "canceled"),
			setup: func(mockAPI *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockSubscription(mockKVStore)
				mockAPI.On("LogDebug", "Unhandled review state", "state", "canceled").Times(1)
			},
		},
		{
			name:  "Error creating post",
			event: GetMockPullRequestReviewEvent("submitted", "approved"),
			setup: func(mockAPI *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockSubscription(mockKVStore)
				mockAPI.On("CreatePost", mock.Anything).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error webhook post", "post", mock.Anything, "error", "error creating post").Times(1)
			},
		},
		{
			name:  "Successful handling of pull request review event",
			event: GetMockPullRequestReviewEvent("submitted", "approved"),
			setup: func(mockAPI *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockSubscription(mockKVStore)
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockKVStore, mockAPI, _, _, _ := GetTestSetup(t)
			p := getPluginTest(mockAPI, mockKVStore)

			tc.setup(mockAPI, mockKVStore)

			p.postPullRequestReviewEvent(tc.event)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestPostPullRequestReviewCommentEvent(t *testing.T) {
	tests := []struct {
		name  string
		event *github.PullRequestReviewCommentEvent
		setup func(*plugintest.API, *mocks.MockKvStore)
	}{
		{
			name:  "No subscriptions found",
			event: GetMockPullRequestReviewCommentEvent(),
			setup: func(_ *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockKVStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "Error creating post",
			event: GetMockPullRequestReviewCommentEvent(),
			setup: func(mockAPI *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockSubscription(mockKVStore)
				mockAPI.On("CreatePost", mock.Anything).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error webhook post", "post", mock.Anything, "error", "error creating post").Times(1)
			},
		},
		{
			name:  "Successful handling of pull request review comment event",
			event: GetMockPullRequestReviewCommentEvent(),
			setup: func(mockAPI *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockSubscription(mockKVStore)
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockKVStore, mockAPI, _, _, _ := GetTestSetup(t)
			p := getPluginTest(mockAPI, mockKVStore)

			tc.setup(mockAPI, mockKVStore)

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
			name:  "Unsupported action",
			event: GetMockIssueCommentEvent(actionEdited, "mockBody", "mockUser"),
			setup: func(_ *plugintest.API, _ *mocks.MockKvStore) {},
		},
		{
			name:  "Commenter is the same as mentioned user",
			event: GetMockIssueCommentEvent(actionCreated, "mention @mockUser", "mockUser"),
			setup: func(_ *plugintest.API, _ *mocks.MockKvStore) {},
		},
		{
			name:  "Comment mentions issue author",
			event: GetMockIssueCommentEvent(actionCreated, "mention @issueAuthor", "mockUser"),
			setup: func(_ *plugintest.API, _ *mocks.MockKvStore) {},
		},
		{
			name:  "Error getting channel details",
			event: GetMockIssueCommentEvent(actionCreated, "mention @otherUser", "mockUser"),
			setup: func(_ *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockKVStore.EXPECT().Get("otherUser_githubusername", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "Error getting channel details",
			event: GetMockIssueCommentEvent(actionCreated, "mention @otherUser", "mockUser"),
			setup: func(mockAPI *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockKVStore.EXPECT().Get("otherUser_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("otherUserID")
					}
					return nil
				}).Times(1)
				mockKVStore.EXPECT().Get("otherUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("GetDirectChannel", "otherUserID", "mockBotID").Return(nil, &model.AppError{Message: "error getting channel"}).Times(1)
			},
		},
		{
			name:  "Error creating post",
			event: GetMockIssueCommentEvent(actionCreated, "mention @otherUser", "mockUser"),
			setup: func(mockAPI *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockKVStore.EXPECT().Get("otherUser_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("otherUserID")
					}
					return nil
				}).Times(1)
				mockKVStore.EXPECT().Get("otherUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("GetDirectChannel", "otherUserID", "mockBotID").Return(&model.Channel{Id: "mockChannelID"}, nil).Times(1)
				mockAPI.On("CreatePost", mock.Anything).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error creating mention post", "error", "error creating post").Times(1)
				mockAPI.On("LogWarn", "Failed to get github user info", "error", "Must connect user account to GitHub first.").Times(1)
			},
		},
		{
			name:  "Successful mention notification",
			event: GetMockIssueCommentEvent(actionCreated, "mention @otherUser", "mockUser"),
			setup: func(mockAPI *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockKVStore.EXPECT().Get("otherUser_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("otherUserID")
					}
					return nil
				}).Times(1)
				mockKVStore.EXPECT().Get("otherUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("GetDirectChannel", "otherUserID", "mockBotID").Return(&model.Channel{Id: "mockChannelID"}, nil).Times(1)
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
				mockAPI.On("LogWarn", "Failed to get github user info", "error", "Must connect user account to GitHub first.")
			},
		},
	}
	for _, tc := range tests {
		mockKVStore, mockAPI, _, _, _ := GetTestSetup(t)
		p := getPluginTest(mockAPI, mockKVStore)

		t.Run(tc.name, func(t *testing.T) {
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
		setup func(*plugintest.API, *mocks.MockKvStore)
	}{
		{
			name:  "Author is the commenter",
			event: GetMockIssueCommentEvent(actionCreated, "mockBody", "issueAuthor"),
			setup: func(_ *plugintest.API, _ *mocks.MockKvStore) {},
		},
		{
			name:  "Unsupported action",
			event: GetMockIssueCommentEvent(actionEdited, "mockBody", "mockUser"),
			setup: func(_ *plugintest.API, _ *mocks.MockKvStore) {},
		},
		{
			name:  "Author not mapped to user ID",
			event: GetMockIssueCommentEvent(actionCreated, "mockBody", "mockUser"),
			setup: func(_ *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockKVStore.EXPECT().Get("issueAuthor_githubusername", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "Author has no permission to repo",
			event: GetMockIssueCommentEvent(actionCreated, "mockBody", "mockUser"),
			setup: func(_ *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockKVStore.EXPECT().Get("issueAuthor_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("authorUserID")
					}
					return nil
				}).Times(1)
			},
		},
		{
			name:  "Unhandled issue type",
			event: GetMockIssueCommentEventWithURL(actionCreated, "mockBody", "mockUser", "https://mockurl.com/unhandledType/123"),
			setup: func(mockAPI *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockKVStore.EXPECT().Get("issueAuthor_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("authorUserID")
					}
					return nil
				}).Times(1)
				mockAPI.On("LogDebug", "Unhandled issue type", "type", "unhandledType").Times(1)
			},
		},
		{
			name:  "Error creating post",
			event: GetMockIssueCommentEventWithURL(actionCreated, "mockBody", "mockUser", "https://mockurl.com/issues/123"),
			setup: func(mockAPI *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockKVStore.EXPECT().Get("issueAuthor_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("authorUserID")
					}
					return nil
				}).Times(1)
				mockKVStore.EXPECT().Get("authorUserID-muted-users", gomock.Any()).Return(nil).Times(1)
				mockKVStore.EXPECT().Get("authorUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("GetDirectChannel", "authorUserID", "mockBotID").Return(&model.Channel{Id: "mockChannelID"}, nil).Times(1)
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Failed to get github user info", "error", "Must connect user account to GitHub first.").Times(1)
			},
		},
		{
			name:  "Successful notification",
			event: GetMockIssueCommentEventWithURL(actionCreated, "mockBody", "mockUser", "https://mockurl.com/issues/123"),
			setup: func(mockAPI *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockKVStore.EXPECT().Get("issueAuthor_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("authorUserID")
					}
					return nil
				}).Times(1)
				mockKVStore.EXPECT().Get("authorUserID-muted-users", gomock.Any()).Return(nil).Times(1)
				mockKVStore.EXPECT().Get("authorUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("LogWarn", "Failed to get github user info", "error", "Must connect user account to GitHub first.").Times(1)
				mockAPI.On("GetDirectChannel", "authorUserID", "mockBotID").Return(&model.Channel{Id: "mockChannelID"}, nil).Times(1)
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockKVStore, mockAPI, _, _, _ := GetTestSetup(t)
			p := getPluginTest(mockAPI, mockKVStore)

			tc.setup(mockAPI, mockKVStore)

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
			name:  "Unsupported issue type",
			event: GetMockIssueCommentEventWithAssignees("mockType", actionCreated, "mockBody", "mockUser", []string{"assigneeUser"}),
			setup: func(mockAPI *plugintest.API, _ *mocks.MockKvStore) {
				mockAPI.On("LogDebug", "Unhandled issue type", "Type", "mockType")
			},
		},
		{
			name:  "Assignee is the author",
			event: GetMockIssueCommentEventWithAssignees("issues", actionCreated, "mockBody", "assigneeUser", []string{"assigneeUser"}),
			setup: func(_ *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockKVStore.EXPECT().Get("assigneeUser_githubusername", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "Issue author is assignee",
			event: GetMockIssueCommentEventWithAssignees("issues", actionCreated, "mockBody", "assigneeUser", []string{"issueAuthor"}),
			setup: func(mockAPI *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockKVStore.EXPECT().Get("issueAuthor_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("issueAuthor")
					}
					return nil
				}).Times(1)
			},
		},
		{
			name:  "Assignee is the sender",
			event: GetMockIssueCommentEventWithAssignees("issues", actionCreated, "mockBody", "mockUser", []string{"mockUser"}),
			setup: func(_ *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockKVStore.EXPECT().Get("mockUser_githubusername", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "Comment mentions assignee (self-mention)",
			event: GetMockIssueCommentEventWithAssignees("issues", actionCreated, "mention @assigneeUser", "mockUser", []string{"assigneeUser"}),
			setup: func(_ *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockKVStore.EXPECT().Get("assigneeUser_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("assigneeUserID")
					}
					return nil
				}).Times(1)
				mockKVStore.EXPECT().Get("assigneeUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "No permission to the repo",
			event: GetMockIssueCommentEventWithAssignees("issues", actionCreated, "mockBody", "mockUser", []string{"assigneeUser"}),
			setup: func(_ *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockKVStore.EXPECT().Get("assigneeUser_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("assigneeUserID")
					}
					return nil
				}).Times(1)
				mockKVStore.EXPECT().Get("assigneeUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockKVStore, mockAPI, _, _, _ := GetTestSetup(t)
			p := getPluginTest(mockAPI, mockKVStore)

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
			name:  "Review requested by sender",
			event: GetMockPullRequestEvent("review_requested", "mockRepo", false, "senderUser", "senderUser", ""),
			setup: func(_ *plugintest.API, _ *mocks.MockKvStore) {},
		},
		{
			name:  "Review requested with no repo permission",
			event: GetMockPullRequestEvent("review_requested", "mockRepo", true, "senderUser", "requestedReviewer", ""),
			setup: func(_ *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockKVStore.EXPECT().Get("requestedReviewer_githubusername", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "Pull request closed by author",
			event: GetMockPullRequestEvent(actionClosed, "mockRepo", false, "authorUser", "authorUser", ""),
			setup: func(_ *plugintest.API, _ *mocks.MockKvStore) {},
		},
		{
			name:  "Pull request closed successfully",
			event: GetMockPullRequestEvent(actionClosed, "mockRepo", false, "authorUser", "senderUser", ""),
			setup: func(mockAPI *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockKVStore.EXPECT().Get("senderUser_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("authorUserID")
					}
					return nil
				}).Times(1)
				mockKVStore.EXPECT().Get("authorUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("GetDirectChannel", "authorUserID", "mockBotID").Return(&model.Channel{Id: "mockChannelID"}, nil)
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
				mockAPI.On("LogWarn", "Failed to get github user info", "error", "Must connect user account to GitHub first.").Times(1)
			},
		},
		{
			name:  "Pull request reopened with no repo permission",
			event: GetMockPullRequestEvent(actionReopened, "mockRepo", true, "authorUser", "senderUser", ""),
			setup: func(_ *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockKVStore.EXPECT().Get("senderUser_githubusername", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "Pull request assigned to self",
			event: GetMockPullRequestEvent(actionAssigned, "mockRepo", false, "assigneeUser", "assigneeUser", "assigneeUser"),
			setup: func(_ *plugintest.API, _ *mocks.MockKvStore) {},
		},
		{
			name:  "Pull request assigned successfully",
			event: GetMockPullRequestEvent(actionAssigned, "mockRepo", false, "senderUser", "assigneeUser", "assigneeUser"),
			setup: func(mockAPI *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockKVStore.EXPECT().Get("assigneeUser_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("assigneeUserID")
					}
					return nil
				}).Times(1)
				mockAPI.On("GetDirectChannel", "assigneeUserID", "mockBotID").Return(&model.Channel{Id: "mockChannelID"}, nil)
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
				mockKVStore.EXPECT().Get("assigneeUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("LogWarn", "Failed to get github user info", "error", "Must connect user account to GitHub first.").Times(1)
			},
		},
		{
			name:  "Review requested with valid user ID",
			event: GetMockPullRequestEvent("review_requested", "mockRepo", false, "senderUser", "requestedReviewer", ""),
			setup: func(mockAPI *plugintest.API, mockKVStore *mocks.MockKvStore) {
				mockKVStore.EXPECT().Get("requestedReviewer_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("requestedUserID")
					}
					return nil
				}).Times(1)
				mockAPI.On("GetDirectChannel", "requestedUserID", "mockBotID").Return(&model.Channel{Id: "mockChannelID"}, nil)
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
				mockKVStore.EXPECT().Get("requestedUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("LogWarn", "Failed to get github user info", "error", "Must connect user account to GitHub first.").Times(1)
			},
		},
		{
			name: "Unhandled event action",
			event: GetMockPullRequestEvent(
				"unsupported_action", "mockRepo", false, "senderUser", "", ""),
			setup: func(mockAPI *plugintest.API, _ *mocks.MockKvStore) {
				mockAPI.On("LogDebug", "Unhandled event action", "action", "unsupported_action").Return(nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockKVStore, mockAPI, _, _, _ := GetTestSetup(t)
			p := getPluginTest(mockAPI, mockKVStore)

			tc.setup(mockAPI, mockKVStore)

			p.handlePullRequestNotification(tc.event)

			mockAPI.AssertExpectations(t)
		})
	}
}

func mockSubscription(mockKVStore *mocks.MockKvStore) {
	mockKVStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
		if v, ok := value.(**Subscriptions); ok {
			*v = GetMockSubscriptions()
		}
		return nil
	}).Times(1)
}
