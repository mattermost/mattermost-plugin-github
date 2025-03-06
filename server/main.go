// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	mmplugin "github.com/mattermost/mattermost/server/public/plugin"

	"github.com/mattermost/mattermost-plugin-github/server/plugin"
)

func main() {
	mmplugin.ClientMain(plugin.NewPlugin())
}
