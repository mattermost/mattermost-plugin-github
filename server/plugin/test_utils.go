package plugin

import (
	"context"
	"crypto/hmac"
	"crypto/sha1" // #nosec G505
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-github/v54/github"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-github/server/mocks"

	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
)

const (
	MockUserID          = "mockUserID"
	MockUsername        = "mockUsername"
	MockAccessToken     = "mockAccessToken"
	MockChannelID       = "mockChannelID"
	MockCreatorID       = "mockCreatorID"
	MockWebhookSecret   = "mockWebhookSecret" // #nosec G101
	MockBotID           = "mockBotID"
	MockOrg             = "mockOrg"
	MockSender          = "mockSender"
	MockPostMessage     = "mockPostMessage"
	MockOrgRepo         = "mockOrg/mockRepo"
	MockHead            = "mockHead"
	MockPRTitle         = "mockPRTitle"
	MockProfileUsername = "@username"
	MockPostID          = "mockPostID"
	MockRepoName        = "mockRepoName"
	MockEventReference  = "refs/heads/main"
	MockUserLogin       = "mockUser"
	MockBranch          = "mockBranch"
	MockRepo            = "mockRepo"
	MockLabel           = "mockLabel"
	MockValidLabel      = "validLabel"
	MockIssueAuthor     = "issueAuthor"
	GithubBaseURL       = "https://github.com/"
)

type GitHubUserResponse struct {
	Username string `json:"username"`
}

func GetMockGHUserInfo(p *Plugin) (*GitHubUserInfo, error) {
	encryptionKey := "dummyEncryptKey1"
	p.setConfiguration(&Configuration{EncryptionKey: encryptionKey})
	encryptedToken, err := encrypt([]byte(encryptionKey), MockAccessToken)
	if err != nil {
		return nil, err
	}
	gitHubUserInfo := &GitHubUserInfo{
		UserID:         MockUserID,
		GitHubUsername: MockUsername,
		Token:          &oauth2.Token{AccessToken: encryptedToken},
		Settings:       &UserSettings{},
	}

	return gitHubUserInfo, nil
}

func GetTestSetup(t *testing.T) (*mocks.MockKvStore, *plugintest.API, *mocks.MockLogger, *mocks.MockLogger, *Context) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockKvStore := mocks.NewMockKvStore(mockCtrl)
	mockAPI := &plugintest.API{}
	mockLogger := mocks.NewMockLogger(mockCtrl)
	mockLoggerWith := mocks.NewMockLogger(mockCtrl)
	mockContext := GetMockContext(mockLogger)

	return mockKvStore, mockAPI, mockLogger, mockLoggerWith, &mockContext
}

func GetMockContext(mockLogger *mocks.MockLogger) Context {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	return Context{
		Ctx:    ctx,
		UserID: MockUserID,
		Log:    mockLogger,
	}
}

func GetMockUserContext(p *Plugin, mockLogger *mocks.MockLogger) (*UserContext, error) {
	mockGHUserInfo, err := GetMockGHUserInfo(p)
	if err != nil {
		return nil, err
	}

	mockUserContext := &UserContext{
		GetMockContext(mockLogger),
		mockGHUserInfo,
	}

	return mockUserContext, nil
}

func generateSignature(secret, body []byte) string {
	h := hmac.New(sha1.New, secret)
	h.Write(body)
	return "sha1=" + hex.EncodeToString(h.Sum(nil))
}

