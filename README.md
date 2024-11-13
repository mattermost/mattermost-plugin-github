# Mattermost Forgejo Plugin

[![Build Status](https://img.shields.io/circleci/project/github/mattermost/mattermost-plugin-github/master.svg)](https://circleci.com/gh/mattermost/mattermost-plugin-github)
[![Code Coverage](https://img.shields.io/codecov/c/github/mattermost/mattermost-plugin-github/master.svg)](https://codecov.io/gh/mattermost/mattermost-plugin-github)
[![Release](https://img.shields.io/github/v/release/mattermost/mattermost-plugin-github)](https://github.com/mattermost/mattermost-plugin-github/releases/latest)
[![HW](https://img.shields.io/github/issues/mattermost/mattermost-plugin-github/Up%20For%20Grabs?color=dark%20green&label=Help%20Wanted)](https://github.com/mattermost/mattermost-plugin-github/issues?q=is%3Aissue+is%3Aopen+sort%3Aupdated-desc+label%3A%22Up+For+Grabs%22+label%3A%22Help+Wanted%22)

A Forgejo plugin for Mattermost

See the [Mattermost Product Documentation](https://docs.mattermost.com/integrate/github-interoperability.html) for details on installing, configuring, enabling, and using this Mattermost integration.

## Development

This plugin contains both a server and web app portion. Read our documentation about the [Developer Workflow](https://developers.mattermost.com/integrate/plugins/developer-workflow/) and [Developer Setup](https://developers.mattermost.com/integrate/plugins/developer-setup/) for more information about developing and extending plugins.

### Releasing new versions

The version of a plugin is determined at compile time, automatically populating a `version` field in the [plugin manifest](plugin.json):
* If the current commit matches a tag, the version will match after stripping any leading `v`, e.g. `1.3.1`.
* Otherwise, the version will combine the nearest tag with `git rev-parse --short HEAD`, e.g. `1.3.1+d06e53e1`.
* If there is no version tag, an empty version will be combined with the short hash, e.g. `0.0.0+76081421`.

To disable this behaviour, manually populate and maintain the `version` field.

## How to Release

To trigger a release, follow these steps:

1. **For Patch Release:** Run the following command:
    ```
    make patch
    ```
   This will release a patch change.

2. **For Minor Release:** Run the following command:
    ```
    make minor
    ```
   This will release a minor change.

3. **For Major Release:** Run the following command:
    ```
    make major
    ```
   This will release a major change.

4. **For Patch Release Candidate (RC):** Run the following command:
    ```
    make patch-rc
    ```
   This will release a patch release candidate.

5. **For Minor Release Candidate (RC):** Run the following command:
    ```
    make minor-rc
    ```
   This will release a minor release candidate.

6. **For Major Release Candidate (RC):** Run the following command:
    ```
    make major-rc
    ```
   This will release a major release candidate.

### Playwright e2e tests

In order to get your environment set up to run [Playwright](https://playwright.dev) tests, please see the setup guide at [e2e/playwright](/e2e/playwright#readme).
