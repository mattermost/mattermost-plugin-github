// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// ***************************************************************
// - [#] indicates a test step (e.g. # Go to a page)
// - [*] indicates an assertion (e.g. * Check the title)
// ***************************************************************

import {expect, test} from '@e2e-support/test_fixture';

import {GithubRHSCategory} from '../../support/components/todo_message';
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

                // * Unconnected version of icon shouldn't be visible
                await expect(page.getByTestId('sidebar-github-unconnected')).not.toBeVisible();

                // * Counters must be visible
                await expect(page.getByTestId('sidebar-github')).toBeVisible();
                await expect(page.getByTestId('sidebar-github-openpr')).toBeVisible();
                await expect(page.getByTestId('sidebar-github-reviewpr')).toBeVisible();
                await expect(page.getByTestId('sidebar-github-assignments')).toBeVisible();
                await expect(page.getByTestId('sidebar-github-unreads')).toBeVisible();
                await expect(page.getByTestId('sidebar-github-refresh')).toBeVisible();
                await expect(page.getByTestId('sidebar-github-refresh')).toBeEnabled();

                // * Assert counters before refresh
                await expect(page.getByTestId('sidebar-github-openpr')).toHaveText('0');
                await expect(page.getByTestId('sidebar-github-reviewpr')).toHaveText('0');
                await expect(page.getByTestId('sidebar-github-assignments')).toHaveText('0');
                await expect(page.getByTestId('sidebar-github-unreads')).toHaveText('0');
            });

            test('from connected account (after refresh)', async ({page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }
                const dmURL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
                await page.goto(dmURL, {waitUntil: 'load'});

                // * Unconnected version of icon shouldn't be visible
                await expect(page.getByTestId('sidebar-github-unconnected')).not.toBeVisible();

                // # Click refresh data (fetch, impacts rate limit)
                await page.getByTestId('sidebar-github-refresh').click();

                // # Waits for 1sec
                await page.waitForTimeout(1000);

                // * Counters must be visible
                await expect(page.getByTestId('sidebar-github')).toBeVisible();
                await expect(page.getByTestId('sidebar-github-openpr')).toBeVisible();
                await expect(page.getByTestId('sidebar-github-reviewpr')).toBeVisible();
                await expect(page.getByTestId('sidebar-github-assignments')).toBeVisible();
                await expect(page.getByTestId('sidebar-github-unreads')).toBeVisible();
                await expect(page.getByTestId('sidebar-github-refresh')).toBeVisible();
                await expect(page.getByTestId('sidebar-github-refresh')).toBeEnabled();

                // * Assert counters before refresh
                await expect(page.getByTestId('sidebar-github-openpr')).toHaveText(expectedData[GithubRHSCategory.OPEN_PR]);
                await expect(page.getByTestId('sidebar-github-reviewpr')).toHaveText(expectedData[GithubRHSCategory.REVIEW_PR]);
                await expect(page.getByTestId('sidebar-github-assignments')).toHaveText(expectedData[GithubRHSCategory.ASSIGNMENTS]);
                await expect(page.getByTestId('sidebar-github-unreads')).toHaveText(expectedData[GithubRHSCategory.UNREAD]);
            });
        });
    },
    unconnected: () => {
        test.describe('left sidebar', () => {
            test('from non connected account', async ({page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }
                const dmURL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
                await page.goto(dmURL, {waitUntil: 'load'});

                // * Unconnected version of icon should be visible
                await expect(page.getByTestId('sidebar-github-unconnected')).toBeVisible();

                // * Unconnected version of icon should have the connect link
                await expect(page.getByTestId('sidebar-github-unconnected')).toHaveAttribute('href', '/plugins/github/oauth/connect');
            });
        });
    },
    noSetup: () => {
        test.describe('left sidebar', () => {
            test('before doing setup', async ({page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }
                const dmURL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
                await page.goto(dmURL, {waitUntil: 'load'});

                // * Unconnected version of icon should be visible
                await expect(page.getByTestId('sidebar-github-unconnected')).toBeVisible();

                // * Unconnected version of icon should have the connect link
                await expect(page.getByTestId('sidebar-github-unconnected')).toHaveAttribute('href', '/plugins/github/oauth/connect');
            });
        });
    },
};
