// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// ***************************************************************
// - [#] indicates a test step (e.g. # Go to a page)
// - [*] indicates an assertion (e.g. * Check the title)
// ***************************************************************
import {test, expect} from '@e2e-support/test_fixture';

import {messages} from '../../support/constants';
import {getGithubBotDMPageURL, getPostMessageLocatorId, waitForNewMessages} from '../../support/utils';
import {getBotTagFromPost, getPostAuthor} from '../../support/components/post';
import CreateIssueForm from '../../support/components/github_create_issue_modal_fixture';
import {closeIssue} from '../../support/github_cleanup';

export default {
    connected: () => {
        test.describe('/github issue create', () => {
            test('from connected account', async ({pages, page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }

                const URL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
                await page.goto(URL, {waitUntil: 'load'});
                await page.waitForTimeout(5000);

                const c = new pages.ChannelsPage(page);

                // # Run create command
                await c.postMessage('/github issue create');
                await c.sendMessage();
                await page.waitForTimeout(500);

                const form = new CreateIssueForm(page);

                // * Check that Create Issue modal is shown
                await expect(form.header).toBeVisible();

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
                const issueNum = text.substring(1);
                const issueUrlMatch = `https://github.com/${repoName}/issues/${issueNum}`;
                expect(href).toEqual(issueUrlMatch);

                const owner = 'MM-Github-Testorg';
                const repo = 'testrepo';
                const issueNumber = parseInt(issueNum, 10);
                await closeIssue(owner, repo, issueNumber);
            });
        });
    },
    unconnected: () => {
        test.describe('/github issue create', () => {
            test('from non connected account', async ({pages, page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }

                const URL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
                await page.goto(URL, {waitUntil: 'load'});

                const c = new pages.ChannelsPage(page);

                // # Run create command
                await c.postMessage('/github issue create');
                await c.sendMessage();

                // # Wait for new messages to ensure the last post is the one we want
                await waitForNewMessages(page);

                // # Get last post
                const post = await c.getLastPost();
                const postId = await post.getId();

                // * Verify that message is sent by the github bot
                await expect(getPostAuthor(post)).toHaveText('github');
                await expect(getBotTagFromPost(post)).toBeVisible();

                // * assert failure message
                await expect(post.container.getByText(messages.UNCONNECTED)).toBeVisible();

                // # Refresh
                await page.reload();

                // * Assert that ephemeral has disappeared
                await expect(page.locator(`#post_${postId}`)).toHaveCount(0);
            });
        });
    },
    noSetup: () => {
        test.describe('/github issue create', () => {
            test('before doing setup', async ({pages, page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }

                const dmURL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
                await page.goto(dmURL, {waitUntil: 'load'});
                await page.waitForTimeout(5000);

                const c = new pages.ChannelsPage(page);

                // # Run todo command
                await c.postMessage('/github todo');
                await c.sendMessage();

                // # Wait for new messages to ensure the last post is the one we want
                await waitForNewMessages(page);

                // # Get last post
                const post = await c.getLastPost();
                const postId = await post.getId();

                // * Verify that message is sent by the github bot
                await expect(getPostAuthor(post)).toHaveText('github');
                await expect(getBotTagFromPost(post)).toBeVisible();

                // * assert failure message
                await expect(post.container.getByText(messages.NOSETUP)).toBeVisible();

                // # Refresh
                await page.reload();

                // * Assert that ephemeral has disappeared
                await expect(page.locator(`#post_${postId}`)).toHaveCount(0);
            });
        });
    },
};
