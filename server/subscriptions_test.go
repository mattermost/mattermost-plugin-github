package main

import (
	"encoding/json"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"sort"
	"testing"
)

// pluginFromRepoNames returns mocked plugin for given repository names
func pluginFromRepoNames(repoNames []string, chanelID string) *Plugin {
	p := NewPlugin()
	mockPluginAPI := &plugintest.API{}

	a := Subscriptions{Repositories: map[string][]*Subscription{}}

	a.Repositories[""] = []*Subscription{}
	for _, st := range repoNames {
		a.Repositories[""] = append(a.Repositories[""], &Subscription{
			Repository: st,
			ChannelID:  chanelID,
		})
	}
	jsn, _ := json.Marshal(a)
	mockPluginAPI.On("KVGet", SUBSCRIPTIONS_KEY).Return(jsn, nil)
	p.SetAPI(mockPluginAPI)
	return p
}

// wantedSubscriptions returns what should be returned after sorting by repo names
func wantedSubscriptions(repoNames []string, chanelID string) []*Subscription {
	sort.Strings(repoNames)
	var subs []*Subscription
	for _, st := range repoNames {
		subs = append(subs, &Subscription{
			ChannelID:  chanelID,
			Repository: st,
		})
	}
	return subs
}

// compareSubsByRepoName checks if two subscription list is identical by repository names
func compareSubsByRepoName(sub1, sub2 []*Subscription) bool {
	if len(sub1) != len(sub2) {
		return false
	}
	for i := 0; i < len(sub1); i++ {
		if sub1[i].Repository != sub2[i].Repository {
			return false
		}
	}
	return true
}

func TestPlugin_GetSubscriptionsByChannel(t *testing.T) {

	type args struct {
		channelID string
	}
	tests := []struct {
		name    string
		plugin  *Plugin
		args    args
		want    []*Subscription
		wantErr bool
	}{
		{
			name:    "basic test",
			args:    args{channelID: "1"},
			plugin:  pluginFromRepoNames([]string{"asd", "123", ""}, "1"),
			want:    wantedSubscriptions([]string{"asd", "123", ""}, "1"),
			wantErr: false,
		},
		{
			name:    "test empty",
			args:    args{channelID: "1"},
			plugin:  pluginFromRepoNames([]string{}, "1"),
			want:    wantedSubscriptions([]string{}, "1"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.plugin.GetSubscriptionsByChannel(tt.args.channelID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSubscriptionsByChannel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !compareSubsByRepoName(got, tt.want) {
				t.Errorf("GetSubscriptionsByChannel() got = %v, want %v", got, tt.want)
			}
		})
	}
}
