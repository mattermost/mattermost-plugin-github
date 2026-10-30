// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {PlaywrightExtended} from '@mattermost/playwright-lib';
import type {Locator} from '@playwright/test';

import type {WebhookRequest} from '../fake-github/webhooks';
import {webhookSecret, webhookURL} from '../support/constants';
import {expect, test, type GitHubFixture} from '../support/fixture';
import {fakeGitHub} from '../support/fake_github';
import {longTitle, repo, seedWork} from '../support/seed';
import {channelPosts, openChannel, openCreateIssueModal, postWithText, replaceText, runSlashCommand, sidebarButton, sidebarButtonBar, waitForPost} from '../support/ui';

test.describe('@visual', () => {
    // eslint-disable-next-line no-process-env
    test.skip(!process.env.PW_VISUAL_BROWSER_WS, 'baselines only match the dockerized browser (npm run test:visual)');

    // Core UI overlapping the target (avatars, status, timestamps) changes between runs.
    async function settle(pw: PlaywrightExtended, target: Locator, {keepHover = false} = {}) {
        const page = target.page();
        await expect(page.locator('.LoadingScreen')).toBeHidden();
        await pw.hideDynamicChannelsContent(page);
        if (!keepHover) {
            await page.mouse.move(0, 0);
        }
        await page.evaluate(() => document.fonts.ready);
        await page.evaluate(() => Promise.all(document.getAnimations().filter((a) => a.effect?.getComputedTiming().endTime !== Infinity).map((a) => a.finished)));
        await target.evaluate((el: HTMLElement) => {
            const {x, y} = el.getBoundingClientRect();
            el.style.transform = `translate(${Math.round(x) - x}px, ${Math.round(y) - y}px)`;
        });
    }

    // Users and commit SHAs are random per run, and "Opened 2 years ago" grows with the calendar.
    async function freeze(github: GitHubFixture, target: Locator) {
        await replaceText(target, [
            [github.login, 'gh-octocat'],
            [`@${github.user.username}`, '@octocat'],
            [github.user.username, 'octocat'],
            [/\b[0-9a-f]{6,40}\b/g, 'a1b2c3'],
            [/\b(a|an|a few|\d+) (seconds?|minutes?|hours?|days?|months?|years?)\b/g, '2 years'],
        ]);
    }

    async function shoot(pw: PlaywrightExtended, github: GitHubFixture, target: Locator, name: string, options?: {keepHover?: boolean}) {
        await expect(target).toBeVisible();
        await freeze(github, target);
        await settle(pw, target, options);
        await expect(target).toHaveScreenshot(name);
    }

    async function rhsTab(pw: PlaywrightExtended, github: GitHubFixture, icon: Parameters<typeof sidebarButton>[1], title: string) {
        await seedWork(github);
        const {page} = await openChannel(pw, github.user, github.team.name);
        await sidebarButton(page, icon).click();
        const list = page.locator('#rhsContainer .scrollbar--view');
        await expect(list.getByRole('link', {name: title})).toBeVisible();
        return list;
    }

    async function webhookPost(pw: PlaywrightExtended, github: GitHubFixture, request: Omit<WebhookRequest, 'repo' | 'sender'>, marker: string) {
        await fakeGitHub.createHook('acme', 'widgets', webhookURL, webhookSecret);
        await github.userClient.executeCommand(`/github subscriptions add ${repo} --features pulls,issues,issue_comments,pushes`, {channel_id: github.channel.id, team_id: github.team.id});
        await fakeGitHub.webhook({repo, sender: github.login, ...request});
        const post = await waitForPost(() => channelPosts(github.adminClient, github.channel.id), (p) => p.message.includes(marker), `webhook post containing "${marker}"`);
        const opened = await openChannel(pw, github.user, github.team.name);
        return {...opened, post, message: opened.page.locator(`#post_${post.id} .post-message__text`)};
    }

    test('sidebar buttons', async ({pw, github}) => {
        await seedWork(github);
        const {page} = await openChannel(pw, github.user, github.team.name);
        await expect(sidebarButton(page, 'fa-compress')).toHaveText('1');
        await shoot(pw, github, sidebarButtonBar(page), 'sidebar-buttons.png');
    });

    test('open pull requests in the right-hand panel', async ({pw, github}) => {
        await shoot(pw, github, await rhsTab(pw, github, 'fa-compress', longTitle), 'rhs-open-prs.png');
    });

    test('review requests in the right-hand panel', async ({pw, github}) => {
        await shoot(pw, github, await rhsTab(pw, github, 'fa-code-fork', 'Please review the gizmo'), 'rhs-review-requests.png');
    });

    test('assignments with a milestone in the right-hand panel', async ({pw, github}) => {
        await shoot(pw, github, await rhsTab(pw, github, 'fa-list-ol', 'Widgets squeak on Mondays'), 'rhs-assignments.png');
    });

    test('unread messages in the right-hand panel', async ({pw, github}) => {
        await shoot(pw, github, await rhsTab(pw, github, 'fa-envelope', 'Unread Messages'), 'rhs-unreads.png');
    });

    test('pull request link tooltip', async ({pw, github}) => {
        const pr = await seedWork(github);
        const url = `https://github.com/${repo}/pull/${pr.number}`;
        await github.userClient.createPost({channel_id: github.channel.id, message: url} as any);
        const {page} = await openChannel(pw, github.user, github.team.name);
        await page.getByRole('link', {name: url}).last().hover();
        const tooltip = page.locator('.github-tooltip .box').first();
        await expect(tooltip).toContainText('Splits polishing into stages.');
        await shoot(pw, github, tooltip, 'tooltip-pull.png', {keepHover: true});
    });

    test('issue link tooltip', async ({pw, github}) => {
        await seedWork(github);
        const url = `https://github.com/${repo}/issues/3`;
        await github.userClient.createPost({channel_id: github.channel.id, message: url} as any);
        const {page} = await openChannel(pw, github.user, github.team.name);
        await page.getByRole('link', {name: url}).last().hover();
        const tooltip = page.locator('.github-tooltip .box').first();
        await expect(tooltip).toContainText('Widgets squeak on Mondays');
        await shoot(pw, github, tooltip, 'tooltip-issue.png', {keepHover: true});
    });

    test('new pull request webhook post', async ({pw, github}) => {
        const pr = await seedWork(github);
        const {message} = await webhookPost(pw, github, {event: 'pull_request', action: 'opened', number: pr.number}, '#new-pull-request');
        await shoot(pw, github, message, 'webhook-new-pull.png');
    });

    test('merged pull request webhook post', async ({pw, github}) => {
        const pr = await seedWork(github);
        await fakeGitHub.updateItem(repo, pr.number, {state: 'closed', merged: true});
        const {message} = await webhookPost(pw, github, {event: 'pull_request', action: 'closed', number: pr.number}, 'was merged');
        await shoot(pw, github, message, 'webhook-merged-pull.png');
    });

    test('new issue webhook post', async ({pw, github}) => {
        await seedWork(github);
        const {message} = await webhookPost(pw, github, {event: 'issues', action: 'opened', number: 3}, '#new-issue');
        await shoot(pw, github, message, 'webhook-new-issue.png');
    });

    test('issue comment webhook post', async ({pw, github}) => {
        await seedWork(github);
        const comment = 'Oiled them, still squeaky. Next step is replacing the **bearing**:\n\n- order part\n- swap it';
        const {message} = await webhookPost(pw, github, {event: 'issue_comment', action: 'created', number: 3, comment}, 'New comment by');
        await shoot(pw, github, message, 'webhook-issue-comment.png');
    });

    test('push webhook post', async ({pw, github}) => {
        await seedWork(github);
        const {message} = await webhookPost(pw, github, {event: 'push'}, 'pushed [1 new commit]');
        await shoot(pw, github, message, 'webhook-push.png');
    });

    test('webhook post in the reply thread', async ({pw, github}) => {
        const pr = await seedWork(github);
        const {channelsPage, post} = await webhookPost(pw, github, {event: 'pull_request', action: 'opened', number: pr.number}, '#new-pull-request');
        await (await channelsPage.centerView.getPostById(post.id)).reply();
        const rhsPost = await channelsPage.sidebarRight.getPostById(post.id);
        await shoot(pw, github, rhsPost.container.locator('.post-message__text'), 'reply-thread.png');
    });

    test('GitHub account in the profile popover', async ({pw, github}) => {
        await github.connect();
        await github.userClient.createPost({channel_id: github.channel.id, message: 'Who am I on GitHub?'} as any);
        const {channelsPage} = await openChannel(pw, github.user, github.team.name);
        const popover = await channelsPage.openProfilePopover(await channelsPage.getLastPost());
        const attribute = popover.container.locator('a[href^="https://github.com/"]').locator('xpath=..');
        await shoot(pw, github, attribute, 'profile-popover.png');
    });

    test('create issue modal', async ({pw, github}) => {
        await seedWork(github);
        await github.userClient.createPost({channel_id: github.channel.id, message: 'Make this an issue'} as any);
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);
        const modal = await openCreateIssueModal(channelsPage, page);
        await shoot(pw, github, modal.locator('.modal-content'), 'create-issue-modal.png');
    });

    test('attach to issue modal', async ({pw, github}) => {
        await seedWork(github);
        await github.userClient.createPost({channel_id: github.channel.id, message: 'Attach this to an issue'} as any);
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);
        const post = await channelsPage.getLastPost();
        await post.hover();
        await post.postMenu.actionsButton.click();
        await page.getByRole('menuitem', {name: 'Attach to GitHub Issue'}).click();
        const modal = page.getByRole('dialog').filter({hasText: 'Attach Message to GitHub Issue'});
        await expect(modal.getByText('Attach this to an issue')).toBeVisible();
        await shoot(pw, github, modal.locator('.modal-content'), 'attach-issue-modal.png');
    });

    test('/github todo reply', async ({pw, github}) => {
        await seedWork(github);
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);
        await runSlashCommand(channelsPage, '/github todo');
        await shoot(pw, github, postWithText(page, 'Unread Messages').locator('.post-message__text'), 'todo.png');
    });
});
