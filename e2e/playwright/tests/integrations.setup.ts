// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {test as setup} from '@e2e-support/test_fixture';

const authFile = __dirname + '/../.auth-user.json';

setup('authenticate', async ({page, pages, pw}) => {
    const {adminClient, adminUser} = await pw.getAdminClient();
    if (adminUser === null) {
        throw new Error('authenticate: can not get adminUser');
    }
    const config = await adminClient.getConfig();
    const login = new pages.LoginPage(page, config);
    await login.goto();
    await login.login(adminUser);
    await page.context().storageState({path: authFile});
});
