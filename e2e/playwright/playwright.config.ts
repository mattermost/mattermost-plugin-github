// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import dotenv from 'dotenv';
import testConfig from '@e2e-test.playwright-config';
dotenv.config({path: `${__dirname}/.env`});

// Configuration override for plugin tests
testConfig.testDir = __dirname + '/tests';
testConfig.outputDir = __dirname + '/test-results';
testConfig.testMatch = 'test.list.ts';
testConfig.timeout = 30 * 1000;
if (!testConfig.use) {
    testConfig.use = {};
}
testConfig.use.video = {
    mode: 'on',
    size: {width: 1024, height: 768},
};
testConfig.projects = [
    {
        name: 'setup',
        testMatch: /integrations\.setup\.ts/,
    },
    {
        name: 'chrome',
        use: {
            browserName: 'chromium',
            permissions: ['notifications'],
            viewport: {width: 1280, height: 1024},
            storageState: __dirname + '/.auth-user.json',
        },
        dependencies: ['setup'],
    },

];

export default testConfig;
