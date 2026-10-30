# GitHub plugin Playwright tests

Two lanes. The fake lane runs on every PR, including forks, with no secrets. The real lane talks to github.com and runs nightly only when enabled.

The older suite in `e2e/playwright` (and `playwright.yml`) is unchanged and still runs against its own mock.

```sh
make e2e-env-up    # Postgres + Mattermost on :8069
make e2e           # builds the E2E bundle (make e2e-dist), installs the plugin, runs the suite
make e2e-env-down

make e2e-real      # normal bundle, against github.com. Needs e2e/integration/.env
```

`make e2e-dist` bakes `http://fake-github:8093` into `testOAuthServerURL`, which now moves the OAuth, REST, GraphQL and upload base URLs together, so the github.com code path is what gets tested. The plugin logs a warning on activation. Never ship that bundle.

Overrides go in the shell or in `e2e/integration/.env` (copy `.env.example`; docker compose reads it too). Real values live in 1Password or GitHub Actions secrets.

| Variable | Default | Purpose |
|---|---|---|
| `MM_IMAGE` | `mattermostdevelopment/mattermost-enterprise-edition:master` | Server image |
| `MM_PORT` | `8069` | Host port; must match between `e2e-env-up` and `e2e` |
| `PW_BROWSER_PORT` | `3336` | Host port of the dockerized visual browser |
| `PW_HEADLESS` | `true` | `false` to watch the browser |

## How GitHub is faked

- `fake-github/server.ts` runs inside the Playwright process on port 8093. Mattermost reaches it as `fake-github` through `extra_hosts`; Chromium through `--host-resolver-rules`.
- It is a classic OAuth app: tokens never expire and have no refresh token. Token errors come back as HTTP 200 with an `error` field, as form or JSON depending on `Accept`. Scopes from earlier grants are merged, and consent is skipped when an earlier grant already covers the request.
- Permissions follow GitHub: private repos need `repo`; repo hooks need repo admin plus `repo` or a `*:repo_hook` scope, otherwise 404; org hooks need org admin plus `admin:org_hook`; notifications need `notifications` or `repo`; issue labels, assignees and milestone are silently dropped without push access; SAML-enforced orgs answer 403 with `X-GitHub-SSO`.
- Creating an issue fires `issues.opened` to matching hooks and creating a hook sends `ping`, signed with `X-Hub-Signature` (HMAC-SHA1) and `X-Hub-Signature-256`.
- Tests drive it through `support/fake_github.ts` (`reset`, `createRepo`, `createItem`, `webhook`, `fault`, `sso`, `revoke`, `requests`, ...).
- Only `connect.spec.ts` drives the browser OAuth flow. Other specs connect with `seedUserInfo()` (`support/kv.ts`), which writes `<userID>_githubtoken` with the plugin's AES-CFB cipher. `installPlugin()` also seeds `mm34646_token_reset_done`, or the plugin resets every stored token a minute after it starts.
- Don't use `pw.initSetup()`: it resets server config and disables the plugin.

## Keeping the fake honest

Every request and response is checked, and each test fails at teardown if anything broke the contract.

- **REST:** validated against [github/rest-api-description](https://github.com/github/rest-api-description) (`api.github.com.json`), trimmed to the operations in `fake-github/contract/operations.json`. A route that isn't listed fails the run.
- **GraphQL:** queries are validated and executed against GitHub's [public schema](https://docs.github.com/en/graphql/overview/public-schema), trimmed to the types the plugin uses. graphql-js rejects `... on Issue { state } ... on PullRequest { state }` as a field conflict while GitHub answers it, so that one message is ignored. The real lane checks GitHub still accepts it.
- **Webhooks:** validated against the `x-webhooks` schemas in the same REST description. These replaced [octokit/webhooks](https://github.com/octokit/webhooks), which is deprecated and no longer updated.
- Nightly, `npm run contract:check` fails if GitHub changed the committed specs. `npm run contract:update` refreshes them.

## Real GitHub

`npm run test:real` (or `make e2e-real`) needs the variables in `.env.example`. Global setup:

- Creates long-lived fixtures once and reuses them: an `e2e-fixture` label, an open issue assigned to the machine account, and an open pull request from the `e2e-fixture` branch. It then polls search until both show up, because the plugin finds them through search and the index lags.
- Points a repository hook at a public placeholder URL with a fresh secret. GitHub refuses hooks to private addresses, so tests fetch the recorded delivery from the hook deliveries API and post it to the plugin.
- Seeds the machine account's token into KV, so most tests skip the browser login.

The sign-in test logs in with password and TOTP, then revokes its grant so the next run starts clean. GitHub skips its emailed device check for accounts with 2FA.

Writes go through `real-github/rate_limit.ts`, which stays under GitHub's content limits (80 a minute, 500 an hour) across runs on one machine. Retries are off so failures don't repeat writes. Traces, screenshots and video are off when `CI` is set. Don't put account, organization or repository names in this tree.

## Screenshots

`npm run test:visual` compares `@visual` shots in a dockerized browser against a pinned server image, in its own throwaway stack, and does not fail the PR job. Update baselines with `npm run test:visual-update-snapshots` and commit the PNGs.
