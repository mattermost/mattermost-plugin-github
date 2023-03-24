import path from 'node:path';

import type {Page} from '@playwright/test';

import {ChannelsPage} from '@e2e-support/ui/pages';
import {UserProfile} from '@mattermost/types/users';
import Client4 from '@mattermost/client/client4';

const SCREENSHOTS_DIR = path.join(__dirname, '../screenshots');

export const sleep = (millis = 500) => new Promise(r => setTimeout(r, millis));

export const fillTextField = async (name: string, value: string, page: Page) => {
    await page.getByTestId(`${name}input`).fill(value);
}

export const submitDialog = async (page: Page) => {
    await page.click('#interactiveDialogSubmit');
}

export const postMessage = async (message: string, c: ChannelsPage, page: Page) => {
    await c.postMessage(message);
    await page.getByTestId('SendMessageButton').click();
};

export const clickPostAction = async (name: string, c: ChannelsPage) => {
    // TODO we need to wait for the next post to come up, since this opening a new tab and OAuth redirect can take an undeterminate
    // page.waitForSelector // Step 3: Create a Webhook in GitHub

    await sleep(500);
    const postElement = await c.getLastPost();
    await postElement.container.getByText(name).last().click();
};

export const getLastPostText = async (c: ChannelsPage, page: Page): Promise<string> => {
    await sleep();

    const post = await c.getLastPost();
    const postId = await post.getId();

    const locatorId = `#post_${postId} .post-message`;
    return page.locator(locatorId).innerText();
}

export const screenshot = async(name: string, page: Page) => {
    await page.screenshot({path: path.join(SCREENSHOTS_DIR, name + '.png')});
    console.log(`Created screenshot ${name}`);
}

export const cleanUpBotDMs = async (client: Client4, userId: UserProfile['id'], botUsername: string) => {
    const bot = await client.getUserByUsername(botUsername);

    const userIds = [userId, bot.id];
    const channel = await client.createDirectChannel(userIds);
    const posts = await client.getPosts(channel.id);

    const deletePostPromises = Object.keys(posts.posts).map(client.deletePost);
    await Promise.all(deletePostPromises);
}

export const getSlackAttachmentLocatorId = (postId: string) => {
    return `#post_${postId} .attachment__body`;
}

export const getPostMessageLocatorId = (postId: string) => {
    return `#post_${postId} .post-message`;
}
