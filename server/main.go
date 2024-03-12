package main

import (
	mmplugin "github.com/mattermost/mattermost/server/public/plugin"

	"github.com/mattermost/mattermost-plugin-github/server/plugin"
)

func main() {
	mmplugin.ClientMain(plugin.NewPlugin())
}
