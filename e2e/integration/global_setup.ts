// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {baseGlobalSetup} from '@mattermost/playwright-lib';

import {startFakeGitHub} from './fake-github/server';
import {loginAdmin} from './support/admin';
import {fakeGitHubPort, fakeGitHubURL, oauthApp} from './support/constants';
import {installPlugin} from './support/plugin';

export default async function globalSetup() {
    await baseGlobalSetup();

    const fake = await startFakeGitHub({port: fakeGitHubPort, publicURL: fakeGitHubURL, app: oauthApp});

    const adminClient = await loginAdmin();
    await installPlugin(adminClient);

    return fake.close;
}
