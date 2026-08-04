// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"fmt"
	"sync"
	"testing"

	"github.com/mattermost/mattermost/server/public/pluginapi"
)

// TestSubscriptionRace drives the plugin's real AddSubscription against the
// vendored pluginapi.MemoryStore, whose SetAtomicWithRetries is byte-identical
// to the production KVService. No plugin logic is mocked. Many channels
// subscribe to the same repository concurrently; afterwards we compare how many
// AddSubscription calls reported success (nil error) against how many
// subscriptions actually persisted.
//
// Correct behaviour: persisted == reported-success (every reported success is
// durable). Before the fix, the read-modify-write happened outside the atomic
// callback, so concurrent writers clobbered each other and persisted was far
// lower than reported-success (silent lost updates).
func TestSubscriptionRace(t *testing.T) {
	const numChannels = 200

	p := NewPlugin()
	p.client = pluginapi.NewClient(p.API, p.Driver)
	p.store = &pluginapi.MemoryStore{}

	var wg sync.WaitGroup
	var mu sync.Mutex
	reportedSuccess := 0

	for i := 0; i < numChannels; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sub := &Subscription{
				ChannelID:  fmt.Sprintf("channel-%d", i),
				Repository: "org/repo",
			}
			if err := p.AddSubscription("org/repo", sub); err == nil {
				mu.Lock()
				reportedSuccess++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()

	subs, err := p.GetSubscriptions()
	if err != nil {
		t.Fatalf("GetSubscriptions failed: %v", err)
	}
	persisted := 0
	for _, chans := range subs.Repositories {
		persisted += len(chans)
	}

	silentlyLost := reportedSuccess - persisted
	t.Logf("concurrent AddSubscription calls : %d", numChannels)
	t.Logf("AddSubscription returned success : %d", reportedSuccess)
	t.Logf("subscriptions actually persisted : %d", persisted)
	t.Logf("silently lost (success but gone) : %d", silentlyLost)

	if silentlyLost > 0 {
		t.Fatalf("lost-update bug: %d subscriptions were reported as saved but silently dropped", silentlyLost)
	}

	// Guard against a false pass: if every AddSubscription call failed,
	// reportedSuccess and persisted are both 0, so silentlyLost is 0 and the
	// check above passes without actually exercising the race.
	if reportedSuccess == 0 {
		t.Fatalf("no AddSubscription calls succeeded; the race was not exercised")
	}
}
