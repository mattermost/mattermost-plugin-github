// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
import dotenv from 'dotenv';
import testConfig from '@e2e-test.playwright-config';

// Configuration override for plugin tests
testConfig.testDir = __dirname + '/tests';
testConfig.testMatch = 'test.list.ts';
testConfig.use!.video = {
    mode:    'retain-on-failure',
    size: { width: 640, height: 480 }
}
testConfig.timeout = 90000;

export default testConfig;
