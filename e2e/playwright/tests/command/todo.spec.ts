// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// ***************************************************************
// - [#] indicates a test step (e.g. # Go to a page)
// - [*] indicates an assertion (e.g. * Check the title)
// ***************************************************************
import {test, expect} from '@e2e-support/test_fixture';
import {UserProfile} from '@mattermost/types/users';
import TodoMessage from '../../support/components/todo_message';
import {messages} from '../../support/constants';
import {getBotTagFromPost, getPostAuthor} from '../../support/components/post';


// This test is meant to get the slash command help for the main command
// at three scenarios: no setup done, setup ready but not connected account and
// fully setup and connected account.
//
// Note that this test does not cover any autocomplete of each of the subcommands,
// that should be covered in each subcommand spec.

// TODO: this is just temporary until we can make the real ouath thing
const mmUsername = process.env.PW_MM_USERNAME;
const mmPassword = process.env.PW_MM_PASSWORD;

const repoRegex = /https:\/\/github.com\/[\w\-]+\/[\w\-]+/;
const prRegex = /https:\/\/github.com\/[\w\-]+\/[\w\-]+\/pull\/\d+/;
const issueRegex = /https:\/\/github.com\/[\w\-]+\/[\w\-]+\/issues\/\d+/;

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

// # Log in as user
test.beforeEach(async ({pw, pages, page}) => {
    const {adminClient, adminUser} = await pw.getAdminClient();
    if (!adminUser) {
        throw new Error("Failed to get admin user");
    }
    await adminClient.patchConfig({
        ServiceSettings: {
            EnableTutorial: false,
            EnableOnboardingFlow: false,
        },
    });

    const adminConfig = await adminClient.getConfig();
    const loginPage = new pages.LoginPage(page, adminConfig);

    await loginPage.goto();
    await loginPage.toBeVisible();
    await loginPage.login({username: mmUsername, password: mmPassword} as UserProfile);
});

// TODO: all tests are not run at the same time since user is hardcoded from ENV
// As soon as we plug this with real setup connect, the test should include those steps and remove the skip
test.describe('/github todo command', () => {

    test.describe('from connected account', () => {
        test('complete', async ({pages, page}) => {
            const c = new pages.ChannelsPage(page);

            // # Run todo command
            await c.postMessage('/github todo');
            await c.sendMessage();

            // # Get last post
            const post = await c.getLastPost();
            const postId = await post.getId();

            const todo = new TodoMessage(post.container);

            // * Assert that titles are there for each section
            // Text are fixed and checked inside todo component handler
            await expect(todo.getTitle('openpr')).toBeVisible();
            await expect(todo.getTitle('assignments')).toBeVisible();
            await expect(todo.getTitle('reviewpr')).toBeVisible();
            await expect(todo.getTitle('unread')).toBeVisible();

            // * Assert that description are there for each section
            // TODO: Counters may vary and should be explicitely changed once the test accounts are set
            await expect(todo.getDesc('openpr')).toHaveText('You have 4 open pull requests:');
            await expect(todo.getDesc('assignments')).toHaveText('You have 4 assignments:');
            await expect(todo.getDesc('reviewpr')).toHaveText('You have 3 pull requests awaiting your review:');
            await expect(todo.getDesc('unread')).toHaveText('You don\'t have any unread messages.');

            // * Assert the open pull request list has 4 items
            const openPr = await todo.getList('openpr');
            await expect(openPr.locator('li')).toHaveCount(4)

            // * Assert the open pull request links are correct <REPO> <PR>
            for (let i=0; i<4; i++) {
                await expect(openPr.locator('li').nth(i).locator('a').nth(0)).toHaveAttribute('href', repoRegex)
                await expect(openPr.locator('li').nth(i).locator('a').nth(1)).toHaveAttribute('href', prRegex)
            }

            // * Assert the review request list has 3 items
            const reviewPr = await todo.getList('reviewpr');
            await expect(reviewPr.locator('li')).toHaveCount(3)

            // * Assert the open pull request links are correct <REPO> <PR>
            for (let i=0; i<3; i++) {
                await expect(reviewPr.locator('li').nth(i).locator('a').nth(0)).toHaveAttribute('href', repoRegex)
                await expect(reviewPr.locator('li').nth(i).locator('a').nth(1)).toHaveAttribute('href', prRegex)
            }

            // * Assert the assignments list has 4 items
            const assignments = await todo.getList('assignments');
            await expect(assignments.locator('li')).toHaveCount(4)

            // * Assert the assignments links are correct <REPO> <ISSUE>
            for (let i=0; i<4; i++) {
                await expect(assignments.locator('li').nth(i).locator('a').nth(0)).toHaveAttribute('href', repoRegex)
                await expect(assignments.locator('li').nth(i).locator('a').nth(1)).toHaveAttribute('href', issueRegex)
            }

            // * Assert the unread has 0 items
            const unread = await todo.getList('unread');
            await expect(unread.locator('li')).toHaveCount(0);

            // # Refresh
            await page.reload();

            // * Assert that ephemeral has disappeared
            await expect(page.locator(`#post_${postId}`)).toHaveCount(0);
        });

    });

    test('from non connected account', async ({pages, page}) => {
        const c = new pages.ChannelsPage(page);

        // # Run todo command
        await c.postMessage('/github todo');
        await c.sendMessage();

        // # Get last post
        const post = await c.getLastPost();
        const postId = await post.getId();

        // * Verify that message is sent by the github bot
        await expect(getPostAuthor(post)).toHaveText('github');
        await expect(getBotTagFromPost(post)).toBeVisible();

        // * assert failure message
        await expect(post.container.getByText(messages.UNCONNECTED)).toBeVisible()

        // # Refresh
        await page.reload();

        // * Assert that ephemeral has disappeared
        await expect(page.locator(`#post_${postId}`)).toHaveCount(0);

    });

    test.only('before doing setup', async ({pages, page}) => {
        const c = new pages.ChannelsPage(page);

        // # Run todo command
        await c.postMessage('/github todo');
        await c.sendMessage();

        // # Get last post
        const post = await c.getLastPost();
        const postId = await post.getId();

        // * Verify that message is sent by the github bot
        await expect(getPostAuthor(post)).toHaveText('github');
        await expect(getBotTagFromPost(post)).toBeVisible();

        // * assert failure message
        await expect(post.container.getByText(messages.NOSETUP)).toBeVisible()

        // # Refresh
        await page.reload();

        // * Assert that ephemeral has disappeared
        await expect(page.locator(`#post_${postId}`)).toHaveCount(0);

    });
});
