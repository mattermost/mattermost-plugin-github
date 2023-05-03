// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// ***************************************************************
// - [#] indicates a test step (e.g. # Go to a page)
// - [*] indicates an assertion (e.g. * Check the title)
// ***************************************************************

import {expect, test} from '@e2e-support/test_fixture';
import {Page} from '@playwright/test';

import CreateIssueForm from '../support/components/github_create_issue_modal_fixture';

import {fillTextField, postMessage, submitDialog, clickPostAction, getGithubBotDMPageURL, getSlackAttachmentLocatorId, getPostMessageLocatorId, waitForNewMessages} from '../support/utils';

const GITHUB_CONNECT_LINK = '/plugins/github/oauth/connect';
const TEST_CLIENT_ID = 'a'.repeat(20);
const TEST_CLIENT_SECRET = 'b'.repeat(40);

export default {
    setup: () => {
        test('/github setup', async ({pw, page, pages}) => {
            const {adminClient, adminUser} = await pw.getAdminClient();
            if (adminUser === null) {
                throw new Error('can not get adminUser');
            }

            const URL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
            await page.goto(URL, {waitUntil: 'load'});

            const c = new pages.ChannelsPage(page);

            // # Run setup command
            await postMessage('/github setup', c, page);

            // # Wait for new messages to ensure the last post is the one we want
            // await waitForNewMessages(page);
            await page.waitForTimeout(1000);

            // # Go through prompts of setup flow
            await clickPostAction('Continue', c, page);
            await clickPostAction("I'll do it myself", c, page);
            await clickPostAction('No', c, page);
            await clickPostAction('Continue', c, page);
            await clickPostAction('Continue', c, page);

            // # Fill out interactive dialog for GitHub client id and client secret
            await fillTextField('client_id', TEST_CLIENT_ID, page);
            await fillTextField('client_secret', TEST_CLIENT_SECRET, page);
            await submitDialog(page);

            await page.waitForTimeout(500);

            const post = await c.getLastPost();
            const postId = await post.getId();
            const locatorId = getSlackAttachmentLocatorId(postId);

            const text = await page.locator(locatorId).innerText();
            expect(text).toEqual('Go here to connect your account.');

            // * Verify connect link has correct URL
            const connectLinkLocator = `${locatorId} a`;
            const href = await page.
                locator(connectLinkLocator).
                getAttribute('href');
            expect(href).toMatch(GITHUB_CONNECT_LINK);

            await page.click(connectLinkLocator);

            // # Say no to "Create a webhook"
            await clickPostAction('No', c, page);

            // # Say no to "Broadcast to channel"
            await clickPostAction('Not now', c, page);
        });
    },
    connect: () => {
        test('/github connect', async ({pages, page, pw}) => {
            const {adminClient, adminUser} = await pw.getAdminClient();
            if (adminUser === null) {
                throw new Error('can not get adminUser');
            }

            const URL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
            await page.goto(URL, {waitUntil: 'load'});

            const c = new pages.ChannelsPage(page);

            // # Run connect command
            await postMessage('/github connect', c, page);

            // # Wait for new messages to ensure the last post is the one we want
            await waitForNewMessages(page);

            let post = await c.getLastPost();
            let postId = await post.getId();
            let locatorId = getPostMessageLocatorId(postId);

            let text = await page.locator(locatorId).innerText();
            expect(text).toEqual('Click here to link your GitHub account.');

            // * Verify connect link has correct URL
            const connectLinkLocator = `${locatorId} a`;
            const href = await page.locator(connectLinkLocator).getAttribute('href');
            expect(href).toMatch(GITHUB_CONNECT_LINK);

            await page.click(connectLinkLocator);
            await page.waitForTimeout(2000);

            post = await c.getLastPost();
            postId = await post.getId();
            locatorId = getPostMessageLocatorId(postId);

            text = await page.locator(locatorId).innerText();
            expect(text).toContain('Welcome to the Mattermost GitHub Plugin!');
        });
    },
    disconnect: () => {
        test('/github disconnect', async ({pages, page, pw}) => {
            const {adminClient, adminUser} = await pw.getAdminClient();
            if (adminUser === null) {
                throw new Error('can not get adminUser');
            }

            const URL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
            await page.goto(URL, {waitUntil: 'load'});

            const c = new pages.ChannelsPage(page);

            // # Run connect command
            await postMessage('/github disconnect', c, page);

            // # Wait for new messages to ensure the last post is the one we want
            await waitForNewMessages(page);

            const post = await c.getLastPost();
            const postId = await post.getId();
            const locatorId = getPostMessageLocatorId(postId);
            const text = await page.locator(locatorId).innerText();
            await expect(text).toContain('Disconnected your GitHub account');
        });
    },
    create: () => {
        test('/github issue create', async ({pages, page, pw}) => {
            const {adminClient, adminUser} = await pw.getAdminClient();
            if (adminUser === null) {
                throw new Error('can not get adminUser');
            }

            const URL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
            await page.goto(URL, {waitUntil: 'load'});

            const c = new pages.ChannelsPage(page);

            // # Run create command
            await postMessage('/github issue create', c, page);
            await page.waitForTimeout(500);

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

            // # Submit form
            await form.submit();

            const post = await c.getLastPost();

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
    }
};
