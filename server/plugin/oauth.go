// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"sync"

	"golang.org/x/oauth2"
)

type GitHubUserRequest struct {
	UserID string `json:"user_id"`
}

type GitHubUserResponse struct {
	Username string `json:"username"`
}

type ConnectedResponse struct {
	Connected           bool                   `json:"connected"`
	GitHubUsername      string                 `json:"github_username"`
	GitHubClientID      string                 `json:"github_client_id"`
	EnterpriseBaseURL   string                 `json:"enterprise_base_url,omitempty"`
	Organizations       []string               `json:"organizations"`
	UserSettings        *UserSettings          `json:"user_settings"`
	ClientConfiguration map[string]interface{} `json:"configuration"`
}

type UserSettings struct {
	SidebarButtons        string `json:"sidebar_buttons"`
	DailyReminder         bool   `json:"daily_reminder"`
	DailyReminderOnChange bool   `json:"daily_reminder_on_change"`
	Notifications         bool   `json:"notifications"`
}

type GitHubUserInfo struct {
	UserID              string
	Token               *oauth2.Token
	GitHubUsername      string
	LastToDoPostAt      int64
	Settings            *UserSettings
	AllowedPrivateRepos bool

	// MM34646ResetTokenDone is set for a user whose token has been reset for MM-34646.
	MM34646ResetTokenDone bool
}

type OAuthCompleteEvent struct {
	UserID string
	Err    error
}

type OAuthState struct {
	UserID         string `json:"user_id"`
	Token          string `json:"token"`
	PrivateAllowed bool   `json:"private_allowed"`
}

type OAuthBroker struct {
	sendOAuthCompleteEvent func(event OAuthCompleteEvent)

	lock              sync.RWMutex // Protects closed and pingSubs
	closed            bool
	oauthCompleteSubs map[string][]chan error
	mapCreate         sync.Once
}

func NewOAuthBroker(sendOAuthCompleteEvent func(event OAuthCompleteEvent)) *OAuthBroker {
	return &OAuthBroker{
		sendOAuthCompleteEvent: sendOAuthCompleteEvent,
	}
}

func (ob *OAuthBroker) SubscribeOAuthComplete(userID string) <-chan error {
	ob.lock.Lock()
	defer ob.lock.Unlock()

	ob.mapCreate.Do(func() {
		ob.oauthCompleteSubs = make(map[string][]chan error)
	})

	ch := make(chan error, 1)
	ob.oauthCompleteSubs[userID] = append(ob.oauthCompleteSubs[userID], ch)

	return ch
}

func (ob *OAuthBroker) UnsubscribeOAuthComplete(userID string, ch <-chan error) {
	ob.lock.Lock()
	defer ob.lock.Unlock()

	for i, sub := range ob.oauthCompleteSubs[userID] {
		if sub == ch {
			ob.oauthCompleteSubs[userID] = append(ob.oauthCompleteSubs[userID][:i], ob.oauthCompleteSubs[userID][i+1:]...)
			break
		}
	}
}

func (ob *OAuthBroker) publishOAuthComplete(userID string, err error, fromCluster bool) {
	ob.lock.Lock()
	defer ob.lock.Unlock()

	if ob.closed {
		return
	}

	for _, userSub := range ob.oauthCompleteSubs[userID] {
		// non-blocking send
		select {
		case userSub <- err:
		default:
		}
	}

	if !fromCluster {
		ob.sendOAuthCompleteEvent(OAuthCompleteEvent{UserID: userID, Err: err})
	}
}

func (ob *OAuthBroker) Close() {
	ob.lock.Lock()
	defer ob.lock.Unlock()

	if !ob.closed {
		ob.closed = true

		for _, userSubs := range ob.oauthCompleteSubs {
			for _, sub := range userSubs {
				close(sub)
			}
		}
	}
}
