package telemetry

import (
	"strings"

	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-plugin-api/experimental/bot/logger"
	"github.com/mattermost/mattermost-plugin-api/experimental/telemetry"
	"github.com/mattermost/mattermost-plugin-github/server/app"
	"github.com/mattermost/mattermost-plugin-github/server/config"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/pkg/errors"
)

const (
	keysPerPage = 1000
)

type Telemetry struct {
	client        *pluginapi.Client
	api           plugin.API
	configService config.Service

	// telemetry client
	telemetryClient telemetry.Client

	// telemetry Tracker
	tracker telemetry.Tracker
}

type Tracker interface {
	TrackEvent(event string, properties map[string]interface{})
	TrackUserEvent(event, userID string, properties map[string]interface{})
}

func (t *Telemetry) TrackEvent(event string, properties map[string]interface{}) {
	err := t.tracker.TrackEvent(event, properties)
	if err != nil {
		t.client.Log.Debug("Error sending telemetry event", "event", event, "error", err.Error())
	}
}

func (t *Telemetry) TrackUserEvent(event, userID string, properties map[string]interface{}) {
	err := t.tracker.TrackUserEvent(event, userID, properties)
	if err != nil {
		t.client.Log.Debug("Error sending user telemetry event", "event", event, "error", err.Error())
	}
}

func (t *Telemetry) SendDailyTelemetry() {
	config := t.configService.GetConfiguration()

	connectedUserCount, err := t.getConnectedUserCount()
	if err != nil {
		t.client.Log.Warn("Failed to get the number of connected users for telemetry", "error", err)
	}

	t.TrackEvent("stats", map[string]interface{}{
		"connected_user_count":          connectedUserCount,
		"is_oauth_configured":           config.IsOAuthConfigured(),
		"is_sass":                       config.IsSASS(),
		"is_organization_locked":        config.GitHubOrg != "",
		"enable_private_repo":           config.EnablePrivateRepo,
		"enable_code_preview":           config.EnableCodePreview,
		"connect_to_private_by_default": config.ConnectToPrivateByDefault,
		"Use_preregistered_application": config.UsePreregisteredApplication,
	})
}

func (t *Telemetry) getConnectedUserCount() (int64, error) {
	checker := func(key string) (keep bool, err error) {
		return strings.HasSuffix(key, app.GithubTokenKey), nil
	}

	var count int64

	for i := 0; ; i++ {
		keys, err := t.client.KV.ListKeys(i, keysPerPage, pluginapi.WithChecker(checker))
		if err != nil {
			return 0, errors.Wrapf(err, "failed to list keys - page, %d", i)
		}

		count += int64(len(keys))

		if len(keys) < keysPerPage {
			break
		}
	}

	return count, nil
}

// Initialize telemetry setups the tracker/clients needed to send telemetry data.
// The telemetry.NewTrackerConfig(...) param will take care of extract/parse the config to set rge right settings.
// If you don't want the default behavior you still can pass a different telemetry.TrackerConfig data.
func (t *Telemetry) initializeTelemetry() {
	var err error

	// Telemetry client
	t.telemetryClient, err = telemetry.NewRudderClient()
	if err != nil {
		t.client.Log.Debug("Telemetry client not started", "error", err.Error())
		return
	}

	manifest := t.configService.GetManifest()

	// Get config values
	t.tracker = telemetry.NewTracker(
		t.telemetryClient,
		t.client.System.GetDiagnosticID(),
		t.client.System.GetServerVersion(),
		manifest.Id,
		manifest.Version,
		"github",
		telemetry.NewTrackerConfig(t.client.Configuration.GetConfig()),
		logger.New(t.api),
	)
}
