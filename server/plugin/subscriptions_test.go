package plugin

import (
	"context"
	"testing"

	"github.com/google/go-github/v54/github"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost/server/public/pluginapi"
)

func CheckError(t *testing.T, wantErr bool, err error) {
	message := "should return no error"
	if wantErr {
		message = "should return error"
	}
	assert.Equal(t, wantErr, err != nil, message)
}

// pluginWithSubs returns a plugin with given subscriptions.
func pluginWithSubs(t *testing.T, subscriptions []*Subscription) *Plugin {
	p := NewPlugin()
	p.client = pluginapi.NewClient(p.API, p.Driver)

	store := &pluginapi.MemoryStore{}
	p.store = store

	for _, sub := range subscriptions {
		err := p.AddSubscription(sub.Repository, sub)
		require.NoError(t, err)
	}

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
			plugin: pluginWithSubs(t, []*Subscription{
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
			plugin:  pluginWithSubs(t, []*Subscription{}),
			want:    wantedSubscriptions([]string{}, "1"),
			wantErr: false,
		},
		{
			name: "test shuffled",
			args: args{channelID: "1"},
			plugin: pluginWithSubs(t, []*Subscription{
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

func TestAddFlag(t *testing.T) {
	tests := []struct {
		name    string
		flags   SubscriptionFlags
		flag    string
		value   string
		want    bool
		wantErr bool
	}{
		{
			name:    "IncludeOnlyOrgMembers flag is parsed",
			flags:   SubscriptionFlags{},
			flag:    "include-only-org-members",
			value:   "true",
			want:    true,
			wantErr: false,
		},
		{
			name:    "IncludeOnlyOrgMembers flag cannot be parsed",
			flags:   SubscriptionFlags{},
			flag:    "include-only-org-members",
			value:   "test",
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			err := tt.flags.AddFlag(tt.flag, tt.value)
			CheckError(t, tt.wantErr, err)
			assert.Equal(t, tt.flags.IncludeOnlyOrgMembers, tt.want)
		})
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		name  string
		flags SubscriptionFlags
		flag  string
		value string
		want  string
	}{
		{
			name:  "Return --include-only-org-members string",
			flags: SubscriptionFlags{},
			flag:  "include-only-org-members",
			value: "true",
			want:  "--include-only-org-members true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			_ = tt.flags.AddFlag(tt.flag, tt.value)
			got := tt.flags.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSubscribe(t *testing.T) {
	tests := []struct {
		name   string
		flags  SubscriptionFlags
		plugin *Plugin
		errMsg string
	}{
		{
			name:   "Return error if GitHub organization is not set when --include-only-org-members flag is true",
			flags:  SubscriptionFlags{IncludeOnlyOrgMembers: true},
			plugin: NewPlugin(),
			errMsg: "Unable to set --include-only-org-members flag. The GitHub plugin is not locked to a single organization.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			err := tt.plugin.Subscribe(
				context.Background(),
				github.NewClient(nil),
				model.NewId(),
				"test-owner",
				"test-repo",
				model.NewId(),
				"test-features",
				tt.flags,
			)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}
