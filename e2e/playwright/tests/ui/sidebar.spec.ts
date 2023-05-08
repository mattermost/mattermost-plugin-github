// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// ***************************************************************
// - [#] indicates a test step (e.g. # Go to a page)
// - [*] indicates an assertion (e.g. * Check the title)
// ***************************************************************

import {expect, test} from '@e2e-support/test_fixture';

import {GithubRHSCategory} from '../../support/components/todo_message';
import Sidebar from '../../support/components/sidebar';
import {expectedData} from '../../support/constants';
import {getGithubBotDMPageURL} from '../../support/utils';

export default {
    connected: () => {
        test.describe('left sidebar', () => {
            test('from connected account (before refresh)', async ({page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }
                const dmURL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
                await page.goto(dmURL, {waitUntil: 'load'});

                const sidebar = new Sidebar(page);

                // * Unconnected version of icon shouldn't be visible
                await expect(sidebar.containerUnconnected).not.toBeVisible();

                // * Counters must be visible
                await expect(sidebar.container).toBeVisible();
                await expect(sidebar.getCounter(GithubRHSCategory.OPEN_PR)).toBeVisible();
                await expect(sidebar.getCounter(GithubRHSCategory.REVIEW_PR)).toBeVisible();
                await expect(sidebar.getCounter(GithubRHSCategory.ASSIGNMENTS)).toBeVisible();
                await expect(sidebar.getCounter(GithubRHSCategory.UNREAD)).toBeVisible();
                await expect(sidebar.refresh).toBeVisible();
                await expect(sidebar.refresh).toBeEnabled();

                // * Assert counters before refresh
                await expect(sidebar.getCounter(GithubRHSCategory.OPEN_PR)).toHaveText('0');
                await expect(sidebar.getCounter(GithubRHSCategory.REVIEW_PR)).toHaveText('0');
                await expect(sidebar.getCounter(GithubRHSCategory.ASSIGNMENTS)).toHaveText('0');
                await expect(sidebar.getCounter(GithubRHSCategory.UNREAD)).toHaveText('0');
            });

            test('from connected account (after refresh)', async ({page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }
                const dmURL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
                await page.goto(dmURL, {waitUntil: 'load'});

                const sidebar = new Sidebar(page);

                // * Unconnected version of icon shouldn't be visible
                await expect(sidebar.containerUnconnected).not.toBeVisible();

                // # Click refresh data (fetch, impacts rate limit)
                await sidebar.refresh.click();

                // # Waits for 1sec
                await page.waitForTimeout(1000);

                // * Counters must be visible
                await expect(sidebar.container).toBeVisible();
                await expect(sidebar.getCounter(GithubRHSCategory.OPEN_PR)).toBeVisible();
                await expect(sidebar.getCounter(GithubRHSCategory.REVIEW_PR)).toBeVisible();
                await expect(sidebar.getCounter(GithubRHSCategory.ASSIGNMENTS)).toBeVisible();
                await expect(sidebar.getCounter(GithubRHSCategory.UNREAD)).toBeVisible();
                await expect(sidebar.refresh).toBeVisible();
                await expect(sidebar.refresh).toBeEnabled();

                // * Assert counters before refresh
                await expect(sidebar.getCounter(GithubRHSCategory.OPEN_PR)).toHaveText(expectedData[GithubRHSCategory.OPEN_PR].count);
                await expect(sidebar.getCounter(GithubRHSCategory.REVIEW_PR)).toHaveText(expectedData[GithubRHSCategory.REVIEW_PR].count);
                await expect(sidebar.getCounter(GithubRHSCategory.ASSIGNMENTS)).toHaveText(expectedData[GithubRHSCategory.ASSIGNMENTS].count);
                await expect(sidebar.getCounter(GithubRHSCategory.UNREAD)).toHaveText(expectedData[GithubRHSCategory.UNREAD].count);
            });
        });
    },
};
