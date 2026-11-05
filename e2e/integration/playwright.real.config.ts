// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

/* eslint-disable no-process-env */

import {defineConfig, devices} from '@playwright/test';

import {mattermostURL} from './support/constants';

const isCI = Boolean(process.env.CI);

// Journey against real github.com. Needs a normal `make dist` build, not `make e2e-dist`.
export default defineConfig({
    testDir: './real-github',
    testMatch: /journey\.spec\.ts$/,
    outputDir: './test-results/real-github',
    globalSetup: require.resolve('./real-github/global_setup'),
    fullyParallel: false,
    workers: 1,
    forbidOnly: isCI,

    // Retries would repeat GitHub writes.
    retries: 0,
    timeout: 5 * 60_000,
    expect: {timeout: 30_000},
    reporter: isCI ? [['github'], ['list']] : [['list'], ['html', {open: 'never', outputFolder: 'playwright-report/real-github'}]],
    use: {
        baseURL: mattermostURL,
        headless: isCI || process.env.PW_HEADLESS !== 'false',

        // Real tokens, GitHub cookies and the login password end up in traces and screenshots, so they stay off in CI.
        trace: isCI ? 'off' : 'retain-on-failure',
        screenshot: 'off',
        video: 'off',
    },
    projects: [
        {
            name: 'real-github',
            use: {...devices['Desktop Chrome'], viewport: {width: 1280, height: 1024}},
        },
    ],
});
