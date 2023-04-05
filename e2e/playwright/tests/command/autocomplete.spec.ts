// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// ***************************************************************
// - [#] indicates a test step (e.g. # Go to a page)
// - [*] indicates an assertion (e.g. * Check the title)
// ***************************************************************
import {test, expect} from '@e2e-support/test_fixture';
import {SlashCommandSuggestions} from '../../support/components/slash_commands';
import "../../support/init_test";


// This test is meant to get the slash command help for the main command
// at three scenarios: no setup done, setup ready but not connected account and
// fully setup and connected account.
//
// Note that this test does not cover any autocomplete of each of the subcommands,
// that should be covered in each subcommand spec.

// TODO: this is just temporary until we can make the real ouath thing
const mmUsername = process.env.PW_MM_USERNAME;
const mmPassword = process.env.PW_MM_PASSWORD;


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

            test('with just the main command', async ({pages, pw}) => {
                // # Log in
                const {adminUser} = await pw.getAdminClient();
                const {page} = await pw.testBrowser.login(adminUser);

                const c = new pages.ChannelsPage(page);
                await c.goto();

                const slash = new SlashCommandSuggestions(page.locator('#suggestionList'));

                // # Run incomplete command to trigger help
                await c.postMessage('/github');

                // * Assert suggestions are visible
                await slash.toBeVisible();

                // * Assert help is visible
                await expect(slash.getItemTitleNth(0)).toHaveText('github [command]');
                //TODO: setup is available but not listed here
                await expect(slash.getItemDescNth(0)).toHaveText('Available commands: connect, disconnect, todo, subscriptions, issue, me, mute, settings, help, about');

                await page.close();
            });

            test('with an additional space', async ({pages, pw}) => {
                // # Log in
                const {adminUser} = await pw.getAdminClient();
                const {page} = await pw.testBrowser.login(adminUser);

                const c = new pages.ChannelsPage(page);
                await c.goto();

                const slash = new SlashCommandSuggestions(page.locator('#suggestionList'));

                // # Run incomplete command+space to trigger autocomplete
                await c.postMessage('/github ');

                // * Assert suggestions are visible
                await slash.toBeVisible();

                // * Assert autocomplete commands
                completeCommands.forEach(async (item) => {
                    await expect(slash.getItemTitleNth(item.position)).toHaveText(item.cmd);
                });

                await page.close();
            });
        });
    },
    unconnected: () => {
        test.describe('available commands when unnconnected', () => {
            test('with just the main command', async ({pages, pw}) => {
                // # Log in
                const {adminUser} = await pw.getAdminClient();
                const {page} = await pw.testBrowser.login(adminUser);

                const c = new pages.ChannelsPage(page);
                await c.goto();

                const slash = new SlashCommandSuggestions(page.locator('#suggestionList'));

                // # Run incomplete command to trigger help
                await c.postMessage('/github');

                // * Assert suggestions are visible
                await slash.toBeVisible();

                // * Assert help is visible
                await expect(slash.getItemTitleNth(0)).toHaveText('github [command]');
                //TODO: setup is available but not listed here
                await expect(slash.getItemDescNth(0)).toHaveText('Available commands: connect, disconnect, todo, subscriptions, issue, me, mute, settings, help, about');

                await page.close();
            });

            test('with an additional space', async ({pages, pw}) => {
                // # Log in
                const {adminUser} = await pw.getAdminClient();
                const {page} = await pw.testBrowser.login(adminUser);

                const c = new pages.ChannelsPage(page);
                await c.goto();

                const slash = new SlashCommandSuggestions(page.locator('#suggestionList'));

                // # Run incomplete command+space to trigger autocomplete
                await c.postMessage('/github ');

                // * Assert suggestions are visible
                await slash.toBeVisible();

                // * Assert autocomplete commands
                completeCommands.forEach(async (item) => {
                    await expect(slash.getItemTitleNth(item.position)).toHaveText(item.cmd);
                });

                await page.close();
            });
        });
    },
    noSetup: () => {
        test.describe('available commands whe no setup', () => {
            test('with just the main command', async ({pages, pw}) => {
                // # Log in
                const {adminUser} = await pw.getAdminClient();
                const {page} = await pw.testBrowser.login(adminUser);

                const c = new pages.ChannelsPage(page);
                await c.goto();
                const slash = new SlashCommandSuggestions(page.locator('#suggestionList'));

                // # Run incomplete command to trigger help
                await c.postMessage('/github');

                // * Assert suggestions are visible
                await slash.toBeVisible();

                // * Assert help is visible
                await expect(slash.getItemTitleNth(0)).toHaveText('github [command]');
                await expect(slash.getItemDescNth(0)).toHaveText('Available commands: setup, about');

                await page.close();
            });

            test('with an additional space', async ({pages, pw}) => {
                // # Log in
                const {adminUser} = await pw.getAdminClient();
                const {page} = await pw.testBrowser.login(adminUser);

                const c = new pages.ChannelsPage(page);
                await c.goto();
                const slash = new SlashCommandSuggestions(page.locator('#suggestionList'));

                // # Run incomplete command+space to trigger autocomplete
                await c.postMessage('/github ');

                // * Assert suggestions are visible
                await slash.toBeVisible();

                // * Assert autocomplete commands are visible
                await expect(slash.getItemTitleNth(1)).toHaveText('setup');
                await expect(slash.getItemTitleNth(2)).toHaveText('about');

                await page.close();
            });
        });
    }
};
