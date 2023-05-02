// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// ***************************************************************
// - [#] indicates a test step (e.g. # Go to a page)
// - [*] indicates an assertion (e.g. * Check the title)
// ***************************************************************

import {expect, test} from '@e2e-support/test_fixture';

import {messages} from '../../support/constants';
import {getGithubBotDMPageURL, waitForNewMessages} from '../../support/utils';
import {
    getBotTagFromPost,
    getPostAuthor,
} from '../../support/components/post';

export default {
    connected: () => {
        test.describe('RHS panel', () => {
            test('from connected account', async ({pages, page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }
            });
        });
    },
    unconnected: () => {
        test.describe('RHS panel', () => {
            test('from non connected account', async ({pages, page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }

                const dmURL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
                await page.goto(dmURL, {waitUntil: 'load'});
            });
        });
    },
    noSetup: () => {
        test.describe('RHS panel', () => {
            test('before doing setup', async ({pages, page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }
            });
        });
    },
};
