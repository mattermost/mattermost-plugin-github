package plugin

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-github/server/mocks"

	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
)

const (
	MockUserID      = "mockUserID"
	MockUsername    = "mockUsername"
	MockAccessToken = "mockAccessToken"
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
