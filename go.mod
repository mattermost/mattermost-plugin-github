module github.com/mattermost/mattermost-plugin-github

go 1.16

require (
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/google/go-github/v41 v41.0.0
	github.com/gorilla/mux v1.8.0
	github.com/mattermost/mattermost-plugin-api v0.0.24
	github.com/mattermost/mattermost-server/v6 v6.3.0
	github.com/microcosm-cc/bluemonday v1.0.17
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8
)

// Until github.com/mattermost/mattermost-server/v6 v6.5.0 is releated,
// this replacement is needed to also import github.com/mattermost/mattermost-plugin-api,
// which uses a different server version.
replace github.com/mattermost/mattermost-server/v6 v6.3.0 => github.com/mattermost/mattermost-server/v6 v6.0.0-20220204112347-b6128201bb5d
