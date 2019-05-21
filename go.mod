module github.com/mattermost/mattermost-plugin-github

go 1.12

require (
	github.com/go-ldap/ldap v3.0.3+incompatible // indirect
	github.com/google/go-github v17.0.0+incompatible
	github.com/hashicorp/go-hclog v0.9.2 // indirect
	github.com/mattermost/mattermost-server v0.0.0-20190508185452-66cb36f5dc35
	github.com/pelletier/go-toml v1.4.0 // indirect
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.3.0
	go.uber.org/atomic v1.4.0 // indirect
	go.uber.org/zap v1.10.0 // indirect
	golang.org/x/crypto v0.0.0-20190513172903-22d7a77e9e5f // indirect
	golang.org/x/oauth2 v0.0.0-20190517181255-950ef44c6e07
)

// Workaround for https://github.com/golang/go/issues/30831 and fallout.
replace github.com/golang/lint => github.com/golang/lint v0.0.0-20190227174305-8f45f776aaf1
