// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import path from 'node:path';
import fs from 'node:fs';

import {test} from '@e2e-support/test_fixture';
import {DeepPartial} from '@mattermost/types/utilities';
import {AdminConfig} from '@mattermost/types/config';
import {cleanUpBotDMs} from './utils';

const pluginDistPath = path.join(__dirname, '../../../../dist');

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

// # Clear bot DM channel
test.beforeEach(async ({pw}) => {
    const {adminClient, adminUser} = await pw.getAdminClient();
    await cleanUpBotDMs(adminClient, adminUser!.id, 'github');
});

const getPluginBundlePath = async (): Promise<string> => {
    const files = await fs.promises.readdir(pluginDistPath);
    const bundle = files.find((fname) => fname.endsWith('.tar.gz'));
    if (!bundle) {
        throw new Error('Failed to find plugin bundle in dist folder');
    }

    return path.join(pluginDistPath, bundle);
}

// # Upload plugin
test.beforeEach(async ({pw}) => {
    const bundlePath = await getPluginBundlePath();
    const {adminClient} = await pw.getAdminClient();

    await adminClient.uploadPluginX(bundlePath, true);
    await adminClient.enablePlugin('github');
});

type GithubPluginConfig = {
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

const githubConfig: GithubPluginConfig = {
    connecttoprivatebydefault: null,
    enablecodepreview: 'public',
    enableleftsidebar: true,
    enableprivaterepo: null,
    enablewebhookeventlogging: false,
    encryptionkey: 'Cq9E3bajxRmPLohnBirnx9ldpJM-dmAK',
    // enterprisebaseurl: 'http://localhost:8080',
    // enterpriseuploadurl: 'http://localhost:8080',
    enterprisebaseurl: 'https://8080-mattermost-mattermostgi-0zihcnzj79x.ws-us90.gitpod.io',
    enterpriseuploadurl: 'https://8080-mattermost-mattermostgi-0zihcnzj79x.ws-us90.gitpod.io',
    githuboauthclientid: 'fakefakefakefakefake',
    githuboauthclientsecret: 'fakefakefakefakefakefakefakefakefakefake',
    githuborg: null,
    usepreregisteredapplication: false,
    webhooksecret: 'qFA0xOlqhGx3uGibtyj15sCxr2HxrKEC',
};

// # Set plugin settings
test.beforeEach(async ({pw, page}) => {
    const {adminClient} = await pw.getAdminClient();

    const config = await adminClient.getConfig();
    const newConfig: DeepPartial<AdminConfig> = {
        PluginSettings: {
            ...config.PluginSettings,
            Plugins: {
                ...config.PluginSettings.Plugins,
                github: githubConfig as any,
            },
        },
    };
});
