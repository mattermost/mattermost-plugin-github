// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {Client4} from '@mattermost/client';
import type {Post} from '@mattermost/types/posts';
import type {UserProfile} from '@mattermost/types/users';
import {expect, type ChannelsPage, type PlaywrightExtended} from '@mattermost/playwright-lib';
import type {Locator, Page} from '@playwright/test';

export async function openChannel(pw: PlaywrightExtended, user: UserProfile, teamName: string, channelName = 'town-square') {
    const {channelsPage, page} = await pw.testBrowser.login(user);
    await channelsPage.goto(teamName, channelName);
    await channelsPage.toBeVisible();
    return {channelsPage, page};
}

export async function runSlashCommand(channelsPage: ChannelsPage, command: string) {
    const postCreate = channelsPage.centerView.postCreate;
    await postCreate.input.fill(command);

    // Enter would pick the highlighted slash-command autocomplete suggestion instead of sending.
    await postCreate.sendMessageButton.click();
    await expect(postCreate.input).toHaveValue('');
}

export function sidebarButton(page: Page, icon: 'fa-compress' | 'fa-code-fork' | 'fa-list-ol' | 'fa-envelope') {
    return page.locator(`a:has(> i.${icon})`).first();
}

export function sidebarButtonBar(page: Page) {
    return sidebarButton(page, 'fa-compress').locator('xpath=ancestor::div[.//i[contains(@class, "fa-envelope")]][1]');
}

// Swaps per-run values (random usernames) for fixed text so screenshots are comparable.
export async function replaceText(target: Locator, replacements: Array<[string | RegExp, string]>) {
    await target.evaluate((root, pairs) => {
        const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
        for (let node = walker.nextNode(); node; node = walker.nextNode()) {
            for (const [source, flags, to] of pairs) {
                node.textContent = node.textContent!.replace(new RegExp(source, flags), to);
            }
        }
    }, replacements.map(([from, to]) => (typeof from === 'string' ? [from.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'g', to] : [from.source, from.flags, to])));
}

export async function openCreateIssueModal(channelsPage: ChannelsPage, page: Page) {
    const post = await channelsPage.getLastPost();
    await post.hover();
    await post.postMenu.actionsButton.click();
    await page.getByRole('menuitem', {name: 'Create GitHub Issue'}).click();
    const modal = page.getByRole('dialog').filter({hasText: 'Create GitHub Issue'});
    await expect(modal.getByLabel('Title for the GitHub Issue')).toBeVisible();
    await page.evaluate(() => Promise.all(document.getAnimations().filter((a) => a.effect?.getComputedTiming().endTime !== Infinity).map((a) => a.finished)));
    return modal;
}

// The react-select menus in the plugin don't expose role=option, so pick by filtering to one match.
export async function chooseOption(page: Page, field: Locator, option: string) {
    await field.getByRole('combobox').click();
    await page.keyboard.type(option);
    await expect(field.getByRole('log')).toContainText(/1 result available|option .* focused/);
    await page.keyboard.press('Enter');
    await expect(field).toContainText(option);
}

export function selectField(modal: Locator, label: string) {
    return modal.getByText(label, {exact: true}).locator('xpath=..');
}

export function postWithText(page: Page, text: string | RegExp) {
    return page.locator('.post').filter({hasText: text}).last();
}

export async function channelPosts(client: Client4, channelId: string): Promise<Post[]> {
    const {order, posts} = await client.getPosts(channelId);
    return order.map((id) => posts[id]);
}

export async function botDMPosts(client: Client4, userId: string): Promise<Post[]> {
    const bot = await client.getUserByUsername('github');
    const channel = await client.createDirectChannel([userId, bot.id]);
    return channelPosts(client, channel.id);
}

export async function waitForPost(load: () => Promise<Post[]>, match: (post: Post) => boolean, message: string): Promise<Post> {
    let found: Post | undefined;
    await expect.poll(async () => {
        found = (await load()).find(match);
        return Boolean(found);
    }, {message, timeout: 15_000}).toBe(true);
    return found!;
}
