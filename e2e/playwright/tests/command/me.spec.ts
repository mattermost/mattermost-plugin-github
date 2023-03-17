// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// ***************************************************************
// - [#] indicates a test step (e.g. # Go to a page)
// - [*] indicates an assertion (e.g. * Check the title)
// ***************************************************************

import {expect, test} from '@e2e-support/test_fixture';
import { ChannelsPost } from '@e2e-support/ui/components';
import {UserProfile} from '@mattermost/types/users';
import {messages} from '../../support/constants';
import {getBotTagFromPost, getPostAuthor} from '../../support/components/post';

// TODO: this is just temporary until we can make the real ouath thing
const mmUsername = process.env.PW_MM_USERNAME;
const mmPassword = process.env.PW_MM_PASSWORD;
const mmGithubHandle = process.env.PW_MM_GITHUB_HANDLE;

// # Log in as user
test.beforeEach(async ({pw, pages, page}) => {
    const {adminClient, adminUser} = await pw.getAdminClient();
    if (!adminUser) {
        throw new Error("Failed to get admin user");
    }
    await adminClient.patchConfig({
        ServiceSettings: {
            EnableTutorial: false,
            EnableOnboardingFlow: false,
        },
    });

    const adminConfig = await adminClient.getConfig();
    const loginPage = new pages.LoginPage(page, adminConfig);

    await loginPage.goto();
    await loginPage.toBeVisible();
    await loginPage.login({username: mmUsername, password: mmPassword} as UserProfile);
});



// TODO: all tests are not run at the same time since user is hardcoded from ENV
// As soon as we plug this with real setup connect, the test should include those steps and remove the skip
test.describe('/github me', () => {

    test.only('from connected account', async ({pages, page}) => {
        const c = new pages.ChannelsPage(page);

        // # Run comand
        await c.postMessage('/github me');
        await c.sendMessage();

        // # Get last post
        const post = await c.getLastPost();
        const postId = await post.getId();

        // * Verify that message is sent by the github bot
        await expect(getPostAuthor(post)).toHaveText('github');
        await expect(getBotTagFromPost(post)).toBeVisible();

        // * assert intro message
        await expect(post.container.getByText('You are connected to Github as')).toBeVisible()
        // * check username
        await expect(post.container.getByRole('link', { name: mmGithubHandle })).toBeVisible();
        // * check profile image
        await expect(post.container.getByRole('heading').locator('img')).toBeVisible();

        // # Refresh
        await page.reload();

        // * Assert that ephemeral has disappeared
        await expect(page.locator(`#post_${postId}`)).toHaveCount(0);
    });

    test('from non connected account', async ({pages, page}) => {
        const c = new pages.ChannelsPage(page);

        // # Run comand
        await c.postMessage('/github me');
        await c.sendMessage();

        // # Get last post
        const post = await c.getLastPost();
        const postId = await post.getId();

        // * Verify that message is sent by the github bot
        await expect(getPostAuthor(post)).toHaveText('github');
        await expect(getBotTagFromPost(post)).toBeVisible();

        // * assert failure message
        await expect(post.container.getByText(messages.UNCONNECTED)).toBeVisible()

        // # Refresh
        await page.reload();

        // * Assert that ephemeral has disappeared
        await expect(page.locator(`#post_${postId}`)).toHaveCount(0);
    });

    test('before doing setup', async ({pages, page}) => {
        const c = new pages.ChannelsPage(page);

        // # Run comand
        await c.postMessage('/github me');
        await c.sendMessage();

        // # Get last post
        const post = await c.getLastPost();
        const postId = await post.getId();

        // * Verify that message is sent by the github bot
        await expect(getPostAuthor(post)).toHaveText('github');
        await expect(getBotTagFromPost(post)).toBeVisible();

        // * assert failure message
        await expect(post.container.getByText(messages.NOSETUP)).toBeVisible()

        // # Refresh
        await page.reload();

        // * Assert that ephemeral has disappeared
        await expect(page.locator(`#post_${postId}`)).toHaveCount(0);

    });
});
