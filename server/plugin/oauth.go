package plugin

import (
	"sync"
)

type OAuthCompleteEvent struct {
	UserID string
	Err    error
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
			close(sub)
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

	for _, userSubs := range ob.oauthCompleteSubs {
		for _, sub := range userSubs {
			// non-blocking send
			select {
			case sub <- err:
			default:
			}
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
