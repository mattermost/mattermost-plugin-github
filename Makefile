.PHONY: dist

GOOS=$(shell uname -s | tr '[:upper:]' '[:lower:]')
GOARCH=amd64

.PHONY: build test run clean stop check-style gofmt

check-style: webapp/.npminstall gofmt
	@echo Checking for style guide compliance

	cd webapp && npm run check

gofmt:
	@echo Running GOFMT

	@for package in $$(go list ./server/...); do \
		echo "Checking "$$package; \
		files=$$(go list -f '{{range .GoFiles}}{{$$.Dir}}/{{.}} {{end}}' $$package); \
		if [ "$$files" ]; then \
			gofmt_output=$$(gofmt -d -s $$files 2>&1); \
			if [ "$$gofmt_output" ]; then \
				echo "$$gofmt_output"; \
				echo "gofmt failure"; \
				exit 1; \
			fi; \
		fi; \
	done
	@echo "gofmt success"; \

test: webapp/.npminstall
	@echo Not yet implemented

webapp/.npminstall:
	@echo Getting dependencies using npm

	cd webapp && npm install
	touch $@

vendor: server/Gopkg.toml
	cd server && go get -u github.com/golang/dep/cmd/dep
	cd server && $(shell go env GOPATH)/bin/dep ensure

dist: webapp/.npminstall plugin.json
	@echo Building plugin

	# Clean old dist
	rm -rf dist
	rm -rf webapp/dist
	rm -f server/plugin.exe

	# Build and copy files from webapp
	cd webapp && npm run build
	mkdir -p dist/github/webapp
	cp webapp/dist/* dist/github/webapp/

	# Build files from server
	cd server && go get github.com/mitchellh/gox
	$(shell go env GOPATH)/bin/gox -osarch='darwin/amd64 linux/amd64 windows/amd64' -output 'dist/intermediate/plugin_{{.OS}}_{{.Arch}}' ./server

	# Copy plugin files
	cp plugin.json dist/github/

	# Copy server executables & compress plugin
	mkdir -p dist/github/server
	mv dist/intermediate/plugin_darwin_amd64 dist/github/server/plugin.exe
	cd dist && tar -zcvf mattermost-github-plugin-darwin-amd64.tar.gz github/*
	mv dist/intermediate/plugin_linux_amd64 dist/github/server/plugin.exe
	cd dist && tar -zcvf mattermost-github-plugin-linux-amd64.tar.gz github/*
	mv dist/intermediate/plugin_windows_amd64.exe dist/github/server/plugin.exe
	cd dist && tar -zcvf mattermost-github-plugin-windows-amd64.tar.gz github/*

	# Clean up temp files
	rm -rf dist/github
	rm -rf dist/intermediate

	@echo MacOS X plugin built at: dist/mattermost-github-plugin-darwin-amd64.tar.gz
	@echo Linux plugin built at: dist/mattermost-github-plugin-linux-amd64.tar.gz
	@echo Windows plugin built at: dist/mattermost-github-plugin-windows-amd64.tar.gz

localdeploy: dist
	cp dist/mattermost-github-plugin-$(GOOS)-$(GOARCH).tar.gz ../mattermost-server/plugins/
	rm -rf ../mattermost-server/plugins/github
	tar -C ../mattermost-server/plugins/ -zxvf ../mattermost-server/plugins/mattermost-github-plugin-$(GOOS)-$(GOARCH).tar.gz

run: webapp/.npminstall
	@echo Not yet implemented

stop:
	@echo Not yet implemented

clean:
	@echo Cleaning plugin

	rm -rf dist
	rm -rf webapp/dist
	rm -rf webapp/node_modules
	rm -rf webapp/.npminstall
	rm -f server/plugin.exe
