// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// ***************************************************************
// - [#] indicates a test step (e.g. # Go to a page)
// - [*] indicates an assertion (e.g. * Check the title)
// ***************************************************************

import {expect, test} from '@e2e-support/test_fixture';

import '../support/init_test';

import {sleep, fillTextField, postMessage, submitDialog, clickPostAction, screenshot, navigateToChannel, getSlackAttachmentLocatorId, getPostMessageLocatorId} from '../support/utils';

const GITHUB_CONNECT_LINK = '/plugins/github/oauth/connect';
const TEST_CLIENT_ID = 'aaaaaaaaaaaaaaaaaaaa';
const TEST_CLIENT_SECRET = 'bbbbbbbbbbbbbbbbbbbbcccccccccccccccccccc';

test('/github setup', async ({pages, page}) => {
    const c = new pages.ChannelsPage(page);

    // # Run setup command
    await postMessage('/github setup', c, page);

    // # Go to github bot DM channel
    await navigateToChannel('github', page);

    // # Go through prompts of setup flow
    let choices: string[] = [
        'Continue',
        "I'll do it myself",
        'No',
        'Continue',
        'Continue',
    ];

    for (const choice of choices) {
        await clickPostAction(choice, c);
    }

    // # Fill out interactive dialog for GitHub client id and client secret
    await fillTextField('client_id', TEST_CLIENT_ID, page);
    await fillTextField('client_secret', TEST_CLIENT_SECRET, page);
    await submitDialog(page);

    await sleep();

    const post = await c.getLastPost();
    const postId = await post.getId();
    const locatorId = getSlackAttachmentLocatorId(postId);

    const text = await page.locator(locatorId).innerText();
    expect(text).toEqual('Go here to connect your account.');

    await screenshot('github_setup/show_connect_link.png', page);

    // * Verify connect link has correct URL
    const connectLinkLocator = `${locatorId} a`;
    const href = await page.locator(connectLinkLocator).getAttribute('href');
    expect(href).toMatch(GITHUB_CONNECT_LINK);

    await page.click(connectLinkLocator);

    // # Say no to "Create a webhook"
    await clickPostAction('No', c);

    // # Say no to "Broadcast to channel"
    await clickPostAction('Not now', c);

    await screenshot('github_setup/done.png', page);
});

test('/github connect', async ({pages, page}) => {
    const c = new pages.ChannelsPage(page);

    // # Run connect command
    await postMessage('/github connect', c, page);
    await sleep();

    let post = await c.getLastPost();
    let postId = await post.getId();
    let locatorId = getPostMessageLocatorId(postId);

    let text = await page.locator(locatorId).innerText();
    expect(text).toEqual('Click here to link your GitHub account.');

    await screenshot('github_connect/show_connect_link.png', page);

    // * Verify connect link has correct URL
    const connectLinkLocator = `${locatorId} a`;
    const href = await page.locator(connectLinkLocator).getAttribute('href');
    expect(href).toMatch(GITHUB_CONNECT_LINK);

    await page.click(connectLinkLocator);
    await screenshot('github_connect/after_clicking_connect_link.png', page);

    // # Go to github bot DM channel
    await navigateToChannel('github', page)
    await sleep();

    post = await c.getLastPost();
    postId = await post.getId();
    locatorId = getPostMessageLocatorId(postId);

    text = await page.locator(locatorId).innerText();
    expect(text).toContain('Welcome to the Mattermost GitHub Plugin!');

    await screenshot('github_connect/after_navigate_to_github_plugin.png', page);
});

test('/github issue create', async ({pages, page}) => {
    const c = new pages.ChannelsPage(page);

    // # Run create command
    await postMessage('/github issue create', c, page);
    await sleep();

    await screenshot('github_issue_create/ran_create_command.png', page);

    // * Check that Create Issue modal is shown
    await expect(page.getByRole('heading', {
        name: 'Create GitHub Issue'
    })).toBeVisible();

    // await page.pause();
});