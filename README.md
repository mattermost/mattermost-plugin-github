# Mattermost GitHub Plugin

[![Build Status](https://img.shields.io/circleci/project/github/mattermost/mattermost-plugin-github/master.svg)](https://circleci.com/gh/mattermost/mattermost-plugin-github)
[![Code Coverage](https://img.shields.io/codecov/c/github/mattermost/mattermost-plugin-github/master.svg)](https://codecov.io/gh/mattermost/mattermost-plugin-github)
[![Release](https://img.shields.io/github/v/release/mattermost/mattermost-plugin-github)](https://github.com/mattermost/mattermost-plugin-github/releases/latest)
[![HW](https://img.shields.io/github/issues/mattermost/mattermost-plugin-github/Up%20For%20Grabs?color=dark%20green&label=Help%20Wanted)](https://github.com/mattermost/mattermost-plugin-github/issues?q=is%3Aissue+is%3Aopen+sort%3Aupdated-desc+label%3A%22Up+For+Grabs%22+label%3A%22Help+Wanted%22)

**Maintainer:** [@hanzei](https://github.com/hanzei)
**Co-Maintainer:** [@larkox](https://github.com/larkox)

A GitHub plugin for Mattermost. Supports GitHub SaaS and Enterprise versions.

## Table of Contents

 - [Audience](#audience)
 - [License](#license)
 - [About the GitHub Plugin](#about-the-github-plugin)
 - [Before You Start](#before-you-start)
 - [Configuration](#configuration)
 - [Using the Plugin](#using-the-plugin)
 - [Onboarding Your Users](#onboarding-your-users)
 - [Slash Commands](#slash-commands)
 - [Frequently Asked Questions](#frequently-asked-questions)
 - [Development](#development)

![GitHub plugin screenshot](images/github_mattermost.png)

## Audience

This guide is intended for Mattermost System Admins setting up the GitHub plugin, Mattermost users who want information about the plugin functionality, and Mattermost users who want to connect their GitHub account to Mattermost. For more information about contributing to this plugin, visit the [Development section](#development).

## License

This repository is licensed under the [Apache 2.0 License](https://github.com/mattermost/mattermost-plugin-github/blob/master/LICENSE).

## About the GitHub Plugin

The Mattermost GitHub plugin uses a webhook to connect your GitHub account to Mattermost to listen for incoming GitHub events. Events notifications are via DM in Mattermost. The Events don’t need separate configuration. 

After a System Admin has configured the GitHub plugin, run `/github connect` in a Mattermost channel to connect your Mattermost and GitHub accounts.

Once connected, you'll have access to the following features:

* __Daily reminders__ - The first time you log in to Mattermost each day, get a post letting you know what issues and pull requests need your attention.
* __Notifications__ - Get a direct message in Mattermost when someone mentions you, requests your review, comments on or modifies one of your pull requests/issues, or assigns you on GitHub.
* __Post actions__ - Create a GitHub issue from a post or attach a post message to an issue. Hover over a post to reveal the post actions menu and click **More Actions (...)**.
* __Sidebar buttons__ - Stay up-to-date with how many reviews, unread messages, assignments, and open pull requests you have with buttons in the Mattermost sidebar.
* __Slash commands__ - Interact with the GitHub plugin using the `/github` slash command. Read more about slash commands [here](#slash-commands).

## Before You Start

This guide assumes:

- You have a GitHub account.
- You're a Mattermost System Admin.
- You're running Mattermost v5.12 or higher.

## Configuration

GitHub plugin configuration starts by registering an OAuth app in GitHub and ends in Mattermost. 

**Note:** If you're using GitHub Enterprise, replace all GitHub links below with your GitHub Enterprise URL.

### Step 1: Register an OAuth Application in GitHub

You must first register the Mattermost GitHub Plugin as an authorized OAuth app regardless of whether you're setting up the GitHub plugin as a system admin or a Mattermost user.

1. Go to https://github.com/settings/applications/new to register an OAuth app.
2. Set the following values:
   - **Application name:** `Mattermost GitHub Plugin - <your company name>`
   - **Homepage URL:** `https://github.com/mattermost/mattermost-plugin-github`
   - **Authorization callback URL:** `https://your-mattermost-url.com/plugins/github/oauth/complete`, replacing `https://your-mattermost-url.com` with your Mattermost URL. This value needs to match the Mattermost server URL that you or your users users log in to. 
3. Submit.
4. Click **Generate a new client secret** and provide your GitHub password to continue.
5. Copy the **Client ID** and **Client Secret** in the resulting screen.
6. Click on both **Generate** buttons in `Webhook Secret` and `At Rest Encryption Key`.
7. Once you've successfully registered the Mattermost GitHub Plugin as an authorized OAuth app, switch to Mattermost and run `/github connect` in a Mattermost channel. You should receive a Direct Message from the GitHub plugin about the features available to you.

A System Admin performs the remaining steps:
7. Go to **System Console > Plugins > GitHub** and enter the **GitHub OAuth Client ID** and **GitHub OAuth Client Secret** you copied in a previous step.
8. Hit **Save**.

### Step 2: Create a Webhook in GitHub

As a system admin, you must create a webhook for each organization you want to receive notifications for or subscribe to.

1. In **System Console > Plugins > GitHub**, generate a new value for **Webhook Secret**. Copy it, as you will use it in a later step.
2. Hit **Save** to save the secret.
3. Go to the **Settings** page of your GitHub organization you want to send notifications from, then select **Webhooks** in the sidebar.
4. Click **Add Webhook**.
5. Set the following values:
   - **Payload URL:** `https://your-mattermost-url.com/plugins/github/webhook`, replacing `https://your-mattermost-url.com` with your Mattermost URL.
   - **Content Type:** `application/json`
   - **Secret:** the webhook secret you copied previously.
6. Select **Let me select individual events** for "Which events would you like to trigger this webhook?".
7. Select the following events: `Branch or Tag creation`, `Branch or Tag deletion`, `Issue comments`, `Issues`, `Pull requests`, `Pull request review`, `Pull request review comments`, `Pushes`, `Stars`.
7. Hit **Add Webhook** to save it.

If you have multiple organizations, repeat the process starting from step 3 to create a webhook for each organization.

### Step 3: Configure the Plugin in Mattermost

As a System Admin, if you have an existing Mattermost user account with the name `github`, the plugin will post using the `github` account but without a `BOT` tag.

To prevent this, either:

- Convert the `github` user to a bot account by running `mattermost user convert github --bot` in the CLI.

or

- If the user is an existing user account you want to preserve, change its username and restart the Mattermost server. Once restarted, the plugin will create a bot 
account with the name `github`.

**Note:** For `v0.9.0` and earlier of the GitHub plugin, instead of using bot accounts, set the username the plugin is attached to in **System Console > Plugins > GitHub**.

#### Generate a Key
  
Open **System Console > Plugins > GitHub** and do the following:

1. Generate a new value for **At Rest Encryption Key**.
2. (Optional) **GitHub Organization:** Lock the plugin to a single GitHub organization by setting this field to the name of your GitHub organization.
3. (Optional) **Enable Private Repositories:** Allow the plugin to receive notifications from private repositories by setting this value to `true`.
4. (**Enterprise Only**) **Enterprise Base URL** and **Enterprise Upload URL**: Set these values to your GitHub Enterprise URLs, e.g. `https://github.example.com`. The Base and Upload URLs are often the same. When enabled, existing users must reconnect their accounts to gain access to private repositories. Affected users will be notified by the plugin once private repositories are enabled.
5. Hit **Save**.
6. Go to **System Console > Plugins > Management** and click **Enable** to enable the GitHub plugin.

You're all set!

## Using the Plugin

Once configuration is complete, run the `/github connect` slash command from any channel within Mattermost to connect your Mattermost account with GitHub.

## Onboarding Your Users

When you’ve tested the plugin and confirmed it’s working, notify your team so they can connect their GitHub account to Mattermost and get started. Copy and paste the text below, edit it to suit your requirements, and send it out.

> Hi team, 

> We've set up the Mattermost GitHub plugin, so you can get notifications from GitHub in Mattermost. To get started, run the `/github connect` slash command from any channel within Mattermost to connect your Mattermost account with GitHub. Then, take a look at the [slash commands](#slash-commands) section for details about how to use the plugin.

## Slash Commands

* __Autocomplete slash commands__ - Explore all the available slash commands by typing `/` in the text input box - the autocomplete suggestions help by providing a format example in black text and a short description of the slash command in grey text. Visit the [executing commands](https://docs.mattermost.com/help/messaging/executing-commands.html) documentation for more details.
* __Subscribe to a repository__ - Use `/github subscriptions add` to subscribe a Mattermost channel to receive notifications for new pull requests, issues, branch creation, and more in a GitHub repository.

   - For instance, to post notifications for issues, issue comments, and pull requests matching the label `Help Wanted` from `mattermost/mattermost-server`, use:
   ```
   /github subscriptions add mattermost/mattermost-server --features issues,pulls,issue_comments,label:"Help Wanted"
   ```
  - The following flags are supported:
     - `--features`: comma-delimited list of one or more of: issues, pulls, pulls_merged, pushes, creates, deletes, issue_creations, issue_comments, pull_reviews, label:"labelname". Defaults to pulls,issues,creates,deletes.
     - `--exclude-org-member`: events triggered by organization members will not be delivered. It will be locked to the organization provided in the plugin configuration and it will only work for users whose membership is public. Note that organization members and collaborators are not the same.
     - `--render-style`: notifications will be delivered in the specified style (for example, the body of a pull request will not be displayed). Supported 
     values are `collapsed`, `skip-body` or `default` (same as omitting the flag).
     - `--exclude`: comma-separated list of the repositories to exclude from getting the subscription notifications like `mattermost/mattermost-server`. Only supported for subscriptions to an organization.
   
* __Get to do items__ - Use `/github todo` to get an ephemeral message with items to do in GitHub, including a list of unread messages and pull requests awaiting your review.
* __Update settings__ - Use `/github settings` to update your settings for notifications and daily reminders.
* __And more!__ - Run `/github help` to see what else the slash command can do.

## Frequently Asked Questions

### How do I connect a repository instead of an organization?

Set up your GitHub webhook from the repository instead of the organization. Notifications and subscriptions will then be sent only for repositories you create webhooks for. The reminder and `/github todo` will still search the whole organization, but only list items assigned to you.

### How do I send notifications when a certain label is applied?

Suppose you want to send notifications to a Mattermost channel when `Severity/Critical` label is applied to any issue in the `mattermost/mattermost-plugin-github` repository. Then, use this command to subscribe to these notifications:

```
/github subscriptions add mattermost/mattermost-plugin-github issues,label:"Severity/Critical"
```

### How do I share feedback on this plugin?

Feel free to create a GitHub issue or [join the GitHub Plugin channel on our community Mattermost instance](https://community-release.mattermost.com/core/channels/github-plugin) to discuss.

### How does the plugin save user data for each connected GitHub user?

GitHub user tokens are AES encrypted with an At Rest Encryption Key configured in the plugin's settings page. Once encrypted, the tokens are saved in the `PluginKeyValueStore` table in your Mattermost database.

## Development

This plugin contains both a server and web app portion. Read our documentation about the [Developer Workflow](https://developers.mattermost.com/extend/plugins/developer-workflow/) and [Developer Setup](https://developers.mattermost.com/extend/plugins/developer-setup/) for more information about developing and extending plugins.
