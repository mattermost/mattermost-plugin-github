package main

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/google/go-github/github"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"golang.org/x/oauth2"
)

type Plugin struct {
	api           plugin.API
	configuration atomic.Value
	githubClient  *github.Client
}

func githubConnect(token string) *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	return client
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

	// Connect to github
	p.githubClient = githubConnect(config.GithubToken)

	// Register commands
	p.api.RegisterCommand(&model.Command{
		Trigger:     "github",
		DisplayName: "Github",
		Description: "Integration with Github.",
	})

	return nil
}

func (p *Plugin) ExecuteCommand(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	split := strings.Split(args.Command, " ")
	command := split[0]
	parameters := []string{}
	action := ""
	if len(split) > 1 {
		action = split[1]
	}
	if len(split) > 2 {
		parameters = split[2:]
	}

	if command != "/github" {
		return nil, nil
	}

	switch action {
	case "subscribe":
		if len(parameters) != 1 {
			return nil, nil
		}
	case "register":
	case "todo":
	}

	resp := &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
		Text:         "You have subscribed to the repository.",
		Username:     "github",
		IconURL:      "https://assets-cdn.github.com/images/modules/logos_page/GitHub-Mark.png",
		Type:         model.POST_DEFAULT,
	}

	return resp, nil
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
