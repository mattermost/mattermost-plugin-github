// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// ***************************************************************
// - [#] indicates a test step (e.g. # Go to a page)
// - [*] indicates an assertion (e.g. * Check the title)
// ***************************************************************

import path from 'node:path';
import fs from 'node:fs';

import {expect, test} from '@e2e-support/test_fixture';
import {UserProfile} from '@mattermost/types/users';

// this is just temporary until we can make the real ouath thing
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


test('/github me', async ({pages, page}) => {
    const c = new pages.ChannelsPage(page);

    // # Run comand
    await c.postMessage('/github me');
    await page.getByTestId('SendMessageButton').click();

    const post = await c.getLastPost();
    const postId = await post.getId();

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
