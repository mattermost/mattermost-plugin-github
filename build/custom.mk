# Include custom targets and environment variables here

ifndef MM_RUDDER_WRITE_KEY
	MM_RUDDER_WRITE_KEY = 1d5bMvdrfWClLxgK1FvV3s4U1tg
endif
GO_BUILD_FLAGS += -ldflags '-X "github.com/mattermost/mattermost-plugin-api/experimental/telemetry.rudderWriteKey=$(MM_RUDDER_WRITE_KEY)"'

ifdef PLUGIN_E2E_MOCK_OAUTH_SERVER_URL
GO_BUILD_FLAGS += -ldflags '-X "github.com/mattermost/mattermost-plugin-github/server/plugin.e2eOAuthServerURL=$(PLUGIN_E2E_MOCK_OAUTH_SERVER_URL)"'
endif
