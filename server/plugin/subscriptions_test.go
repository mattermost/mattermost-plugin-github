package plugin

import (
	"encoding/json"
	"testing"

	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/stretchr/testify/assert"
)

func CheckError(t *testing.T, wantErr bool, err error) {
	message := "should return no error"
	if wantErr {
		message = "should return error"
	}
	assert.Equal(t, wantErr, err != nil, message)
}

// pluginWithMockedSubs returns mocked plugin for given subscriptions
func pluginWithMockedSubs(subscriptions []*Subscription) *Plugin {
	p := NewPlugin()
	mockPluginAPI := &plugintest.API{}

	subs := Subscriptions{Repositories: map[string][]*Subscription{}}
	subs.Repositories[""] = subscriptions
	jsn, _ := json.Marshal(subs)
	mockPluginAPI.On("KVGet", SubscriptionsKey).Return(jsn, nil)
	p.SetAPI(mockPluginAPI)
	p.client = pluginapi.NewClient(p.API, p.Driver)
	return p
}

// wantedSubscriptions returns what should be returned after sorting by repo names
func wantedSubscriptions(repoNames []string, chanelID string) []*Subscription {
	var subs []*Subscription
	for _, st := range repoNames {
		subs = append(subs, &Subscription{
			ChannelID:  chanelID,
			Repository: st,
		})
	}
	return subs
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
			name: "basic test",
			args: args{channelID: "1"},
			plugin: pluginWithMockedSubs([]*Subscription{
				{
					ChannelID:  "1",
					Repository: "asd",
				},
				{
					ChannelID:  "1",
					Repository: "123",
				},
				{
					ChannelID:  "1",
					Repository: "",
				},
			}),
			want:    wantedSubscriptions([]string{"", "123", "asd"}, "1"),
			wantErr: false,
		},
		{
			name:    "test empty",
			args:    args{channelID: "1"},
			plugin:  pluginWithMockedSubs([]*Subscription{}),
			want:    wantedSubscriptions([]string{}, "1"),
			wantErr: false,
		},
		{
			name: "test shuffled",
			args: args{channelID: "1"},
			plugin: pluginWithMockedSubs([]*Subscription{
				{
					ChannelID:  "1",
					Repository: "c",
				},
				{
					ChannelID:  "1",
					Repository: "b",
				},
				{
					ChannelID:  "1",
					Repository: "ab",
				},
				{
					ChannelID:  "1",
					Repository: "a",
				},
			}),
			want:    wantedSubscriptions([]string{"a", "ab", "b", "c"}, "1"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.plugin.GetSubscriptionsByChannel(tt.args.channelID)

			CheckError(t, tt.wantErr, err)

			assert.Equal(t, tt.want, got, "they should be same")
		})
	}
}
