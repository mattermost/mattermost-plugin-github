package main

import (
	"net/http"
	"sync/atomic"

	"github.com/mattermost/mattermost-server/plugin"
)

type Plugin struct {
	api           plugin.API
	configuration atomic.Value
}

func (p *Plugin) OnActivate(api plugin.API) error {
	p.api = api
	if err := p.OnConfigurationChange(); err != nil {
		return err
	}

	config := p.config()
	if err := config.IsValid(); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) config() *Configuration {
	return p.configuration.Load().(*Configuration)
}

func (p *Plugin) OnConfigurationChange() error {
	var configuration Configuration
	err := p.api.LoadPluginConfiguration(&configuration)
	p.configuration.Store(&configuration)
	return err
}

func (p *Plugin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	config := p.config()
	if err := config.IsValid(); err != nil {
		http.Error(w, "This plugin is not configured.", http.StatusNotImplemented)
		return
	}

	switch path := r.URL.Path; path {
	default:
		http.NotFound(w, r)
	}
}
