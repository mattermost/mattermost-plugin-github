// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// ***************************************************************
// - [#] indicates a test step (e.g. # Go to a page)
// - [*] indicates an assertion (e.g. * Check the title)
// ***************************************************************

import { expect, test } from "@e2e-support/test_fixture";
import { messages } from "../../support/constants";
import {
    getBotTagFromPost,
    getPostAuthor,
} from "../../support/components/post";
import "../../support/init_test";

const mmGithubHandle = 'MM-Github-Plugin';

export default {
    connected: () => {
        test.describe("/github me", () => {
            test("from connected account", async ({ pages, pw }) => {

                // # Log in
                const {adminUser} = await pw.getAdminClient();
                const {page} = await pw.testBrowser.login(adminUser);

                const c = new pages.ChannelsPage(page);
                await c.goto();

                // # Run comand
                await c.postMessage("/github me");
                await c.sendMessage();

                // # Get last post
                const post = await c.getLastPost();
                const postId = await post.getId();

                // * Verify that message is sent by the github bot
                await expect(getPostAuthor(post)).toHaveText("github");
                await expect(getBotTagFromPost(post)).toBeVisible();

                // * assert intro message
                await expect(
                    post.container.getByText("You are connected to Github as")
                ).toBeVisible();
                // * check username
                await expect(
                    post.container.getByRole("link", { name: mmGithubHandle })
                ).toBeVisible();
                // * check profile image
                await expect(
                    post.container.getByRole("heading").locator("img")
                ).toBeVisible();

                // # Refresh
                await page.reload();

                // * Assert that ephemeral has disappeared
                await expect(page.locator(`#post_${postId}`)).toHaveCount(0);

                await page.close();
            });
        });
    },
    unconnected: () => {
        test.describe("/github me", () => {
            test("from non connected account", async ({ pages, pw }) => {
                // # Log in
                const {adminUser} = await pw.getAdminClient();
                const {page} = await pw.testBrowser.login(adminUser);

                const c = new pages.ChannelsPage(page);
                await c.goto();

                // # Run comand
                await c.postMessage("/github me");
                await c.sendMessage();
                await page.waitForTimeout(500);

                // # Get last post
                const post = await c.getLastPost();
                const postId = await post.getId();

                // * Verify that message is sent by the github bot
                await expect(getPostAuthor(post)).toHaveText("github");
                await expect(getBotTagFromPost(post)).toBeVisible();

                // * assert failure message
                await expect(
                    post.container.getByText(messages.UNCONNECTED)
                ).toBeVisible();

                // # Refresh
                await page.reload();

                // * Assert that ephemeral has disappeared
                await expect(page.locator(`#post_${postId}`)).toHaveCount(0);

                await page.close();
            });
        });
    },
    noSetup: () => {
        test.describe("/github me", () => {
            test("before doing setup", async ({ pages, pw }) => {
                // # Log in
                const {adminUser} = await pw.getAdminClient();
                const {page} = await pw.testBrowser.login(adminUser);

                const c = new pages.ChannelsPage(page);
                await c.goto();

                // # Run comand
                await c.postMessage("/github me");
                await c.sendMessage();

                // # Get last post
                const post = await c.getLastPost();
                const postId = await post.getId();

                // * Verify that message is sent by the github bot
                await expect(getPostAuthor(post)).toHaveText("github");
                await expect(getBotTagFromPost(post)).toBeVisible();

                // * assert failure message
                await expect(
                    post.container.getByText(messages.NOSETUP)
                ).toBeVisible();

                // # Refresh
                await page.reload();

                // * Assert that ephemeral has disappeared
                await expect(page.locator(`#post_${postId}`)).toHaveCount(0);

                await page.close();
            });
        });
    },
};
