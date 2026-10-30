// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Client4} from '@mattermost/client';
import {testConfig} from '@mattermost/playwright-lib';

export async function loginAdmin(): Promise<Client4> {
    const client = new Client4();
    client.setUrl(testConfig.baseURL);
    await client.login(testConfig.adminUsername, testConfig.adminPassword);
    return client;
}
