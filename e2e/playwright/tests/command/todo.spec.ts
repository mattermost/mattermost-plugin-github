// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// ***************************************************************
// - [#] indicates a test step (e.g. # Go to a page)
// - [*] indicates an assertion (e.g. * Check the title)
// ***************************************************************
import {test, expect} from '@e2e-support/test_fixture';

import TodoMessage, {GithubRHSCategory} from '../../support/components/todo_message';
import {messages} from '../../support/constants';
import {getGithubBotDMPageURL, waitForNewMessages} from '../../support/utils';
import {getBotTagFromPost, getPostAuthor} from '../../support/components/post';

const repoRegex = /https:\/\/github.com\/[\w-]+\/[\w-]+/;
const prRegex = /https:\/\/github.com\/[\w-]+\/[\w-]+\/pull\/\d+/;
const issueRegex = /https:\/\/github.com\/[\w-]+\/[\w-]+\/issues\/\d+/;

export default {
    connected: () => {
        test.describe('/github todo command', () => {
            test('from connected account', async ({pages, page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }

                const dmURL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
                await page.goto(dmURL, {waitUntil: 'load'});

                const c = new pages.ChannelsPage(page);

                // # Run todo command
                await c.postMessage('/github todo');
                await c.sendMessage();

                // # Wait for new messages to ensure the last post is the one we want
                await waitForNewMessages(page);

                // # Get last post
                const post = await c.getLastPost();
                const postId = await post.getId();

                const todo = new TodoMessage(post.container);

                // * Assert that titles are there for each section
                // Text are fixed and checked inside todo component handler
                await expect(todo.getTitle(GithubRHSCategory.OPEN_PR)).toBeVisible();
                await expect(todo.getTitle(GithubRHSCategory.ASSIGNMENTS)).toBeVisible();
                await expect(todo.getTitle(GithubRHSCategory.REVIEW_PR)).toBeVisible();
                await expect(todo.getTitle(GithubRHSCategory.UNREAD)).toBeVisible();

                // * Assert that description are there for each section
                // Singular/plurals are not taken into account: ticket separated at https://mattermost.atlassian.net/browse/MM-52416
                // Counters may vary and should be explicitely changed once the test accounts are set
                await expect(todo.getDesc(GithubRHSCategory.OPEN_PR)).toHaveText('You have 1 open pull requests:');
                await expect(todo.getDesc(GithubRHSCategory.ASSIGNMENTS)).toHaveText('You have 19 assignments:');
                await expect(todo.getDesc(GithubRHSCategory.REVIEW_PR)).toHaveText('You have 1 pull requests awaiting your review:');
                await expect(todo.getDesc(GithubRHSCategory.UNREAD)).toHaveText('You have 1 unread messages:');

                // * Assert the open pull request list has 1 items
                const openPr = await todo.getList(GithubRHSCategory.OPEN_PR);
                await expect(openPr.locator('li')).toHaveCount(1);

                // * Assert the open pull request links are correct <REPO> <PR>
                await expect(openPr.locator('li').nth(0).locator('a').nth(0)).toHaveAttribute('href', repoRegex);
                await expect(openPr.locator('li').nth(0).locator('a').nth(1)).toHaveAttribute('href', prRegex);

                // * Assert the review request list has 1 items
                const reviewPr = await todo.getList(GithubRHSCategory.REVIEW_PR);
                await expect(reviewPr.locator('li')).toHaveCount(1);

                // * Assert the pull request links are correct <REPO> <PR>
                await expect(reviewPr.locator('li').nth(0).locator('a').nth(0)).toHaveAttribute('href', repoRegex);
                await expect(reviewPr.locator('li').nth(0).locator('a').nth(1)).toHaveAttribute('href', prRegex);

                // * Assert the assignments list has 1 items
                const assignments = await todo.getList(GithubRHSCategory.ASSIGNMENTS);
                await expect(assignments.locator('li')).toHaveCount(19);

                // * Assert the assignments links are correct <REPO> <ISSUE>
                await expect(assignments.locator('li').nth(0).locator('a').nth(0)).toHaveAttribute('href', repoRegex);
                await expect(assignments.locator('li').nth(0).locator('a').nth(1)).toHaveAttribute('href', issueRegex);

                // * Assert the unread has 1 items
                const unread = await todo.getList(GithubRHSCategory.UNREAD);
                await expect(unread.locator('li')).toHaveCount(1);

                // # Refresh
                await page.reload();

                // * Assert that ephemeral has disappeared
                await expect(page.locator(`#post_${postId}`)).toHaveCount(0);
            });
        });
    },
    unconnected: () => {
        test.describe('/github todo command', () => {
            test('from non connected account', async ({pages, page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }

                const dmURL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
                await page.goto(dmURL, {waitUntil: 'load'});

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
                await expect(post.container.getByText(messages.UNCONNECTED)).toBeVisible();

                // # Refresh
                await page.reload();

                // * Assert that ephemeral has disappeared
                await expect(page.locator(`#post_${postId}`)).toHaveCount(0);
            });
        });
    },
    noSetup: () => {
        test.describe('/github todo command', () => {
            test('before doing setup', async ({pages, page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }

                const dmURL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
                await page.goto(dmURL, {waitUntil: 'load'});

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
