# Mattermost GitHub Plugin ![CircleCI branch](https://img.shields.io/circleci/project/github/mattermost/mattermost-plugin-github/master.svg)

A GitHub plugin for Mattermost. Supports GitHub SaaS and Enterprise versions.

![GitHub plugin screenshot](https://user-images.githubusercontent.com/13119842/54380268-6adab180-4661-11e9-8470-a9c615c00041.png)

## Features

* __Daily reminders__ - the first time you log in to Mattermost each day, get a post letting you know what issues and pull requests need your attention
* __Notifications__ - get a direct message in Mattermost when someone mentions you, requests your review, comments on or modifies one of your pull requests/issues, or assigns you on GitHub
* __Sidebar buttons__ - stay up-to-date with how many reviews, unread messages, assignments and open pull requests you have with buttons in the Mattermost sidebar
* __Slash commands__ - interact with the GitHub plugin using the `/github` slash command
    * __Subscribe to a respository__ - Use `/github subscribe` to subscribe a Mattermost channel to receive notifications for new pull requests, issues, branch creation and more in a GitHub repository.
        - For instance, to post notifications for issues, issue comments and pull requests matching the label `Help Wanted` from `mattermost/mattermost-server`, use
        
          ```
          /github subscribe mattermost/mattermost-server issues,pulls,issue_comments,label:"Help Wanted"
          ```

    * __Get to do items__ - Use `/github todo` to get an ephemeral message with items to do in GitHub, including a list of unread messages and pull requests awaiting your review
    * __Update settings__ - Use `/github settings` to update your settings for notifications and daily reminders
    * __And more!__ - Run `/github help` to see what else the slash command can do

To use any of the above features, first connect your Mattermost account to your GitHub account with `/github connect`.

## Configuration

__If you are using GitHub Enterprise, replace all GitHub links below with your GitHub Enterprise URL__

### Step 1: Register an OAuth application in GitHub

1. Go to https://github.com/settings/applications/new to register an OAuth app.
2. Set the following values:
   - "Mattermost GitHub Plugin - <your company name>" as the **Application Name**.
   - "https://github.com/mattermost/mattermost-plugin-github" as the **Homepage URL**.
   - "https://your-mattermost-url.com/plugins/github/oauth/complete" as the **Authorization callback URL**, replacing `https://your-mattermost-url.com` with your Mattermost URL.
3. Submit. Copy the **Client ID** and **Client Secret** in the resulting screen.
4. Go to **System Console > Plugins > GitHub** and enter **GitHub OAuth Client ID** and **GitHub OAuth Client Secret** you copied in a previous step.
   
   **Note**: If you are running Mattermost v5.11 or earlier, you must first go to the [releases page of this GitHub repository](https://github.com/mattermost/mattermost-plugin-github/releases), download the latest release, and upload it to your Mattermost instance [following this documentation](https://docs.mattermost.com/administration/plugins.html#plugin-uploads).
5. Hit **Save**.

### Step 2: Create a webhook in GitHub

1. In **System Console > Plugins > GitHub**, generate a new value for **Webhook Secret**. Copy it as you will use it in a later step.
2. Hit **Save** to save the secret.
3. Go to the **Settings** page of your GitHub organization you want to send notifications from, then select **Webhooks** in the sidebar.
   
   **Note**: You must create a webhook for each organization you want to receive notifications for or subscribe to.

4. Click **Add webhook**
5. Set the following values:
   - "https://your-mattermost-url.com/plugins/github/webhook" as the **Payload URL**, replacing `https://your-mattermost-url.com` with your Mattermost URL
   - "application/json" as the **Content Type**
   - the webhook secret you copied previously as the **Secret**
6. Select **Let me select individual events** for "Which events would you like to trigger this webhook?", then select the following events: Issues, Issue comments, Pull requests, Pull request reviews, Pull request review comments, Pushes, Branch or Tag creation, Branch or Tag deletion
7. Hit **Add Webhook** to save it.

### Step 3: Configure plugin in Mattermost

1. Make sure you do not have an existing user account with the name `github`. You probably do if you are upgrading from a previous version of the plugin. If you do, convert it to a bot account by using the Mattermost CLI command `mattermost user convert --bot github`. If you where using an account that was not named github for the github plugin, rename it go github then convert it to a bot account.
2. Go to **System Console > Plugins > GitHub** and do the following values:
  - Generate a new value for **At Rest Encryption Key**.
  - (Optional) Lock the plugin to a single GitHub organization by setting the **GitHub Organization** field to the name of your GitHub organization.
  - (Optional) Allow the plugin to receive notifications from private repositories by setting **Enable Private Repositories** to true.
   
    When enabled, existing users must reconnect their accounts to gain access to private repositories. Affected users will be notified by the plugin once private repositories are enabled.

  - (Enterprise only) Set the **Enterprise Base URL** and **Enterprise Upload URL** to your GitHub Enterprise URLs, e.g. `https://github.example.com`. The Base and Upload URLs are often the same.
3. Hit **Save**.
4. Go to **System Console > Plugins > Management** and click **Enable** to enable the GitHub plugin.

You're all set! To test it, run the `/github connect` slash command to connect your Mattermost account with GitHub.

### Configuration Notes in HA

If you are running Mattermost v5.11 or earlier in [High Availability mode](https://docs.mattermost.com/deployment/cluster.html), please review the following:

1. To install the plugin, [use these documented steps](https://docs.mattermost.com/administration/plugins.html#plugin-uploads-in-high-availability-mode)
2. Then, modify the config.json [using the standard doc steps](https://docs.mattermost.com/deployment/cluster.html#updating-configuration-changes-while-operating-continuously) to the following

```
"PluginSettings": {
    ...
    "Plugins": {
        "github": {
            "encryptionkey": "<your encryption key, from step 5 above>",
            "githuboauthclientid": "<your oauth client id, from step 2 above>",
            "githuboauthclientsecret": "<your oauth client secret, from step 2 above>",
            "githuborg": "<your github org>",
            "webhooksecret": "<your webhook secret, from step 3 above>"
        },
    ...
    "PluginStates": {
        ...
        "github": {
            "Enable": true
        },
        ...
    }
},
```

## Development

This plugin contains both a server and web app portion.

Use `make dist` to build distributions of the plugin that you can upload to a Mattermost server.

Use `make check-style` to check the style.

Use `make deploy` to deploy the plugin to your local server. Before running `make deploy` you need to set a few environment variables:

```
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_USERNAME=admin
export MM_ADMIN_PASSWORD=password
```

## Frequently Asked Questions

### How do I connect a repository instead of an organization?

Set up your GitHub webhook from the repository instead of the organization. Notifications and subscriptions will then be sent only for repositories you create webhooks for.

The reminder and `/github todo` will still search the whole organization, but only list items assigned to you.

## Feedback and Feature Requests

Feel free to create a GitHub issue or [join the GitHub Plugin channel on our community Mattermost instance](https://pre-release.mattermost.com/core/channels/github-plugin) to discuss.