func GetMockPingEvent() *github.PingEvent {
	return &github.PingEvent{
		Zen:    github.String("Keep it logically awesome."),
		HookID: github.Int64(123456),
		Hook: &github.Hook{
			Type: github.String("Repository"),
			ID:   github.Int64(654321),
			Config: map[string]interface{}{
				"url":          "https://example.com/webhook",
				"content_type": "json",
				"secret":       "mocksecret",
				"insecure_ssl": "0",
			},
			Active: github.Bool(true),
		},
		Repo: &github.Repository{
			Name:     github.String(MockRepoName),
			FullName: github.String(MockOrgRepo),
			Private:  github.Bool(false),
			HTMLURL:  github.String(fmt.Sprintf("%s/%s", GithubBaseURL, MockOrgRepo)),
		},
		Org: &github.Organization{
			Login: github.String("mockorg"),
			ID:    github.Int64(12345),
			URL:   github.String(fmt.Sprintf("%s/mockorg", GithubBaseURL)),
		},
		Sender: &github.User{
			Login: github.String(MockUserLogin),
			ID:    github.Int64(98765),
			URL:   github.String(fmt.Sprintf("%s/users/%s", GithubBaseURL, MockUserLogin)),
		},
		Installation: &github.Installation{
			ID:     github.Int64(246810),
			NodeID: github.String("MDQ6VXNlcjE="),
		},
	}
}

func GetMockPRDescriptionEvent(repo, org, sender, prUser, action, label string) *github.PullRequestEvent {
	return &github.PullRequestEvent{
		Action: github.String(action),
		PullRequest: &github.PullRequest{
			Title:   github.String(MockPRTitle),
			Body:    github.String("Mock PR description with label: " + label),
			State:   github.String("open"),
			User:    &github.User{Login: github.String(prUser)},
			Head:    &github.PullRequestBranch{Ref: github.String(MockBranch)},
			Base:    &github.PullRequestBranch{Ref: github.String("main")},
			HTMLURL: github.String(GithubBaseURL + org + "/" + repo + "/pull/1"),
			Number:  github.Int(1),
		},
		Repo: &github.Repository{
			Name:     github.String(repo),
			Owner:    &github.User{Login: github.String(org)},
			FullName: github.String(org + "/" + repo),
		},
		Sender: &github.User{
			Login: github.String(sender),
		},
	}
}

func GetMockIssueEvent(repo, org, sender, action, label string) *github.IssuesEvent {
	event := &github.IssuesEvent{
		Repo: &github.Repository{
			Name:     github.String(repo),
			Owner:    &github.User{Login: github.String(org)},
			FullName: github.String(fmt.Sprintf("%s/%s", repo, org)),
		},
		Sender: &github.User{Login: github.String(sender)},
		Issue: &github.Issue{
			Number: github.Int(123),
			Labels: []*github.Label{
				{Name: github.String(label)},
			},
		},
		Action: github.String(action),
	}

	if action == actionLabeled || action == "unlabeled" {
		event.Label = &github.Label{Name: github.String(label)}
	}

	return event
}

func GetMockIssueEventWithTimeDiff(repo, org, sender, action, label string, timeDiff time.Duration) *github.IssuesEvent {
	event := GetMockIssueEvent(repo, org, sender, action, label)
	event.Issue.CreatedAt = &github.Timestamp{Time: time.Now().Add(timeDiff)}
	return event
}

func GetMockPushEvent() *github.PushEvent {
	return &github.PushEvent{
		PushID: github.Int64(1),
		Head:   github.String(MockHead),
		Repo: &github.PushEventRepository{
			Name:     github.String(MockRepoName),
			FullName: github.String(MockOrgRepo),
			Private:  github.Bool(false),
			HTMLURL:  github.String(fmt.Sprintf("%s/%s", GithubBaseURL, MockOrgRepo)),
		},
		Ref:     github.String(MockEventReference),
		Compare: github.String("%s%s/compare/old...new"),
		Sender: &github.User{
			Login: github.String(MockUserLogin),
		},
		Commits: []*github.HeadCommit{
			{
				ID:      github.String("abcdef123456"),
				URL:     github.String(fmt.Sprintf("%s%s/commit/abcdef123456", GithubBaseURL, MockOrgRepo)),
				Message: github.String("Initial commit"),
				Author: &github.CommitAuthor{
					Name: github.String("John Doe"),
				},
			},
			{
				ID:      github.String("123456abcdef"),
				URL:     github.String(fmt.Sprintf("%s%s/commit/123456abcdef", GithubBaseURL, MockOrgRepo)),
				Message: github.String("Update README"),
				Author: &github.CommitAuthor{
					Name: github.String("Jane Smith"),
				},
			},
		},
	}
}

