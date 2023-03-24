// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// ***************************************************************
// - [#] indicates a test step (e.g. # Go to a page)
// - [*] indicates an assertion (e.g. * Check the title)
// ***************************************************************

import {expect, test} from '@e2e-support/test_fixture';

import '../support/init_test';

import {sleep, fillTextField, postMessage, submitDialog, clickPostAction, screenshot, getSlackAttachmentLocatorId, getPostMessageLocatorId} from '../support/utils';

const GITHUB_CONNECT_LINK = '/plugins/github/oauth/connect';
const TEST_CLIENT_ID = 'aaaaaaaaaaaaaaaaaaaa';
const TEST_CLIENT_SECRET = 'bbbbbbbbbbbbbbbbbbbbcccccccccccccccccccc';

test('/github setup', async ({pw, pages}) => {
    // # Log in
    const {adminUser} = await pw.getAdminClient();
    const {page} = await pw.testBrowser.login(adminUser);

    // # Navigate to Channels
    const c = new pages.ChannelsPage(page);
    await c.goto();

    // # Run setup command
    await postMessage('/github setup', c, page);

    // # Go to github bot DM channel
    const teamName = page.url().split('/')[3];
    await c.goto(teamName, 'messages/@github');

    // # Go through prompts of setup flow
    let choices: string[] = [
        'Continue',
        "I'll do it myself",
        'No',
        'Continue',
        'Continue',
    ];

    let i = 0;
    for (const choice of choices) {
        i++;
        await screenshot(`post_action_before_${i}`, page);
        await clickPostAction(choice, c);
        await screenshot(`post_action_after_${i}`, page);
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

    await screenshot('github_setup/show_connect_link', page);

    // * Verify connect link has correct URL
    const connectLinkLocator = `${locatorId} a`;
    const href = await page.locator(connectLinkLocator).getAttribute('href');
    expect(href).toMatch(GITHUB_CONNECT_LINK);

    await screenshot(`connect_click_before`, page);
    await page.click(connectLinkLocator);
    await screenshot(`connect_click_after`, page);

    // # Say no to "Create a webhook"
    await screenshot(`webhook_question_before`, page);
    await clickPostAction('No', c);
    await screenshot(`webhook_question_aftrt`, page);

    // # Say no to "Broadcast to channel"
    await screenshot(`broadcast_question_before`, page);
    await clickPostAction('Not now', c);
    await screenshot(`broadcast_question_after`, page);

    await screenshot('github_setup/done', page);
});

test('/github connect', async ({pw, pages}) => {
    // # Log in
    const {adminUser} = await pw.getAdminClient();
    const {page} = await pw.testBrowser.login(adminUser);

    // # Navigate to Channels
    const c = new pages.ChannelsPage(page);
    await c.goto();

    // # Run connect command
    await postMessage('/github connect', c, page);
    await sleep();

    let post = await c.getLastPost();
    let postId = await post.getId();
    let locatorId = getPostMessageLocatorId(postId);

    let text = await page.locator(locatorId).innerText();
    expect(text).toEqual('Click here to link your GitHub account.');

    await screenshot('github_connect/show_connect_link', page);

    // * Verify connect link has correct URL
    const connectLinkLocator = `${locatorId} a`;
    const href = await page.locator(connectLinkLocator).getAttribute('href');
    expect(href).toMatch(GITHUB_CONNECT_LINK);

    await page.click(connectLinkLocator);
    await screenshot('github_connect/after_clicking_connect_link', page);

    // # Go to github bot DM channel
    const teamName = page.url().split('/')[3];
    await c.goto(teamName, 'messages/@github');
    await sleep();

    post = await c.getLastPost();
    postId = await post.getId();
    locatorId = getPostMessageLocatorId(postId);

    text = await page.locator(locatorId).innerText();
    expect(text).toContain('Welcome to the Mattermost GitHub Plugin!');

    await screenshot('github_connect/after_navigate_to_github_plugin', page);
});

test('/github issue create', async ({pw, pages}) => {
    // # Log in
    const {adminUser} = await pw.getAdminClient();
    const {page} = await pw.testBrowser.login(adminUser);

    // # Navigate to Channels
    const c = new pages.ChannelsPage(page);
    await c.goto();

    // # Run create command
    await postMessage('/github issue create', c, page);
    await sleep();

    await screenshot('github_issue_create/ran_create_command', page);

    // * Check that Create Issue modal is shown
    await expect(page.getByRole('heading', {
        name: 'Create GitHub Issue'
    })).toBeVisible();

    // await page.pause();
});
