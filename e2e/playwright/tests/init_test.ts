// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import path from 'node:path';
import fs from 'node:fs';

import {test} from '@e2e-support/test_fixture';

// # Log in
test.beforeEach(async ({pw, pages, page}) => {
    const {adminClient, adminUser} = await pw.getAdminClient();
    if (!adminUser) {
        throw new Error('Failed to get admin user');
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
    await loginPage.login(adminUser);
});

const getPluginBundlePath = async (): Promise<string> => {
    const dir = path.join(__dirname, '../../../dist');
    const files = await fs.promises.readdir(dir);
    const bundle = files.find((fname) => fname.endsWith('.tar.gz'));
    if (!bundle) {
        throw new Error('Failed to find plugin bundle in dist folder');
    }

    return path.join(dir, bundle);
}

// # Upload plugin
test.beforeEach(async ({pw}) => {
    const bundlePath = await getPluginBundlePath();
    const {adminClient} = await pw.getAdminClient();

    await adminClient.uploadPluginX(bundlePath, true);
    await adminClient.enablePlugin('github');
});
