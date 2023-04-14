In order to get your environment set up to run [Playwright](https://playwright.dev) tests, you can run `./setup-environment`, or run equivalent commands for your current setup.

What this script does:

- Navigate to the folder above `mattermost-plugin-github`
- Clone `mattermost-server` (if it is already cloned there, please have a clean git index to avoid issues with conflicts)
- `cd mattermost-server`
- Install webapp dependencies - `cd webapp && npm i`
- Install Playwright test dependencies - `cd ../e2e-tests/playwright && npm i`
- Install Playwright - `npx install playwright`
- Install GitHub plugin e2e dependencies - `cd ../../../mattermost-plugin-github/e2e/playwright && npm i`
- Build and deploy plugin with mock OAuth server URL env var - `PLUGIN_E2E_MOCK_OAUTH_SERVER_URL=http://localhost:8080 make deploy`

-----

Then to run the tests:

Start Mattermost server:
- `cd <path>/mattermost-server/server`
- `make test-data`
- `make run-server`

Run test:
- Create a personal access token from GitHub
- Set `PLUGIN_E2E_MOCK_OAUTH_TOKEN` environment variable to access token
- `cd <path>/mattermost-plugin-github/e2e/playwright`
- `npm test`

To see the test report:
- `cd <path>/mattermost-plugin-github/e2e/playwright`
- `npm run show-report`
- Navigate to http://localhost:9323

To see test screenshots:
- `cd <path>/mattermost-plugin-github/e2e/playwright/screenshots`
