// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-github/v54/github"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/mattermost/mattermost-plugin-github/server/mocks"
)

func setupSAMLSSOTest(t *testing.T) (*Plugin, *plugintest.API, *mocks.MockKvStore, *gomock.Controller) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockKvStore := mocks.NewMockKvStore(ctrl)

	api := &plugintest.API{}
	p := NewPlugin()
	p.store = mockKvStore
	p.BotUserID = MockBotID
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)

	return p, api, mockKvStore, ctrl
}

func TestNotifySAMLSSORequired_SendsDMOnce(t *testing.T) {
	p, api, mockKvStore, ctrl := setupSAMLSSOTest(t)
	defer ctrl.Finish()

	userID := "user1"
	key := userID + samlSSONotifiedKey

	mockKvStore.EXPECT().Get(key, gomock.Any()).Return(nil)
	mockKvStore.EXPECT().Set(key, true).Return(true, nil)

	api.On("GetDirectChannel", userID, MockBotID).Return(&model.Channel{Id: "dm-channel"}, nil)
	api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
		return post.ChannelId == "dm-channel" && post.Message == samlSSOUserMessage && post.Type == "custom_git_saml_sso"
	})).Return(&model.Post{}, nil)

	p.notifySAMLSSORequired(userID)

	mockKvStore.EXPECT().Get(key, gomock.Any()).DoAndReturn(func(_ string, out any) error {
		ptr, ok := out.(*bool)
		require.True(t, ok)
		*ptr = true
		return nil
	})

	p.notifySAMLSSORequired(userID)

	api.AssertExpectations(t)
}

func TestWriteSAMLSSOErrorIfNeeded(t *testing.T) {
	p, api, mockKvStore, ctrl := setupSAMLSSOTest(t)
	defer ctrl.Finish()

	userID := "user1"
	key := userID + samlSSONotifiedKey
	c := &UserContext{Context: Context{UserID: userID}}

	mockKvStore.EXPECT().Get(key, gomock.Any()).Return(nil)
	mockKvStore.EXPECT().Set(key, true).Return(true, nil)
	api.On("GetDirectChannel", userID, MockBotID).Return(&model.Channel{Id: "dm-channel"}, nil)
	api.On("CreatePost", mock.Anything).Return(&model.Post{}, nil)

	recorder := &responseRecorder{}
	samlErr := &github.ErrorResponse{
		Response: &http.Response{StatusCode: http.StatusForbidden},
		Message:  "Resource protected by organization SAML enforcement. You must grant your personal token access to this organization.",
	}

	require.True(t, p.writeSAMLSSOErrorIfNeeded(c, recorder, samlErr))
	require.Equal(t, http.StatusForbidden, recorder.statusCode)
	require.Contains(t, recorder.body, apiErrorIDSAMLSSORequired)
	require.Contains(t, recorder.body, samlSSOUserMessage)

	require.False(t, p.writeSAMLSSOErrorIfNeeded(c, recorder, errors.New("unrelated error")))
}

type responseRecorder struct {
	statusCode int
	body       string
}

func (r *responseRecorder) Header() http.Header {
	return http.Header{}
}

func (r *responseRecorder) Write(body []byte) (int, error) {
	r.body += string(body)
	return len(body), nil
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}
