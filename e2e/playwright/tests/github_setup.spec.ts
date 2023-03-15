// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// ***************************************************************
// - [#] indicates a test step (e.g. # Go to a page)
// - [*] indicates an assertion (e.g. * Check the title)
// ***************************************************************

import path from 'node:path';

import {expect, test} from '@e2e-support/test_fixture';

import './init_test';

const SCREENSHOTS_DIR = path.join(__dirname, '../screenshots');

test('/github setup', async ({pw, pages, page, context}) => {
    const c = new pages.ChannelsPage(page);

    // Utility functions similar to this should be shared from mm-webapp.

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

    // # Run setup command
    await postMessage('/github connect');

    // # go to github bot DM channel
    await page.locator('.SidebarChannelGroup_content').getByText('github').click();

    // # Go through prompts of setup flow
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

    // # Fill out interactive dialog for GitHub client id and client secret
    await page.getByTestId('client_idinput').fill('text'.repeat(5));
    await page.getByTestId('client_secretinput').fill('text'.repeat(10));
    await page.click('#interactiveDialogSubmit');

    const post = await c.getLastPost();
    const postId = await post.getId();

    const locatorId = `#post_${postId} .attachment__body`;
    const text = await page.locator(locatorId).innerText();
    expect(text).toEqual('Go here to connect your account.');

    // Not asserting anything here, though this can be used as an artifact in CI.
    await screenshot('github_setup/show_connect_link.png');

    // * Verify connect link has correct URL
    const siteURL = await getSiteURL();
    const expectedConnectLinkURL = `${siteURL}/plugins/github/oauth/connect`;

    const connectLinkLocator = `${locatorId} a`;
    const href = await page.locator(connectLinkLocator).getAttribute('href');
    expect(href).toEqual(expectedConnectLinkURL);
});
