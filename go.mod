module github.com/mattermost/mattermost-plugin-github

go 1.12

require (
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/sprig v2.18.0+incompatible
	github.com/go-ldap/ldap v3.0.3+incompatible // indirect
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-github/v25 v25.1.1
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/mattermost/mattermost-server v0.0.0-20190610144121-1a7a34b652f6
	github.com/nicksnyder/go-i18n v1.10.0 // indirect
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.4.0
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
)

replace (
	git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999
	// Workaround for https://github.com/golang/go/issues/30831 and fallout.
	github.com/golang/lint => github.com/golang/lint v0.0.0-20190227174305-8f45f776aaf1
)

replace github.com/mattermost/mattermost-server => /Users/gsagula/go/src/github.com/mattermost/mattermost-server
