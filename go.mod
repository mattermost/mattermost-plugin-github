module github.com/mattermost/mattermost-plugin-github

go 1.16

require (
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/google/go-github/v41 v41.0.0
	github.com/gorilla/mux v1.8.0
	github.com/mattermost/mattermost-plugin-api v0.1.3-0.20230323124751-86c7be7ffbac
	// mmgoget: github.com/mattermost/mattermost-server/v6@v7.5.0 is replaced by -> github.com/mattermost/mattermost-server/v6@21aec2741b
	github.com/mattermost/mattermost-server/v6 v6.0.0-20221109191448-21aec2741bfe
	github.com/microcosm-cc/bluemonday v1.0.19
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.8.0
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8
)
