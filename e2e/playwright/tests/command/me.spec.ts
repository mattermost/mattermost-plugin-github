// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// ***************************************************************
// - [#] indicates a test step (e.g. # Go to a page)
// - [*] indicates an assertion (e.g. * Check the title)
// ***************************************************************

import {expect, test} from '@e2e-support/test_fixture';

import {messages} from '../../support/constants';
import {getGithubBotDMPageURL, waitForNewMessages} from '../../support/utils';
import {
    getBotTagFromPost,
    getPostAuthor,
} from '../../support/components/post';

const mmGithubHandle = 'MM-Github-Plugin';

export default {
    connected: () => {
        test.describe('/github me', () => {
            test('from connected account', async ({pages, page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }

                const dmURL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
                await page.goto(dmURL, {waitUntil: 'load'});

                const c = new pages.ChannelsPage(page);

                // # Run comand
                await c.postMessage('/github me');
                await c.sendMessage();

                // # Wait for new messages to ensure the last post is the one we want
                await waitForNewMessages(page);

                // # Get last post
                const post = await c.getLastPost();
                const postId = await post.getId();

                // * Verify that message is sent by the github bot
                await expect(getPostAuthor(post)).toHaveText('github');
                await expect(getBotTagFromPost(post)).toBeVisible();

                // * assert intro message
                await expect(post.container.getByText('You are connected to Github as')).toBeVisible();

                // * check username
                await expect(post.container.getByRole('link', {name: mmGithubHandle})).toBeVisible();

                // * check profile image
                await expect(post.container.getByRole('heading').locator('img')).toBeVisible();

                // # Refresh
                await page.reload();

                // * Assert that ephemeral has disappeared
                await expect(page.locator(`#post_${postId}`)).toHaveCount(0);
            });
        });
    },
    unconnected: () => {
        test.describe('/github me', () => {
            test('from non connected account', async ({pages, page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }

                const dmURL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
                await page.goto(dmURL, {waitUntil: 'load'});

                const c = new pages.ChannelsPage(page);

                // # Run comand
                await c.postMessage('/github me');
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
        test.describe('/github me', () => {
            test('before doing setup', async ({pages, page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }

                const dmURL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
                await page.goto(dmURL, {waitUntil: 'load'});

                const c = new pages.ChannelsPage(page);

                // # Run comand
                await c.postMessage('/github me');
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
