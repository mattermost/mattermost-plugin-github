// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// ***************************************************************
// - [#] indicates a test step (e.g. # Go to a page)
// - [*] indicates an assertion (e.g. * Check the title)
// ***************************************************************
import {test, expect} from '@e2e-support/test_fixture';

import {SlashCommandSuggestions} from '../../support/components/slash_commands';
import {getGithubBotDMPageURL} from '../../support/utils';

const completeCommands = [
    {position: 1, cmd: 'connect'},
    {position: 2, cmd: 'disconnect'},
    {position: 3, cmd: 'todo'},
    {position: 4, cmd: 'subscriptions [command]'},
    {position: 5, cmd: 'issue [command]'},
    {position: 6, cmd: 'me'},
    {position: 7, cmd: 'mute [command]'},
    {position: 8, cmd: 'settings [setting] [value]'},
    {position: 9, cmd: 'setup [command]'},
];

export default {
    connected: () => {
        test.describe('available commands', () => {
            test('with just the main command', async ({pages, page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }
                const dmURL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
                await page.goto(dmURL, {waitUntil: 'load'});

                const c = new pages.ChannelsPage(page);
                const slash = new SlashCommandSuggestions(page.locator('#suggestionList'));

                // # Run incomplete command to trigger help
                await c.postMessage('/forgejo');

                // * Assert suggestions are visible
                await expect(slash.container).toBeVisible();

                // * Assert help is visible
                await expect(slash.getItemTitleNth(0)).toHaveText('github [command]');

                //TODO: setup is available but not listed here
                await expect(slash.getItemDescNth(0)).toHaveText('Available commands: connect, disconnect, todo, subscriptions, issue, me, mute, settings, help, about');
            });

            test('with an additional space', async ({pages, page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }

                const dmURL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
                await page.goto(dmURL, {waitUntil: 'load'});

                const c = new pages.ChannelsPage(page);
                const slash = new SlashCommandSuggestions(page.locator('#suggestionList'));

                // # Run incomplete command+space to trigger autocomplete
                await c.postMessage('/forgejo ');

                // * Assert suggestions are visible
                await expect(slash.container).toBeVisible();

                // * Assert autocomplete commands
                completeCommands.forEach(async (item) => {
                    await expect(slash.getItemTitleNth(item.position)).toHaveText(item.cmd);
                });
            });
        });
    },
    unconnected: () => {
        test.describe('available commands when unnconnected', () => {
            test('with just the main command', async ({pages, page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }

                const dmURL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
                await page.goto(dmURL, {waitUntil: 'load'});

                const c = new pages.ChannelsPage(page);

                const slash = new SlashCommandSuggestions(page.locator('#suggestionList'));

                // # Run incomplete command to trigger help
                await c.postMessage('/forgejo');

                // * Assert suggestions are visible
                await expect(slash.container).toBeVisible();

                // * Assert help is visible
                await expect(slash.getItemTitleNth(0)).toHaveText('github [command]');

                //TODO: setup is available but not listed here
                await expect(slash.getItemDescNth(0)).toHaveText('Available commands: connect, disconnect, todo, subscriptions, issue, me, mute, settings, help, about');
            });

            test('with an additional space', async ({pages, page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }

                const dmURL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
                await page.goto(dmURL, {waitUntil: 'load'});

                const c = new pages.ChannelsPage(page);

                const slash = new SlashCommandSuggestions(page.locator('#suggestionList'));

                // # Run incomplete command+space to trigger autocomplete
                await c.postMessage('/forgejo ');

                // * Assert suggestions are visible
                await expect(slash.container).toBeVisible();

                // * Assert autocomplete commands
                completeCommands.forEach(async (item) => {
                    await expect(slash.getItemTitleNth(item.position)).toHaveText(item.cmd);
                });
            });
        });
    },
    noSetup: () => {
        test.describe('available commands when no setup', () => {
            test('with just the main command', async ({page, pages, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }

                const dmURL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
                await page.goto(dmURL, {waitUntil: 'load'});

                const c = new pages.ChannelsPage(page);
                const slash = new SlashCommandSuggestions(page.locator('#suggestionList'));

                // # Run incomplete command to trigger help
                await c.postMessage('/forgejo');

                // * Assert suggestions are visible
                await expect(slash.container).toBeVisible();

                // * Assert help is visible
                await expect(slash.getItemTitleNth(0)).toHaveText('github [command]');
                await expect(slash.getItemDescNth(0)).toHaveText('Available commands: setup, about');
            });

            test('with an additional space', async ({pages, page, pw}) => {
                const {adminClient, adminUser} = await pw.getAdminClient();
                if (adminUser === null) {
                    throw new Error('can not get adminUser');
                }

                const dmURL = await getGithubBotDMPageURL(adminClient, '', adminUser.id);
                await page.goto(dmURL, {waitUntil: 'load'});

                const c = new pages.ChannelsPage(page);
                const slash = new SlashCommandSuggestions(page.locator('#suggestionList'));

                // # Run incomplete command+space to trigger autocomplete
                await c.postMessage('/forgejo ');

                // * Assert suggestions are visible
                await expect(slash.container).toBeVisible();

                // * Assert autocomplete commands are visible
                await expect(slash.getItemTitleNth(1)).toHaveText('setup');
                await expect(slash.getItemTitleNth(2)).toHaveText('about');
            });
        });
    },
};
