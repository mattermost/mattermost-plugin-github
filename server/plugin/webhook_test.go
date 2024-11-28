package plugin

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-github/v54/github"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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
				mockAPI.On("CreatePost", mock.Anything).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error webhook post", "post", mock.Anything, "error", "error creating post")
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
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
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
				mockAPI.On("CreatePost", mock.Anything).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error webhook post", "post", mock.Anything, "error", "error creating post")
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
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
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
				mockAPI.On("CreatePost", mock.Anything).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error webhook post", "post", mock.Anything, "error", "error creating post")
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
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
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
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)

	tests := []struct {
		name        string
		event       *github.IssueCommentEvent
		setup       func()
		expectedErr string
	}{
		{
			name:  "no subscriptions found",
			event: GetMockIssueCommentEvent(actionCreated, "mockBody", "mockUser"),
			setup: func() {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "event action is not created",
			event: GetMockIssueCommentEvent("edited", "mockBody", "mockUser"),
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
			name:  "successful event handling with no label filtering",
			event: GetMockIssueCommentEvent(actionCreated, "mockBody", "mockUser"),
			setup: func() {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptions()
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
			},
		},
		{
			name:  "error creating post",
			event: GetMockIssueCommentEvent(actionCreated, "mockBody", "mockUser"),
			setup: func() {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptions()
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.Anything).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error webhook post", "post", mock.Anything, "error", "error creating post").Times(1)
			},
		},
		{
			name:  "successful handle post issue comment event",
			event: GetMockIssueCommentEvent(actionCreated, "mockBody", "mockUser"),
			setup: func() {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptions()
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

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
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)

	tests := []struct {
		name  string
		event *github.PullRequestReviewEvent
		setup func()
	}{
		{
			name:  "no subscriptions found",
			event: GetMockPullRequestReviewEvent("submitted", "approved", MockRepoName, false, MockUserLogin, MockIssueAuthor),
			setup: func() {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "unsupported action in event",
			event: GetMockPullRequestReviewEvent("deleted", "approved", MockRepoName, false, MockUserLogin, MockIssueAuthor),
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
			name:  "unsupported review state",
			event: GetMockPullRequestReviewEvent("submitted", "canceled", MockRepoName, false, MockUserLogin, MockIssueAuthor),
			setup: func() {
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
			setup: func() {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptions()
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.Anything).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error webhook post", "post", mock.Anything, "error", "error creating post").Times(1)
			},
		},
		{
			name:  "successful handling of pull request review event",
			event: GetMockPullRequestReviewEvent("submitted", "approved", MockRepoName, false, MockUserLogin, MockIssueAuthor),
			setup: func() {
				mockKvStore.EXPECT().Get(SubscriptionsKey, gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = GetMockSubscriptions()
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

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
				mockAPI.On("CreatePost", mock.Anything).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error webhook post", "post", mock.Anything, "error", "error creating post").Times(1)
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
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
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
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)

	tests := []struct {
		name  string
		event *github.IssueCommentEvent
		setup func()
	}{
		{
			name:  "unsupported action",
			event: GetMockIssueCommentEvent(actionEdited, "mockBody", "mockUser"),
			setup: func() {},
		},
		{
			name:  "commenter is the same as mentioned user",
			event: GetMockIssueCommentEvent(actionCreated, "mention @mockUser", "mockUser"),
			setup: func() {},
		},
		{
			name:  "comment mentions issue author",
			event: GetMockIssueCommentEvent(actionCreated, "mention @issueAuthor", "mockUser"),
			setup: func() {},
		},
		{
			name:  "error getting channel details",
			event: GetMockIssueCommentEvent(actionCreated, "mention @otherUser", "mockUser"),
			setup: func() {
				mockKvStore.EXPECT().Get("otherUser_githubusername", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "error getting channel details",
			event: GetMockIssueCommentEvent(actionCreated, "mention @otherUser", "mockUser"),
			setup: func() {
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
			setup: func() {
				mockKvStore.EXPECT().Get("otherUser_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("otherUserID")
					}
					return nil
				}).Times(1)
				mockKvStore.EXPECT().Get("otherUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("GetDirectChannel", "otherUserID", "mockBotID").Return(&model.Channel{Id: "mockChannelID"}, nil).Times(1)
				mockAPI.On("CreatePost", mock.Anything).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error creating mention post", "error", "error creating post").Times(1)
				mockAPI.On("LogWarn", "Failed to get github user info", "error", "Must connect user account to GitHub first.").Times(1)
			},
		},
		{
			name:  "successful mention notification",
			event: GetMockIssueCommentEvent(actionCreated, "mention @otherUser", "mockUser"),
			setup: func() {
				mockKvStore.EXPECT().Get("otherUser_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("otherUserID")
					}
					return nil
				}).Times(1)
				mockKvStore.EXPECT().Get("otherUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("GetDirectChannel", "otherUserID", "mockBotID").Return(&model.Channel{Id: "mockChannelID"}, nil).Times(1)
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
				mockAPI.On("LogWarn", "Failed to get github user info", "error", "Must connect user account to GitHub first.")
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			p.handleCommentMentionNotification(tc.event)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestHandleCommentAuthorNotification(t *testing.T) {
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)

	tests := []struct {
		name  string
		event *github.IssueCommentEvent
		setup func()
	}{
		{
			name:  "author is the commenter",
			event: GetMockIssueCommentEvent(actionCreated, "mockBody", "issueAuthor"),
			setup: func() {},
		},
		{
			name:  "unsupported action",
			event: GetMockIssueCommentEvent(actionEdited, "mockBody", "mockUser"),
			setup: func() {},
		},
		{
			name:  "author not mapped to user ID",
			event: GetMockIssueCommentEvent(actionCreated, "mockBody", "mockUser"),
			setup: func() {
				mockKvStore.EXPECT().Get("issueAuthor_githubusername", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "author has no permission to repo",
			event: GetMockIssueCommentEvent(actionCreated, "mockBody", "mockUser"),
			setup: func() {
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
			setup: func() {
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
			setup: func() {
				mockKvStore.EXPECT().Get("issueAuthor_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("authorUserID")
					}
					return nil
				}).Times(1)
				mockKvStore.EXPECT().Get("authorUserID-muted-users", gomock.Any()).Return(nil).Times(1)
				mockKvStore.EXPECT().Get("authorUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("GetDirectChannel", "authorUserID", "mockBotID").Return(&model.Channel{Id: "mockChannelID"}, nil).Times(1)
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Failed to get github user info", "error", "Must connect user account to GitHub first.").Times(1)
			},
		},
		{
			name:  "successful notification",
			event: GetMockIssueCommentEventWithURL(actionCreated, "mockBody", "mockUser", "https://mockurl.com/issues/123"),
			setup: func() {
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
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			p.handleCommentAuthorNotification(tc.event)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestHandleCommentAssigneeNotification(t *testing.T) {
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)

	tests := []struct {
		name  string
		event *github.IssueCommentEvent
		setup func()
	}{
		{
			name:  "unsupported issue type",
			event: GetMockIssueCommentEventWithAssignees("mockType", actionCreated, "mockBody", "mockUser", []string{"assigneeUser"}),
			setup: func() {
				mockAPI.On("LogDebug", "Unhandled issue type", "Type", "mockType")
			},
		},
		{
			name:  "assignee is the author",
			event: GetMockIssueCommentEventWithAssignees("issues", actionCreated, "mockBody", "assigneeUser", []string{"assigneeUser"}),
			setup: func() {
				mockKvStore.EXPECT().Get("assigneeUser_githubusername", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "issue author is assignee",
			event: GetMockIssueCommentEventWithAssignees("issues", actionCreated, "mockBody", "assigneeUser", []string{"issueAuthor"}),
			setup: func() {
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
			setup: func() {
				mockKvStore.EXPECT().Get("mockUser_githubusername", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "comment mentions assignee (self-mention)",
			event: GetMockIssueCommentEventWithAssignees("issues", actionCreated, "mention @assigneeUser", "mockUser", []string{"assigneeUser"}),
			setup: func() {
				mockKvStore.EXPECT().Get("assigneeUser_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("assigneeUserID")
					}
					return nil
				}).Times(1)
				mockKvStore.EXPECT().Get("assigneeUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				// mockAPI.On("LogDebug", "Commenter is muted, skipping notification")
				// mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
			},
		},
		{
			name:  "no permission to the repo",
			event: GetMockIssueCommentEventWithAssignees("issues", actionCreated, "mockBody", "mockUser", []string{"assigneeUser"}),
			setup: func() {
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
			tc.setup()

			p.handleCommentAssigneeNotification(tc.event)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestHandlePullRequestNotification(t *testing.T) {
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)

	tests := []struct {
		name  string
		event *github.PullRequestEvent
		setup func()
	}{
		{
			name:  "review requested by sender",
			event: GetMockPullRequestEvent("review_requested", "mockRepo", false, "senderUser", "senderUser", ""),
			setup: func() {},
		},
		{
			name:  "review requested with no repo permission",
			event: GetMockPullRequestEvent("review_requested", "mockRepo", true, "senderUser", "requestedReviewer", ""),
			setup: func() {
				mockKvStore.EXPECT().Get("requestedReviewer_githubusername", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "pull request closed by author",
			event: GetMockPullRequestEvent(actionClosed, "mockRepo", false, "authorUser", "authorUser", ""),
			setup: func() {},
		},
		{
			name:  "pull request closed successfully",
			event: GetMockPullRequestEvent(actionClosed, "mockRepo", false, "authorUser", "senderUser", ""),
			setup: func() {
				mockKvStore.EXPECT().Get("senderUser_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("authorUserID")
					}
					return nil
				}).Times(1)
				mockKvStore.EXPECT().Get("authorUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("GetDirectChannel", "authorUserID", "mockBotID").Return(&model.Channel{Id: "mockChannelID"}, nil)
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
				mockAPI.On("LogWarn", "Failed to get github user info", "error", "Must connect user account to GitHub first.").Times(1)
			},
		},
		{
			name:  "pull request reopened with no repo permission",
			event: GetMockPullRequestEvent(actionReopened, "mockRepo", true, "authorUser", "senderUser", ""),
			setup: func() {
				mockKvStore.EXPECT().Get("senderUser_githubusername", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "pull request assigned to self",
			event: GetMockPullRequestEvent(actionAssigned, "mockRepo", false, "assigneeUser", "assigneeUser", "assigneeUser"),
			setup: func() {},
		},
		{
			name:  "pull request assigned successfully",
			event: GetMockPullRequestEvent(actionAssigned, "mockRepo", false, "senderUser", "assigneeUser", "assigneeUser"),
			setup: func() {
				mockKvStore.EXPECT().Get("assigneeUser_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("assigneeUserID")
					}
					return nil
				}).Times(1)
				mockAPI.On("GetDirectChannel", "assigneeUserID", "mockBotID").Return(&model.Channel{Id: "mockChannelID"}, nil)
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
				mockKvStore.EXPECT().Get("assigneeUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("LogWarn", "Failed to get github user info", "error", "Must connect user account to GitHub first.").Times(1)
			},
		},
		{
			name:  "review requested with valid user ID",
			event: GetMockPullRequestEvent("review_requested", "mockRepo", false, "senderUser", "requestedReviewer", ""),
			setup: func() {
				mockKvStore.EXPECT().Get("requestedReviewer_githubusername", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(*[]byte); ok {
						*v = []byte("requestedUserID")
					}
					return nil
				}).Times(1)
				mockAPI.On("GetDirectChannel", "requestedUserID", "mockBotID").Return(&model.Channel{Id: "mockChannelID"}, nil)
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
				mockKvStore.EXPECT().Get("requestedUserID_githubtoken", gomock.Any()).Return(nil).Times(1)
				mockAPI.On("LogWarn", "Failed to get github user info", "error", "Must connect user account to GitHub first.").Times(1)
			},
		},
		{
			name: "unhandled event action",
			event: GetMockPullRequestEvent(
				"unsupported_action", "mockRepo", false, "senderUser", "", ""),
			setup: func() {
				mockAPI.On("LogDebug", "Unhandled event action", "action", "unsupported_action").Return(nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			p.handlePullRequestNotification(tc.event)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestHandleIssueNotification(t *testing.T) {
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)

	tests := []struct {
		name  string
		event *github.IssuesEvent
		setup func()
	}{
		{
			name:  "issue closed by author",
			event: GetMockIssuesEvent(actionClosed, MockRepo, false, "authorUser", "authorUser", ""),
			setup: func() {},
		},
		{
			name:  "issue closed successfully",
			event: GetMockIssuesEvent(actionClosed, MockRepo, true, "authorUser", "senderUser", ""),
			setup: func() {
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
			setup: func() {
				mockKvStore.EXPECT().Get("authorUser_githubusername", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "issue assigned to self",
			event: GetMockIssuesEvent(actionAssigned, MockRepo, false, "assigneeUser", "assigneeUser", "assigneeUser"),
			setup: func() {},
		},
		{
			name:  "issue assigned successfully",
			event: GetMockIssuesEvent(actionAssigned, MockRepo, false, "senderUser", "assigneeUser", "assigneeUser"),
			setup: func() {
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
			setup: func() {
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
			setup: func() {
				mockAPI.On("LogDebug", "Unhandled event action", "action", "unsupported_action").Return(nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			p.handleIssueNotification(tc.event)

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestHandlePullRequestReviewNotification(t *testing.T) {
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)

	tests := []struct {
		name  string
		event *github.PullRequestReviewEvent
		setup func()
	}{
		{
			name:  "review submitted by author",
			event: GetMockPullRequestReviewEvent(actionSubmitted, "approved", MockRepo, false, "authorUser", "authorUser"),
			setup: func() {},
		},
		{
			name:  "review action not submitted",
			event: GetMockPullRequestReviewEvent("dismissed", "approved", MockRepo, false, "authorUser", "reviewerUser"),
			setup: func() {},
		},
		{
			name:  "review with author not mapped to user ID",
			event: GetMockPullRequestReviewEvent(actionSubmitted, "approved", MockRepo, false, "unknownAuthor", "reviewerUser"),
			setup: func() {
				mockKvStore.EXPECT().Get("reviewerUser_githubusername", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "private repo, no permission for author",
			event: GetMockPullRequestReviewEvent(actionSubmitted, "approved", MockRepo, true, "authorUser", "reviewerUser"),
			setup: func() {
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
			setup: func() {
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
			tc.setup()

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
						*v = &Subscriptions{
							Repositories: map[string][]*Subscription{
								"mockrepo/mockorg": {
									{
										ChannelID:  MockChannelID,
										CreatorID:  MockCreatorID,
										Features:   featureStars,
										Repository: MockRepo,
									},
									{
										ChannelID:  MockChannelID,
										CreatorID:  MockCreatorID,
										Features:   featureDeletes,
										Repository: MockRepo,
									},
								},
							},
						}
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.Anything).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error webhook post", "post", mock.Anything, "error", "error creating post")
			},
		},
		{
			name:  "successful star event notification",
			event: GetMockStarEvent(MockRepo, MockOrg, false, MockSender),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = &Subscriptions{
							Repositories: map[string][]*Subscription{
								"mockrepo/mockorg": {
									{
										ChannelID:  MockChannelID,
										CreatorID:  MockCreatorID,
										Features:   featureStars,
										Repository: MockRepo,
									},
									{
										ChannelID:  MockChannelID,
										CreatorID:  MockCreatorID,
										Features:   featureDeletes,
										Repository: MockRepo,
									},
								},
							},
						}
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
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
	mockKvStore, mockAPI, _, _, _ := GetTestSetup(t)
	p := getPluginTest(mockAPI, mockKvStore)

	tests := []struct {
		name  string
		event *github.ReleaseEvent
		setup func()
	}{
		{
			name:  "no subscribed channels for repository",
			event: GetMockReleaseEvent(MockRepo, MockOrg, "created", MockSender),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:  "unsupported action",
			event: GetMockReleaseEvent(MockRepo, MockOrg, "edited", MockSender),
			setup: func() {},
		},
		{
			name:  "error creating post",
			event: GetMockReleaseEvent(MockRepo, MockOrg, "created", MockSender),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = &Subscriptions{
							Repositories: map[string][]*Subscription{
								"mockrepo/mockorg": {
									{
										ChannelID:  MockChannelID,
										CreatorID:  MockCreatorID,
										Features:   featureReleases,
										Repository: MockRepo,
									},
									{
										ChannelID:  MockChannelID,
										CreatorID:  MockCreatorID,
										Features:   featureDeletes,
										Repository: MockRepo,
									},
								},
							},
						}
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.Anything).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error webhook post", "Post", mock.Anything, "Error", "error creating post")
			},
		},
		{
			name:  "successful release event notification",
			event: GetMockReleaseEvent(MockRepo, MockOrg, "created", MockSender),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = &Subscriptions{
							Repositories: map[string][]*Subscription{
								"mockrepo/mockorg": {
									{
										ChannelID:  MockChannelID,
										CreatorID:  MockCreatorID,
										Features:   featureReleases,
										Repository: MockRepo,
									},
									{
										ChannelID:  MockChannelID,
										CreatorID:  MockCreatorID,
										Features:   featureDeletes,
										Repository: MockRepo,
									},
								},
							},
						}
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

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
						*v = &Subscriptions{
							Repositories: map[string][]*Subscription{
								"mockrepo/mockorg": {
									{
										ChannelID:  MockChannelID,
										CreatorID:  MockCreatorID,
										Features:   featureDiscussions,
										Repository: MockRepo,
									},
									{
										ChannelID:  MockChannelID,
										CreatorID:  MockCreatorID,
										Features:   featureDeletes,
										Repository: MockRepo,
									},
								},
							},
						}
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.Anything).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error creating discussion notification post", "Post", mock.Anything, "Error", "error creating post")
			},
		},
		{
			name:  "successful discussion notification",
			event: GetMockDiscussionEvent(MockRepo, MockOrg, MockSender),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = &Subscriptions{
							Repositories: map[string][]*Subscription{
								"mockrepo/mockorg": {
									{
										ChannelID:  MockChannelID,
										CreatorID:  MockCreatorID,
										Features:   featureDiscussions,
										Repository: MockRepo,
									},
								},
							},
						}
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
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
			name:  "unsupported action",
			event: GetMockDiscussionCommentEvent(MockRepo, MockOrg, "edited", MockSender),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = &Subscriptions{
							Repositories: map[string][]*Subscription{
								"mockrepo/mockorg": {
									{
										ChannelID:  MockChannelID,
										CreatorID:  MockCreatorID,
										Features:   featureDiscussionComments,
										Repository: MockRepo,
									},
								},
							},
						}
					}
					return nil
				}).Times(1)
			},
		},
		{
			name:  "error creating discussion comment post",
			event: GetMockDiscussionCommentEvent(MockRepo, MockOrg, "created", MockSender),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = &Subscriptions{
							Repositories: map[string][]*Subscription{
								"mockrepo/mockorg": {
									{
										ChannelID:  MockChannelID,
										CreatorID:  MockCreatorID,
										Features:   featureDiscussionComments,
										Repository: MockRepo,
									}, {
										ChannelID:  MockChannelID,
										CreatorID:  MockCreatorID,
										Features:   featureDeletes,
										Repository: MockRepo,
									},
								},
							},
						}
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.Anything).Return(nil, &model.AppError{Message: "error creating post"}).Times(1)
				mockAPI.On("LogWarn", "Error creating discussion comment post", "Post", mock.Anything, "Error", "error creating post")
			},
		},
		{
			name:  "successful discussion comment notification",
			event: GetMockDiscussionCommentEvent(MockRepo, MockOrg, "created", MockSender),
			setup: func() {
				mockKvStore.EXPECT().Get("subscriptions", gomock.Any()).DoAndReturn(func(key string, value interface{}) error {
					if v, ok := value.(**Subscriptions); ok {
						*v = &Subscriptions{
							Repositories: map[string][]*Subscription{
								"mockrepo/mockorg": {
									{
										ChannelID:  MockChannelID,
										CreatorID:  MockCreatorID,
										Features:   featureDiscussionComments,
										Repository: MockRepo,
									},
								},
							},
						}
					}
					return nil
				}).Times(1)
				mockAPI.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			p.postDiscussionCommentEvent(tc.event)

			mockAPI.AssertExpectations(t)
		})
	}
}