func GetMockPushEventWithoutCommit() *github.PushEvent {
	return &github.PushEvent{
		PushID: github.Int64(1),
		Head:   github.String(MockHead),
		Repo: &github.PushEventRepository{
			Name:     github.String(MockRepoName),
			FullName: github.String(MockOrgRepo),
			Private:  github.Bool(false),
			HTMLURL:  github.String(fmt.Sprintf("%s%s", GithubBaseURL, MockOrgRepo)),
		},
		Ref:     github.String(MockEventReference),
		Compare: github.String(fmt.Sprintf("%s%s/compare/old...new", GithubBaseURL, MockOrgRepo)),
		Sender: &github.User{
			Login: github.String(MockUserLogin),
		},
	}
}

func GetMockSubscriptions() *Subscriptions {
	return &Subscriptions{
		Repositories: map[string][]*Subscription{
			"mockorg/mockrepo": {
				{
					ChannelID:  "channel1",
					CreatorID:  "user1",
					Features:   Features("pushes"),
					Flags:      SubscriptionFlags{},
					Repository: MockOrgRepo,
				},
				{
					ChannelID:  "channel2",
					CreatorID:  "user2",
					Features:   Features("creates"),
					Flags:      SubscriptionFlags{},
					Repository: MockOrgRepo,
				},
				{
					ChannelID:  "channel2",
					CreatorID:  "user3",
					Features:   Features("deletes"),
					Flags:      SubscriptionFlags{},
					Repository: MockOrgRepo,
				},
				{
					ChannelID:  "channel4",
					CreatorID:  "user4",
					Features:   Features("issue_comments"),
					Flags:      SubscriptionFlags{},
					Repository: MockOrgRepo,
				},
				{
					ChannelID:  "channel5",
					CreatorID:  "user5",
					Features:   Features("pull_reviews"),
					Flags:      SubscriptionFlags{},
					Repository: MockOrgRepo,
				},
			},
		},
	}
}

func GetMockCreateEvent() *github.CreateEvent {
	return &github.CreateEvent{
		Ref:     github.String("v1.0.0"),
		RefType: github.String("tag"),
		Repo: &github.Repository{
			Name:     github.String(MockRepoName),
			FullName: github.String(MockOrgRepo),
			Private:  github.Bool(false),
			HTMLURL:  github.String(fmt.Sprintf("%s%s", GithubBaseURL, MockOrgRepo)),
		},
		Sender: &github.User{
			Login: github.String(MockUserLogin),
		},
	}
}

func GetMockCreateEventWithUnsupportedRefType() *github.CreateEvent {
	return &github.CreateEvent{
		Ref:     github.String("feature/new-feature"),
		RefType: github.String("unsupported"),
		Repo: &github.Repository{
			Name:     github.String(MockRepoName),
			FullName: github.String(MockOrgRepo),
			Private:  github.Bool(false),
			HTMLURL:  github.String(fmt.Sprintf("%s%s", GithubBaseURL, MockOrgRepo)),
		},
		Sender: &github.User{
			Login: github.String(MockUserLogin),
		},
	}
}

func GetMockDeleteEvent() *github.DeleteEvent {
	return &github.DeleteEvent{
		Ref:     github.String(MockBranch),
		RefType: github.String("branch"),
		Repo: &github.Repository{
			Name:     github.String(MockRepoName),
			FullName: github.String(MockOrgRepo),
			Private:  github.Bool(false),
			HTMLURL:  github.String(fmt.Sprintf("%s%s", GithubBaseURL, MockOrgRepo)),
		},
		Sender: &github.User{
			Login: github.String(MockUserLogin),
		},
	}
}

