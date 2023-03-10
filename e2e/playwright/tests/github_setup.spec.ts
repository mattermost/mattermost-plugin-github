// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import path from 'node:path';
import fs from 'node:fs';

import {expect, test} from '@e2e-support/test_fixture';
import {UserProfile} from '@mattermost/types/users';

const SCREENSHOTS_DIR = path.join(__dirname, '../screenshots');

// Log in
test.beforeEach(async ({pw, pages, page}) => {
    const {adminClient} = await pw.getAdminClient();
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

    const user = {
        username: 'sysadmin',
        password: 'Sys@dmin-sample1',
    } as UserProfile;
    await loginPage.login(user);
});

// utility function
const getPluginBundlePath = async (): Promise<string> => {
    const dir = path.join(__dirname, '../../../dist');
    const files = await fs.promises.readdir(dir);
    const bundle = files.find((fname) => fname.endsWith('.tar.gz'));
    if (!bundle) {
        throw new Error('Failed to find plugin bundle in dist folder');
    }

    return path.join(dir, bundle);
}

// Upload plugin
test.beforeEach(async ({pw}) => {
    const bundlePath = await getPluginBundlePath();
    const {adminClient} = await pw.getAdminClient();

    await adminClient.uploadPluginX(bundlePath, true);
    await adminClient.enablePlugin('github');
});

// Navigate to GitHub bot DM channel
test.beforeEach(async ({page}) => {
    await page.click('.SidebarLink[aria-label="github"]');
});

test('/github setup', async ({pw, pages, page, context}) => {
    const c = new pages.ChannelsPage(page);

    // utility functions similar to this should be shared from mm-webapp.
    // utility function
    const postMessage = async (command: string) => {
        await c.postMessage(command);
        await page.getByTestId('SendMessageButton').click();
    };

    // utility function
    const clickPostAction = async (name: string) => {
        const postElement = await c.getLastPost();
        await postElement.container.getByText(name).last().click();
    };

    // utility function
    const getSiteURL = async (): Promise<string> => {
        const {adminClient} = await pw.getAdminClient();
        const config = await adminClient.getConfig();
        return config.ServiceSettings.SiteURL;
    }

    // utility function
    const screenshot = async (name: string) => {
        await page.screenshot({path: path.join(SCREENSHOTS_DIR, name)});
    }

    // ---- TEST ----

    // run setup command
    await postMessage('/github setup');

    // go through prompts of setup flow
    const choices: string[] = [
        'Continue',
        "I'll do it myself",
        'No',
        'Continue',
        'Continue',
    ];

    for (const choice of choices) {
        await clickPostAction(choice);
    }

    // fill out interactive dialog for GitHub client id and client secret
    await page.getByTestId('client_idinput').fill('text'.repeat(5));
    await page.getByTestId('client_secretinput').fill('text'.repeat(10));
    await page.click('#interactiveDialogSubmit');

    const post = await c.getLastPost();
    const postId = await post.getId();

    const locatorId = `#post_${postId} .attachment__body`;
    const text = await page.locator(locatorId).innerText();
    expect(text).toEqual('Go here to connect your account.');

    // not actually asserting anything here, though this could be used as an artifact in CI.
    await screenshot('github_setup/show_connect_link.png');

    // verify connect link has correct URL
    const siteURL = await getSiteURL();
    const expectedConnectLinkURL = `${siteURL}/plugins/github/oauth/connect`;

    const connectLinkLocator = `${locatorId} a`;
    const href = await page.locator(connectLinkLocator).getAttribute('href');
    expect(href).toEqual(expectedConnectLinkURL);
});
