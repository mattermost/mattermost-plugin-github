module github.com/mattermost/mattermost-plugin-github

go 1.12

require (
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/sprig v2.18.0+incompatible
	github.com/go-ldap/ldap v3.0.3+incompatible // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-github/v25 v25.1.1
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/jehiah/go-strftime v0.0.0-20171201141054-1d33003b3869 // indirect
	github.com/mattermost/mattermost-server v0.0.0-20191017141203-48c06e9bce3b
	github.com/minio/minio-go v0.0.0-20190422205105-a8704b60278f // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
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
