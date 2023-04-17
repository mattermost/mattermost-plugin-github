package plugin

import (
	"strings"

	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-plugin-api/experimental/bot/logger"
	"github.com/mattermost/mattermost-plugin-api/experimental/telemetry"
	"github.com/pkg/errors"
)

const (
	keysPerPage = 1000
)

func (p *Plugin) TrackEvent(event string, properties map[string]interface{}) {
	err := p.tracker.TrackEvent(event, properties)
	if err != nil {
		p.client.Log.Debug("Error sending telemetry event", "event", event, "error", err.Error())
	}
}

func (p *Plugin) TrackUserEvent(event, userID string, properties map[string]interface{}) {
	err := p.tracker.TrackUserEvent(event, userID, properties)
	if err != nil {
		p.client.Log.Debug("Error sending user telemetry event", "event", event, "error", err.Error())
	}
}

func (p *Plugin) SendDailyTelemetry() {
	config := p.getConfiguration()

	connectedUserCount, err := p.getConnectedUserCount()
	if err != nil {
		p.client.Log.Warn("Failed to get the number of connected users for telemetry", "error", err)
	}

	p.TrackEvent("stats", map[string]interface{}{
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

func (p *Plugin) getConnectedUserCount() (int64, error) {
	checker := func(key string) (keep bool, err error) {
		return strings.HasSuffix(key, githubTokenKey), nil
	}

	var count int64

	for i := 0; ; i++ {
		keys, err := p.client.KV.ListKeys(i, keysPerPage, pluginapi.WithChecker(checker))
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
func (p *Plugin) initializeTelemetry() {
	var err error

	// Telemetry client
	p.telemetryClient, err = telemetry.NewRudderClient()
	if err != nil {
		p.client.Log.Debug("Telemetry client not started", "error", err.Error())
		return
	}

	// Get config values
	p.tracker = telemetry.NewTracker(
		p.telemetryClient,
		p.client.System.GetDiagnosticID(),
		p.client.System.GetServerVersion(),
		Manifest.Id,
		Manifest.Version,
		"github",
		telemetry.NewTrackerConfig(p.client.Configuration.GetConfig()),
		logger.New(p.API),
	)
}
