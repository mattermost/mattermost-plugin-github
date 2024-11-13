// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {Page} from '@playwright/test';

import {ChannelsPage} from '@e2e-support/ui/pages';
import {UserProfile} from '@mattermost/types/users';
import Client4 from '@mattermost/client/client4';

export const waitForNewMessages = async (page: Page) => {
    await page.waitForTimeout(1000);

    // This should be able to be waited based on locators instead of pure time-based
    // The following code work "almost" always. Commented for now to have green tests.
    // await page.locator('#postListContent').getByTestId('NotificationSeparator').getByText('New Messages').waitFor();
};

export const getGithubBotDMPageURL = async (client: Client4, teamName: string, userId: string) => {
    let team = teamName;
    if (team === '') {
        const teams = await client.getTeamsForUser(userId);
        team = teams[0].name;
    }
    return `${team}/messages/@forgejo?skip_forgejo_fetch=true`;
};

export const fillTextField = async (name: string, value: string, page: Page) => {
    await page.getByTestId(`${name}input`).fill(value);
};

export const submitDialog = async (page: Page) => {
    await page.click('#interactiveDialogSubmit');
};

export const postMessage = async (message: string, c: ChannelsPage, page: Page) => {
    await c.postMessage(message);
    await page.getByTestId('SendMessageButton').click();
};

export const clickPostAction = async (name: string, c: ChannelsPage) => {
    // We need to wait for the next post to come up, since this opening a new tab and OAuth redirect can take an undeterminate
    // https://mattermost.atlassian.net/browse/MM-51906
    const postElement = await c.getLastPost();
    await postElement.container.getByText(name).last().click();
};

export const cleanUpBotDMs = async (client: Client4, userId: UserProfile['id'], botUsername: string) => {
    const bot = await client.getUserByUsername(botUsername);

    const userIds = [userId, bot.id];
    const channel = await client.createDirectChannel(userIds);
    const posts = await client.getPosts(channel.id);

    const deletePostPromises = Object.keys(posts.posts).map(client.deletePost);
    await Promise.all(deletePostPromises);
};

export const getSlackAttachmentLocatorId = (postId: string) => {
    return `#post_${postId} .attachment__body`;
};

export const getPostMessageLocatorId = (postId: string) => {
    return `#post_${postId} .post-message`;
};