func GetMockDeleteEventWithInvalidType() *github.DeleteEvent {
	return &github.DeleteEvent{
		Ref:     github.String(MockBranch),
		RefType: github.String("invalidType"),
		Repo: &github.Repository{
			Name:     github.String(MockRepoName),
			FullName: github.String(MockOrgRepo),
			Private:  github.Bool(false),
			HTMLURL:  github.String(fmt.Sprintf("%s%s", GithubBaseURL, MockOrgRepo)),
		},
		Sender: &github.User{
			Login: github.String(MockUserLogin),
		},
	}
}

func GetMockPullRequestReviewEvent(action, state, repo string, isPrivate bool, reviewer, author string) *github.PullRequestReviewEvent {
	return &github.PullRequestReviewEvent{
		Action: github.String(action),
		Repo: &github.Repository{
			Name:     github.String(repo),
			FullName: github.String(MockOrgRepo),
			Private:  github.Bool(isPrivate),
			HTMLURL:  github.String(fmt.Sprintf("%s%s", GithubBaseURL, MockOrgRepo)),
		},
		Sender: &github.User{Login: github.String(reviewer)},
		Review: &github.PullRequestReview{
			User: &github.User{
				Login: github.String(reviewer),
			},
			State: github.String(state),
		},
		PullRequest: &github.PullRequest{
			User: &github.User{Login: github.String(author)},
		},
	}
}

func GetMockPullRequestReviewCommentEvent() *github.PullRequestReviewCommentEvent {
	return &github.PullRequestReviewCommentEvent{
		Repo: &github.Repository{
			Name:     github.String(MockRepoName),
			FullName: github.String(MockOrgRepo),
			Private:  github.Bool(false),
			HTMLURL:  github.String(fmt.Sprintf("%s%s", GithubBaseURL, MockOrgRepo)),
		},
		Comment: &github.PullRequestComment{
			ID:      github.Int64(12345),
			Body:    github.String("This is a review comment"),
			HTMLURL: github.String(fmt.Sprintf("%s%s/pull/1#discussion_r12345", GithubBaseURL, MockOrgRepo)),
		},
		Sender: &github.User{
			Login: github.String(MockUserLogin),
		},
		PullRequest: &github.PullRequest{},
	}
}

func GetMockIssueCommentEvent(action, body, sender string) *github.IssueCommentEvent {
	return &github.IssueCommentEvent{
		Action: github.String(action),
		Repo: &github.Repository{
			Name:     github.String(MockRepo),
			FullName: github.String(MockOrgRepo),
			Private:  github.Bool(false),
		},
		Comment: &github.IssueComment{
			Body: github.String(body),
		},
		Issue: &github.Issue{
			User:      &github.User{Login: github.String(MockIssueAuthor)},
			Assignees: []*github.User{{Login: github.String("assigneeUser")}},
		},
		Sender: &github.User{
			Login: github.String(sender),
		},
	}
}

func GetMockIssueCommentEventWithURL(action, body, sender, url string) *github.IssueCommentEvent {
	event := GetMockIssueCommentEvent(action, body, sender)
	event.Issue.HTMLURL = github.String(url)
	return event
}

func GetMockIssueCommentEventWithAssignees(eventType, action, body, sender string, assignees []string) *github.IssueCommentEvent {
	assigneeUsers := make([]*github.User, len(assignees))
	for i, assignee := range assignees {
		assigneeUsers[i] = &github.User{Login: github.String(assignee)}
	}

	return &github.IssueCommentEvent{
		Action: github.String(action),
		Repo: &github.Repository{
			Name:     github.String(MockRepo),
			FullName: github.String(MockOrgRepo),
			Private:  github.Bool(false),
		},
		Comment: &github.IssueComment{
			Body: github.String(body),
		},
		Issue: &github.Issue{
			User:      &github.User{Login: github.String(MockIssueAuthor)},
			Assignees: assigneeUsers,
			HTMLURL:   github.String(fmt.Sprintf("%s%s/%s/123", GithubBaseURL, MockOrgRepo, eventType)),
		},
		Sender: &github.User{
			Login: github.String(sender),
		},
	}
}

