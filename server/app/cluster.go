package app

import (
	"encoding/json"

	"github.com/google/go-github/v41/github"
	"github.com/mattermost/mattermost-server/v6/model"
)

const (
	webHookPingEventID   = "webhook-hello"
	oauthCompleteEventID = "oauth-complete"
)

func (a *App) sendGitHubPingEvent(event *github.PingEvent) {
	a.sendMessageToCluster(webHookPingEventID, event)
}

func (a *App) sendOAuthCompleteEvent(event OAuthCompleteEvent) {
	a.sendMessageToCluster(oauthCompleteEventID, event)
}

func (a *App) sendMessageToCluster(id string, v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		a.client.Log.Warn("couldn't get JSON bytes from cluster message",
			"id", id,
			"error", err,
		)
		return
	}

	event := model.PluginClusterEvent{Id: id, Data: b}
	opts := model.PluginClusterEventSendOptions{
		SendType: model.PluginClusterEventSendTypeReliable,
	}

	if err := a.client.Cluster.PublishPluginEvent(event, opts); err != nil {
		a.client.Log.Warn("error publishing cluster event",
			"id", id,
			"error", err,
		)
	}
}

func (a *App) HandleClusterEvent(ev model.PluginClusterEvent) {
	switch ev.Id {
	case webHookPingEventID:
		var event github.PingEvent
		if err := json.Unmarshal(ev.Data, &event); err != nil {
			a.client.Log.Warn("cannot unmarshal cluster event with GitHub ping event", "error", err)
			return
		}

		a.WebhookBroker.publishPing(&event, true)
	case oauthCompleteEventID:
		var event OAuthCompleteEvent
		if err := json.Unmarshal(ev.Data, &event); err != nil {
			a.client.Log.Warn("cannot unmarshal cluster event with OAuth complete event", "error", err)
			return
		}

		a.OauthBroker.PublishOAuthComplete(event.UserID, event.Err, true)
	default:
		a.client.Log.Warn("unknown cluster event", "id", ev.Id)
	}
}
