// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {baseGlobalSetup} from '@mattermost/playwright-lib';

import {loginAdmin} from '../support/admin';
import {installPlugin} from '../support/plugin';

import {missingRealGitHubEnv, realGitHub} from './env';
import {frontLoad} from './fixtures';

export default async function globalSetup() {
    const missing = missingRealGitHubEnv();
    if (missing.length) {
        throw new Error(`Set ${missing.join(', ')} in e2e/integration/.env (see .env.example). Do not paste the values into chat.`);
    }
    const state = await frontLoad();
    await baseGlobalSetup();

    const adminClient = await loginAdmin();
    await installPlugin(adminClient, {
        githuboauthclientid: realGitHub.oauth.clientId,
        githuboauthclientsecret: realGitHub.oauth.clientSecret,
        webhooksecret: state.webhookSecret,
        githuborg: realGitHub.org,
        enableprivaterepo: true,
    });
}
