// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// ***************************************************************
// - [#] indicates a test step (e.g. # Go to a page)
// - [*] indicates an assertion (e.g. * Check the title)
// ***************************************************************

import {expect, test} from '@e2e-support/test_fixture';
import {Page} from '@playwright/test';

import CreateIssueForm from '../support/components/github_create_issue_modal_fixture';

import '../support/init_test';

import {
    fillTextField,
    postMessage,
    submitDialog,
    clickPostAction,
    screenshot,
    getSlackAttachmentLocatorId,
    getPostMessageLocatorId,
    DEFAULT_WAIT_MILLIS,
} from '../support/utils';

const GITHUB_CONNECT_LINK = '/plugins/github/oauth/connect';
const TEST_CLIENT_ID = 'a'.repeat(20);
const TEST_CLIENT_SECRET = 'b'.repeat(40);

test('/github setup', async ({pw, pages, page: originalPage}) => {
    // # Log in
    const {adminUser} = await pw.getAdminClient();
    const {page} = await pw.testBrowser.login(adminUser);
    await originalPage.close();

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
        await page.waitForTimeout(DEFAULT_WAIT_MILLIS);
        await screenshot(`post_action_before_${i}`, page);
        await clickPostAction(choice, c);
        await screenshot(`post_action_after_${i}`, page);
    }

    // # Fill out interactive dialog for GitHub client id and client secret
    await fillTextField('client_id', TEST_CLIENT_ID, page);
    await fillTextField('client_secret', TEST_CLIENT_SECRET, page);
    await submitDialog(page);

    await page.waitForTimeout(DEFAULT_WAIT_MILLIS);

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
    await page.close();
});

test('/github connect', async ({pw, pages, page: originalPage}) => {
    // # Log in
    const {adminUser} = await pw.getAdminClient();
    const {page} = await pw.testBrowser.login(adminUser);
    await originalPage.close();

    // # Navigate to Channels
    const c = new pages.ChannelsPage(page);
    await c.goto();

    // # Run connect command
    await postMessage('/github connect', c, page);
    await page.waitForTimeout(DEFAULT_WAIT_MILLIS);

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
    await page.waitForTimeout(DEFAULT_WAIT_MILLIS);

    post = await c.getLastPost();
    postId = await post.getId();
    locatorId = getPostMessageLocatorId(postId);

    text = await page.locator(locatorId).innerText();
    expect(text).toContain('Welcome to the Mattermost GitHub Plugin!');

    await screenshot('github_connect/after_navigate_to_github_plugin', page);
    await page.close();
});

test('/github issue create', async ({pw, pages, page: originalPage}) => {
    // # Log in
    const {adminUser} = await pw.getAdminClient();
    const {page} = await pw.testBrowser.login(adminUser);
    await originalPage.close();

    // # Navigate to Channels
    const c = new pages.ChannelsPage(page);
    await c.goto();

    // # Run create command
    await postMessage('/github issue create', c, page);
    await page.waitForTimeout(DEFAULT_WAIT_MILLIS);

    await screenshot('github_issue_create/ran_create_command', page);

    const form = new CreateIssueForm(page);

    // * Check that Create Issue modal is shown
    await expect(form.header).toBeVisible();

    await page.pause();

    const repoSearch = 'MM-Github-Testorg';
    const repoName = 'MM-Github-Testorg/testrepo';
    await form.selectRepo(repoSearch, repoName);

    // # Select labels
    await form.selectLabels(['enhancement']);

    // # Select assignees
    await form.selectAssignees(['MM-Github-Plugin']);

    // # Issue title
    await form.issueTitle.fill('The title');

    // # Issue description
    await form.issueDescription.fill('My description');

    await screenshot('github_issue_create/form_filled_out', page);

    // # Submit form
    await form.submit();

    const post = await c.getLastPost();

    await screenshot('github_issue_create/after_form_submission', page);

    const postTextMatch = 'Created GitHub issue';
    await expect(post.container.getByText(postTextMatch)).toBeVisible();

    const postId = await post.getId();
    const locatorId = getPostMessageLocatorId(postId);

    // * Verify the issue URL is for the right repository
    const issueLinkSelector = `${locatorId} a`;
    const issueLinkLocator = page.locator(issueLinkSelector);
    const href = await issueLinkLocator.getAttribute('href');
    const text = await issueLinkLocator.innerText();

    // [#4](https://github.com/MM-Github-Testorg/testrepo/issues/4)
    expect(text[0]).toEqual('#');
    const issueUrlMatch = `https://github.com/${repoName}/issues/${text.substring(1)}`;
    expect(href).toEqual(issueUrlMatch);


    await page.close();
});

const parseMarkdownLink = (link: string) => {
    if (!(link.startsWith('[') && link.includes('](') && link.endsWith(')'))) {
        throw new Error(`Invalid markdown link to parse: "${link}"`);
    }

    const parts = link.split('](');
    const label = parts[0].substring(1);
}
