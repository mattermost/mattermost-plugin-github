// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// ***************************************************************
// - [#] indicates a test step (e.g. # Go to a page)
// - [*] indicates an assertion (e.g. * Check the title)
// ***************************************************************

import {expect, test} from '@e2e-support/test_fixture';

import {GithubRHSCategory} from '../../support/components/todo_message';
import RHS from '../../support/components/rhs';
import Sidebar from '../../support/components/sidebar';
import {expectedData, username} from '../../support/constants';
import {getGithubBotDMPageURL} from '../../support/utils';

export default {
    connected: () => {
        test.describe('RHS panel', () => {
            test('Your open PRs', async ({page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }
                const dmURL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
                await page.goto(dmURL, {waitUntil: 'load'});

                const sidebar = new Sidebar(page);
                const rhs = new RHS(page);

                await sidebar.refresh.click();
                await page.waitForTimeout(1000);

                // # Click on Open PRs icon at sidebar
                await sidebar.getCounter(GithubRHSCategory.OPEN_PR).click();

                // * RHS must be visible
                await expect(rhs.container).toBeVisible();

                // * Assert RHS title is GitHub
                await expect(rhs.header_title).toHaveText('GitHub');

                // * Assert RHS subtitle text is correct
                await expect(rhs.title).toHaveText('Your Open Pull Requests');

                // * Assert RHS subtitle link is correct
                await expect(rhs.title).toBeVisible();
                await expect(rhs.title).toBeEnabled();
                await expect(rhs.title).toHaveAttribute('href', `https://github.com/pulls?q=is%3Aopen+is%3Apr+author%3A${username}+archived%3Afalse`);
                await expect(rhs.title).toHaveAttribute('target', '_blank');
                await expect(rhs.title).toHaveAttribute('rel', 'noopener noreferrer');

                // * Assert RHS item count is correct
                await expect(rhs.items).toHaveCount(Number(expectedData[GithubRHSCategory.OPEN_PR]));
            });

            test('Your review PRs', async ({page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }
                const dmURL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
                await page.goto(dmURL, {waitUntil: 'load'});

                const sidebar = new Sidebar(page);
                const rhs = new RHS(page);

                await sidebar.refresh.click();
                await page.waitForTimeout(1000);

                // # Click on Open PRs icon at sidebar
                await sidebar.getCounter(GithubRHSCategory.REVIEW_PR).click();

                // * RHS must be visible
                await expect(rhs.container).toBeVisible();

                // * Assert RHS title is GitHub
                await expect(rhs.header_title).toHaveText('GitHub');

                // * Assert RHS subtitle text is correct
                await expect(rhs.title).toHaveText('Pull Requests Needing Review');

                // * Assert RHS subtitle link is correct
                await expect(rhs.title).toBeVisible();
                await expect(rhs.title).toBeEnabled();
                await expect(rhs.title).toHaveAttribute('href', `https://github.com/pulls?q=is%3Aopen+is%3Apr+review-requested%3A${username}+archived%3Afalse`);
                await expect(rhs.title).toHaveAttribute('target', '_blank');
                await expect(rhs.title).toHaveAttribute('rel', 'noopener noreferrer');

                // * Assert RHS item count is correct
                await expect(rhs.items).toHaveCount(Number(expectedData[GithubRHSCategory.REVIEW_PR]));
            });

            test('Your assignments', async ({page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }
                const dmURL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
                await page.goto(dmURL, {waitUntil: 'load'});

                const sidebar = new Sidebar(page);
                const rhs = new RHS(page);

                await sidebar.refresh.click();
                await page.waitForTimeout(1000);

                // # Click on Open PRs icon at sidebar
                await sidebar.getCounter(GithubRHSCategory.ASSIGNMENTS).click();

                // * RHS must be visible
                await expect(rhs.container).toBeVisible();

                // * Assert RHS title is GitHub
                await expect(rhs.header_title).toHaveText('GitHub');

                // * Assert RHS subtitle text is correct
                await expect(rhs.title).toHaveText('Your Assignments');

                // * Assert RHS subtitle link is correct
                await expect(rhs.title).toBeVisible();
                await expect(rhs.title).toBeEnabled();
                await expect(rhs.title).toHaveAttribute('href', `https://github.com/pulls?q=is%3Aopen+archived%3Afalse+assignee%3A${username}`);
                await expect(rhs.title).toHaveAttribute('target', '_blank');
                await expect(rhs.title).toHaveAttribute('rel', 'noopener noreferrer');

                // * Assert RHS item count is correct
                await expect(rhs.items).toHaveCount(Number(expectedData[GithubRHSCategory.ASSIGNMENTS]));
            });

            test('Unread notifications', async ({page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }
                const dmURL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
                await page.goto(dmURL, {waitUntil: 'load'});

                const sidebar = new Sidebar(page);
                const rhs = new RHS(page);

                await sidebar.refresh.click();
                await page.waitForTimeout(1000);

                // # Click on Open PRs icon at sidebar
                await sidebar.getCounter(GithubRHSCategory.UNREAD).click();

                // * RHS must be visible
                await expect(rhs.container).toBeVisible();

                // * Assert RHS title is GitHub
                await expect(rhs.header_title).toHaveText('GitHub');

                // * Assert RHS subtitle text is correct
                await expect(rhs.title).toHaveText('Unread Messages');

                // * Assert RHS subtitle link is correct
                await expect(rhs.title).toBeVisible();
                await expect(rhs.title).toBeEnabled();
                await expect(rhs.title).toHaveAttribute('href', 'https://github.com/notifications');
                await expect(rhs.title).toHaveAttribute('target', '_blank');
                await expect(rhs.title).toHaveAttribute('rel', 'noopener noreferrer');

                // * Assert RHS item count is correct
                await expect(rhs.items).toHaveCount(Number(expectedData[GithubRHSCategory.UNREAD]));
            });
        });
    },
};