func GetMockPullRequestEvent(action, repoName, eventLabel string, isPrivate bool, sender, user, assignee string) *github.PullRequestEvent {
	return &github.PullRequestEvent{
		Action: github.String(action),
		Label:  &github.Label{Name: github.String(eventLabel)},
		Repo: &github.Repository{
			Name:     github.String(repoName),
			FullName: github.String(fmt.Sprintf("mockOrg/%s", repoName)),
			Private:  github.Bool(isPrivate),
		},
		PullRequest: &github.PullRequest{
			User:               &github.User{Login: github.String(user)},
			HTMLURL:            github.String(fmt.Sprintf("%s%s/%s/pull/123", GithubBaseURL, MockOrgRepo, repoName)),
			Assignee:           &github.User{Login: github.String(assignee)},
			RequestedReviewers: []*github.User{{Login: github.String(user)}},
			Labels:             []*github.Label{{Name: github.String("validLabel")}},
			Draft:              github.Bool(true),
		},
		Sender: &github.User{
			Login: github.String(sender),
		},
		RequestedReviewer: &github.User{Login: github.String(user)},
	}
}

func GetMockIssuesEvent(action, repoName string, isPrivate bool, author, sender, assignee string) *github.IssuesEvent {
	return &github.IssuesEvent{
		Action: &action,
		Repo:   &github.Repository{FullName: &repoName, Private: &isPrivate},
		Issue:  &github.Issue{User: &github.User{Login: &author}},
		Sender: &github.User{Login: &sender},
		Assignee: func() *github.User {
			if assignee == "" {
				return nil
			}
			return &github.User{Login: &assignee}
		}(),
	}
}

func GetMockStarEvent(repo, org string, isPrivate bool, sender string) *github.StarEvent {
	return &github.StarEvent{
		Repo: &github.Repository{
			Name:     github.String(repo),
			Private:  github.Bool(isPrivate),
			FullName: github.String(fmt.Sprintf("%s/%s", repo, org)),
		},
		Sender: &github.User{Login: github.String(sender)},
	}
}

func GetMockReleaseEvent(repo, org, action, sender string) *github.ReleaseEvent {
	return &github.ReleaseEvent{
		Action: &action,
		Repo: &github.Repository{
			Name:     github.String(repo),
			Owner:    &github.User{Login: github.String(org)},
			FullName: github.String(fmt.Sprintf("%s/%s", repo, org)),
		},
		Sender: &github.User{Login: github.String(sender)},
	}
}

func GetMockDiscussionEvent(repo, org, sender string) *github.DiscussionEvent {
	return &github.DiscussionEvent{
		Repo: &github.Repository{
			Name:     github.String(repo),
			Owner:    &github.User{Login: github.String(org)},
			FullName: github.String(fmt.Sprintf("%s/%s", repo, org)),
		},
		Sender: &github.User{Login: github.String(sender)},
		Discussion: &github.Discussion{
			Number: github.Int(123),
		},
	}
}

func GetMockDiscussionCommentEvent(repo, org, action, sender string) *github.DiscussionCommentEvent {
	return &github.DiscussionCommentEvent{
		Action: &action,
		Repo: &github.Repository{
			Name:     github.String(repo),
			Owner:    &github.User{Login: github.String(org)},
			FullName: github.String(fmt.Sprintf("%s/%s", repo, org)),
		},
		Sender: &github.User{Login: github.String(sender)},
		Comment: &github.CommentDiscussion{
			ID: github.Int64(456),
		},
	}
}
