# Include custom targets and environment variables here

# If there's no MM_RUDDER_PLUGINS_PROD, add DEV data
RUDDER_WRITE_KEY = 1d5bMvdrfWClLxgK1FvV3s4U1tg
ifdef MM_RUDDER_PLUGINS_PROD
	RUDDER_WRITE_KEY = $(MM_RUDDER_PLUGINS_PROD)
endif

LDFLAGS += -X "github.com/mattermost/mattermost-plugin-api/experimental/telemetry.rudderWriteKey=$(RUDDER_WRITE_KEY)"
ifdef PLUGIN_E2E_MOCK_OAUTH_SERVER_URL
	LDFLAGS +=  -X "github.com/mattermost/mattermost-plugin-github/server/plugin.testOAuthServerURL=$(PLUGIN_E2E_MOCK_OAUTH_SERVER_URL)"
endif
GO_BUILD_FLAGS = -ldflags '$(LDFLAGS)'


