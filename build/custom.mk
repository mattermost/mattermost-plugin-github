# Include custom targets and environment variables here

ifdef PLUGIN_E2E_MOCK_OAUTH_SERVER_URL
	GO_BUILD_FLAGS += -ldflags '-X "github.com/mattermost/mattermost-plugin-github/server/plugin.testOAuthServerURL=$(PLUGIN_E2E_MOCK_OAUTH_SERVER_URL)"'
endif
.PHONY: deploy-e2e
deploy-e2e: dist
	E2E_TESTING=true PLUGIN_E2E_MOCK_OAUTH_SERVER_URL=http://localhost:8080 ./build/bin/pluginctl deploy $(PLUGIN_ID) dist/$(BUNDLE_NAME)

E2E_DIR = e2e/integration
E2E_FAKE_GITHUB_URL ?= http://fake-github:8093

.PHONY: e2e-dist
e2e-dist: ## Builds dist/ with GitHub OAuth, REST and GraphQL pointed at the E2E fake. Never ship this bundle.
	$(MAKE) dist PLUGIN_E2E_MOCK_OAUTH_SERVER_URL=$(E2E_FAKE_GITHUB_URL)

$(E2E_DIR)/node_modules: $(E2E_DIR)/package-lock.json
	cd $(E2E_DIR) && $(NPM) ci && npx playwright install chromium
	touch $@

.PHONY: e2e-env-up
e2e-env-up: ## Starts the Mattermost stack used by the E2E suite
	cd $(E2E_DIR) && docker compose up -d --wait

.PHONY: e2e-env-down
e2e-env-down: ## Stops the E2E Mattermost stack and drops its data
	cd $(E2E_DIR) && docker compose --profile visual down -v

.PHONY: e2e
e2e: e2e-dist $(E2E_DIR)/node_modules ## Runs the fake-GitHub E2E suite against the stack from e2e-env-up
	cd $(E2E_DIR) && npx playwright test

.PHONY: e2e-real
e2e-real: dist $(E2E_DIR)/node_modules ## Runs the real GitHub journey (needs e2e/integration/.env)
	cd $(E2E_DIR) && npm run test:real
