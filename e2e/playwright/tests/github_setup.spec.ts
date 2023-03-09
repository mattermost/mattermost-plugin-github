// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test} from '@e2e-support/test_fixture';
import {UserProfile} from '@mattermost/types/users';

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

// Upload plugin
test.beforeEach(async ({pw, pages, page}) => {
    const {adminClient} = await pw.getAdminClient();
    await adminClient.uploadPluginX('../../dist/github-2.1.2.tar.gz', true);
    await adminClient.enablePlugin('github');
});

// Navigate to GitHub bot DM channel
test.beforeEach(async ({pw, pages, page}) => {
    await page.click('.SidebarLink[aria-label="github"]');
});

test('/github setup', async ({pw, pages, page}) => {
    const c = new pages.ChannelsPage(page);

    const runCommand = (cmd: string) => c.postMessage(cmd).then(() => page.getByTestId('SendMessageButton').click());
    const clickPostAction = async (name: string) => {
        const postElement = await c.getLastPost();
        await postElement.container.getByText(name).last().click();
    };

    await runCommand('/github setup');

    let choices: string[] = [
        'Continue',
        "I'll do it myself",
        'No',
        'Continue',
        'Continue',
    ];

    for (const choice of choices) {
        await clickPostAction(choice);
    }

    await page.getByTestId('client_idinput').fill('text'.repeat(5));
    await page.getByTestId('client_secretinput').fill('text'.repeat(10));

    await page.click('#interactiveDialogSubmit');

    expect(true).toBe(false);
});
