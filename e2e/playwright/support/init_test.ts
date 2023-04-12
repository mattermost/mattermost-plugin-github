// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import path from 'node:path';
import fs from 'node:fs';

import {test} from '@e2e-support/test_fixture';
import {cleanUpBotDMs} from './utils';

import {clearKVStoreForPlugin} from './kv';
import {DeepPartial} from '@mattermost/types/utilities';
import {AdminConfig} from '@mattermost/types/config';

import {runOAuthServer} from './init_mock_oauth_server';

const pluginDistPath = path.join(__dirname, '../../../dist');
const pluginId = 'github';

// # Clear plugin's KV store
test.beforeAll(async () => {
    if (process.env.AVOID_TEST_CLEANUP === 'true') {
        return;
    }

    await clearKVStoreForPlugin(pluginId);
});

// # Run OAuth server
test.beforeAll(async () => {
    await runOAuthServer();
});

// # Upload plugin
test.beforeEach(async ({pw}) => {
    const files = await fs.promises.readdir(pluginDistPath);
    const bundle = files.find((fname) => fname.endsWith('.tar.gz'));
    if (!bundle) {
        throw new Error('Failed to find plugin bundle in dist folder');
    }

    const bundlePath = path.join(pluginDistPath, bundle);
    const {adminClient} = await pw.getAdminClient();

    await adminClient.uploadPluginX(bundlePath, true);
    await adminClient.enablePlugin(pluginId);
});

// # Clear bot DM channel
test.beforeEach(async ({pw}) => {
    const {adminClient, adminUser} = await pw.getAdminClient();
    await cleanUpBotDMs(adminClient, adminUser!.id, pluginId);
});

type GithubPluginSettings = {
    connecttoprivatebydefault: string | null;
    enablecodepreview: string;
    enableleftsidebar: boolean;
    enableprivaterepo: boolean | null;
    enablewebhookeventlogging: boolean;
    encryptionkey: string;
    enterprisebaseurl: string;
    enterpriseuploadurl: string;
    githuboauthclientid: string;
    githuboauthclientsecret: string;
    githuborg: string | null;
    usepreregisteredapplication: boolean;
    webhooksecret: string;
}

const githubConfig: GithubPluginSettings = {
    githuboauthclientid: '',
    githuboauthclientsecret: '',

    connecttoprivatebydefault: null,
    enablecodepreview: 'public',
    enableleftsidebar: true,
    enableprivaterepo: null,
    enablewebhookeventlogging: false,
    encryptionkey: '',
    enterprisebaseurl: '',
    enterpriseuploadurl: '',
    githuborg: null,
    usepreregisteredapplication: false,
    webhooksecret: '',
};

// # Set plugin settings
test.beforeAll(async ({pw}) => {
    if (process.env.AVOID_TEST_CLEANUP === 'true') {
        return
    }

    const {adminClient} = await pw.getAdminClient();

    const config = await adminClient.getConfig();
    const newConfig: DeepPartial<AdminConfig> = {
        PluginSettings: {
            ...config.PluginSettings,
            Plugins: {
                ...config.PluginSettings.Plugins,
                [pluginId]: githubConfig as any,
            },
        },
    };

    await adminClient.patchConfig(newConfig);
});

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
});
