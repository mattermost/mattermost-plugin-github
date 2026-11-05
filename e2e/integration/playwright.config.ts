// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

/* eslint-disable no-process-env */

import {defineConfig, devices} from '@playwright/test';

import {fakeGitHubHost, mattermostURL} from './support/constants';
import type {FailureScreenshotOptions} from './support/fixture';

const isCI = Boolean(process.env.CI);
const visualBrowser = process.env.PW_VISUAL_BROWSER_WS;

export default defineConfig<FailureScreenshotOptions>({
    testDir: './tests',
    outputDir: './test-results',

    // No platform suffix: baselines always come from the dockerized browser.
    snapshotPathTemplate: '{testDir}/__screenshots__/{testFilePath}/{arg}{ext}',
    globalSetup: require.resolve('./global_setup'),
    fullyParallel: false,
    workers: 1,
    forbidOnly: isCI,
    retries: isCI ? 1 : 0,
    timeout: 60_000,
    expect: {timeout: 15_000, toHaveScreenshot: {stylePath: './support/visual.css'}},
    reporter: isCI ? [['github'], ['html', {open: 'never', outputFolder: 'playwright-report'}]] : [['list'], ['html', {open: 'never'}]],
    use: {
        baseURL: mattermostURL,
        headless: process.env.PW_HEADLESS !== 'false',
        // Only fake-github tokens and a throwaway server's sessions end up in these, so CI keeps them too.
        trace: 'retain-on-failure',
        screenshot: 'off',
        failureScreenshots: true,

        // The dockerized visual browser shares Mattermost's network and /etc/hosts, where fake-github already resolves.
        launchOptions: {
            slowMo: Number(process.env.PW_SLOWMO ?? 0),
            args: visualBrowser ? [] : [`--host-resolver-rules=MAP ${fakeGitHubHost} 127.0.0.1`],
        },
        connectOptions: visualBrowser ? {wsEndpoint: visualBrowser} : undefined,
    },
    projects: [
        {
            name: 'chrome',
            use: {...devices['Desktop Chrome'], viewport: {width: 1280, height: 1024}},
        },
    ],
});
