package plugin

func (p *Plugin) trackStartConfigurationWizard(userID string) {
	_ = p.tracker.TrackUserEvent("configuration_wizard_start", userID, map[string]interface{}{})
}

func (p *Plugin) trackCompleteConfigurationWizard(userID string) {
	_ = p.tracker.TrackUserEvent("configuration_wizard_complete", userID, map[string]interface{}{})
}
