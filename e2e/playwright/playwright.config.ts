import dotenv from 'dotenv';
dotenv.config({path: `${__dirname}/.env`});

import testConfig from '@e2e-test.playwright-config';

testConfig.testDir = __dirname + '/tests';

export default testConfig;
