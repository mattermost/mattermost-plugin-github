# Include custom targets and environment variables here

# If there's no MM_RUDDER_PLUGINS_PROD, add DEV data
ifndef MM_RUDDER_PLUGINS_PROD
	MM_RUDDER_PLUGINS_PROD = 1d5bMvdrfWClLxgK1FvV3s4U1tg
endif
GO_BUILD_FLAGS += -ldflags '-X "github.com/mattermost/mattermost-plugin-api/experimental/telemetry.rudderWriteKey=$(MM_RUDDER_PLUGINS_PROD)"'
