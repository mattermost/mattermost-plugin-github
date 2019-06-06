module github.com/mattermost/mattermost-plugin-github

go 1.12

require (
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.4.2 // indirect
	github.com/Masterminds/sprig v2.18.0+incompatible
	github.com/go-ldap/ldap v3.0.3+incompatible // indirect
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-github/v25 v25.1.1
	github.com/hashicorp/go-hclog v0.9.2 // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/lib/pq v1.1.1 // indirect
	github.com/mattermost/go-i18n v1.10.0 // indirect
	github.com/mattermost/mattermost-server v0.0.0-20190531122116-acf6437b8169
	github.com/nicksnyder/go-i18n v1.10.0 // indirect
	github.com/pelletier/go-toml v1.4.0 // indirect
	github.com/pkg/errors v0.8.1
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/stretchr/testify v1.3.0
	go.uber.org/atomic v1.4.0 // indirect
	go.uber.org/zap v1.10.0 // indirect
	golang.org/x/crypto v0.0.0-20190530122614-20be4c3c3ed5 // indirect
	golang.org/x/net v0.0.0-20190522155817-f3200d17e092 // indirect
	golang.org/x/oauth2 v0.0.0-20190523182746-aaccbc9213b0
	golang.org/x/sys v0.0.0-20190531175056-4c3a928424d2 // indirect
	google.golang.org/appengine v1.6.0 // indirect
	google.golang.org/genproto v0.0.0-20190530194941-fb225487d101 // indirect
	google.golang.org/grpc v1.21.0 // indirect
)

// Workaround for https://github.com/golang/go/issues/30831 and fallout.
replace github.com/golang/lint => github.com/golang/lint v0.0.0-20190227174305-8f45f776aaf1
