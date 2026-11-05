// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {PlaywrightExtended} from '@mattermost/playwright-lib';
import type {Locator} from '@playwright/test';

import type {WebhookRequest} from '../fake-github/webhooks';
import {webhookSecret, webhookURL} from '../support/constants';
import {expect, test, type GitHubFixture} from '../support/fixture';
import {fakeGitHub} from '../support/fake_github';
import {longTitle, repo, seedWork} from '../support/seed';
import {botDMPosts, channelPosts, openChannel, openCreateIssueModal, postWithText, replaceText, runSlashCommand, selectField, sidebarButton, sidebarButtonBar, waitForPost} from '../support/ui';

test.describe('visual', {tag: '@visual'}, () => {
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

    async function webhookPost(pw: PlaywrightExtended, github: GitHubFixture, request: Omit<WebhookRequest, 'repo' | 'sender'>, marker: string, features = 'pulls,issues,issue_comments,pushes') {
        await fakeGitHub.createHook('acme', 'widgets', webhookURL, webhookSecret);
        await github.userClient.executeCommand(`/github subscriptions add ${repo} --features ${features}`, {channel_id: github.channel.id, team_id: github.team.id});
        await fakeGitHub.webhook({repo, sender: github.login, ...request});
        const post = await waitForPost(() => channelPosts(github.adminClient, github.channel.id), (p) => p.message.includes(marker), `webhook post containing "${marker}"`);
        const opened = await openChannel(pw, github.user, github.team.name);
        const message = (await opened.channelsPage.centerView.getPostById(post.id)).container.locator('.post-message__text');
        return {...opened, post, message};
    }

    /**
     * @objective Verify the GitHub sidebar buttons with counts render like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for sidebar buttons', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Seed GitHub work and open the channel
        await seedWork(github);
        const {page} = await openChannel(pw, github.user, github.team.name);

        // * Verify the open pull requests count and that the sidebar buttons match the baseline
        await expect(sidebarButton(page, 'fa-compress')).toHaveText('1');
        await shoot(pw, github, sidebarButtonBar(page), 'sidebar-buttons.png');
    });

    /**
     * @objective Verify the open pull requests list in the right-hand panel renders like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for open pull requests in RHS', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Open the open pull requests tab in the RHS
        const list = await rhsTab(pw, github, 'fa-compress', longTitle);

        // * Verify the list matches the baseline
        await shoot(pw, github, list, 'rhs-open-prs.png');
    });

    /**
     * @objective Verify the review requests list in the right-hand panel renders like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for review requests in RHS', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Open the review requests tab in the RHS
        const list = await rhsTab(pw, github, 'fa-code-fork', 'Please review the gizmo');

        // * Verify the list matches the baseline
        await shoot(pw, github, list, 'rhs-review-requests.png');
    });

    /**
     * @objective Verify the assignments list with a milestone in the right-hand panel renders like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for assignments with milestone in RHS', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Open the assignments tab in the RHS
        const list = await rhsTab(pw, github, 'fa-list-ol', 'Widgets squeak on Mondays');

        // * Verify the list matches the baseline
        await shoot(pw, github, list, 'rhs-assignments.png');
    });

    /**
     * @objective Verify the unread messages list in the right-hand panel renders like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for unread messages in RHS', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Open the unread messages tab in the RHS
        const list = await rhsTab(pw, github, 'fa-envelope', 'Unread Messages');

        // * Verify the list matches the baseline
        await shoot(pw, github, list, 'rhs-unreads.png');
    });

    /**
     * @objective Verify the empty open pull requests list in the right-hand panel renders like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for empty RHS', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Connect GitHub with no work and open the open pull requests tab in the RHS
        await github.connect();
        const {page} = await openChannel(pw, github.user, github.team.name);
        await sidebarButton(page, 'fa-compress').click();

        // * Verify the empty state matches the baseline
        const list = page.locator('#rhsContainer .scrollbar--view');
        await expect(list).toContainText('You have no active items');
        await shoot(pw, github, list, 'rhs-empty.png');
    });

    /**
     * @objective Verify the tooltip shown when hovering a pull request link renders like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for pull request link tooltip', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Post a pull request link and hover it
        const pr = await seedWork(github);
        const url = `https://github.com/${repo}/pull/${pr.number}`;
        await github.userClient.createPost({channel_id: github.channel.id, message: url} as any);
        const {page} = await openChannel(pw, github.user, github.team.name);
        await page.getByRole('link', {name: url}).last().hover();

        // * Verify the tooltip shows the pull request and matches the baseline
        const tooltip = page.locator('.github-tooltip .box').first();
        await expect(tooltip).toContainText('Splits polishing into stages.');
        await shoot(pw, github, tooltip, 'tooltip-pull.png', {keepHover: true});
    });

    /**
     * @objective Verify the tooltip shown when hovering an issue link renders like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for issue link tooltip', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Post an issue link and hover it
        await seedWork(github);
        const url = `https://github.com/${repo}/issues/3`;
        await github.userClient.createPost({channel_id: github.channel.id, message: url} as any);
        const {page} = await openChannel(pw, github.user, github.team.name);
        await page.getByRole('link', {name: url}).last().hover();

        // * Verify the tooltip shows the issue and matches the baseline
        const tooltip = page.locator('.github-tooltip .box').first();
        await expect(tooltip).toContainText('Widgets squeak on Mondays');
        await shoot(pw, github, tooltip, 'tooltip-issue.png', {keepHover: true});
    });

    /**
     * @objective Verify the post for a pull request opened webhook renders like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for new pull request webhook post', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Deliver a pull request opened webhook
        const pr = await seedWork(github);
        const {message} = await webhookPost(pw, github, {event: 'pull_request', action: 'opened', number: pr.number}, '#new-pull-request');

        // * Verify the post matches the baseline
        await shoot(pw, github, message, 'webhook-new-pull.png');
    });

    /**
     * @objective Verify the post for a merged pull request webhook renders like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for merged pull request webhook post', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Merge the pull request and deliver a pull request closed webhook
        const pr = await seedWork(github);
        await fakeGitHub.updateItem(repo, pr.number, {state: 'closed', merged: true});
        const {message} = await webhookPost(pw, github, {event: 'pull_request', action: 'closed', number: pr.number}, 'was merged');

        // * Verify the post matches the baseline
        await shoot(pw, github, message, 'webhook-merged-pull.png');
    });

    /**
     * @objective Verify the post for a pull request closed without merging renders like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for closed pull request webhook post', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Close the pull request without merging and deliver a pull request closed webhook
        const pr = await seedWork(github);
        await fakeGitHub.updateItem(repo, pr.number, {state: 'closed'});
        const {message} = await webhookPost(pw, github, {event: 'pull_request', action: 'closed', number: pr.number}, 'was closed by');

        // * Verify the post matches the baseline
        await shoot(pw, github, message, 'webhook-closed-pull.png');
    });

    /**
     * @objective Verify the post for an approving pull request review webhook renders like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for pull request review webhook post', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Deliver a pull request review submitted webhook approving the review request
        await seedWork(github);
        const comment = 'Looks good. One nit: rename `polish()` to **buff()**.';
        const {message} = await webhookPost(pw, github, {event: 'pull_request_review', action: 'submitted', number: 2, reviewState: 'APPROVED', comment}, ' approved ', 'pull_reviews');

        // * Verify the post matches the baseline
        await shoot(pw, github, message, 'webhook-pull-review.png');
    });

    /**
     * @objective Verify the post for an issue opened webhook renders like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for new issue webhook post', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Deliver an issue opened webhook
        await seedWork(github);
        const {message} = await webhookPost(pw, github, {event: 'issues', action: 'opened', number: 3}, '#new-issue');

        // * Verify the post matches the baseline
        await shoot(pw, github, message, 'webhook-new-issue.png');
    });

    /**
     * @objective Verify the post for an issue closed webhook renders like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for closed issue webhook post', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Close the issue and deliver an issue closed webhook
        await seedWork(github);
        await fakeGitHub.updateItem(repo, 3, {state: 'closed'});
        const {message} = await webhookPost(pw, github, {event: 'issues', action: 'closed', number: 3}, 'closed by');

        // * Verify the post matches the baseline
        await shoot(pw, github, message, 'webhook-closed-issue.png');
    });

    /**
     * @objective Verify the post for an issue labeled with the subscription's label renders like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for labeled issue webhook post', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Label the issue and deliver an issue labeled webhook to a label-filtered subscription
        await seedWork(github);
        await fakeGitHub.updateItem(repo, 3, {labels: ['bug']});
        const {message} = await webhookPost(pw, github, {event: 'issues', action: 'labeled', number: 3, label: 'bug'}, '#issue-labeled', 'issues,label:"bug"');

        // * Verify the post matches the baseline
        await shoot(pw, github, message, 'webhook-labeled-issue.png');
    });

    /**
     * @objective Verify the post for an issue comment webhook with markdown renders like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for issue comment webhook post', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Deliver an issue comment webhook with markdown
        await seedWork(github);
        const comment = 'Oiled them, still squeaky. Next step is replacing the **bearing**:\n\n- order part\n- swap it';
        const {message} = await webhookPost(pw, github, {event: 'issue_comment', action: 'created', number: 3, comment}, 'New comment by');

        // * Verify the post matches the baseline
        await shoot(pw, github, message, 'webhook-issue-comment.png');
    });

    /**
     * @objective Verify the post for a push webhook renders like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for push webhook post', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Deliver a push webhook
        await seedWork(github);
        const {message} = await webhookPost(pw, github, {event: 'push'}, 'pushed [1 new commit]');

        // * Verify the post matches the baseline
        await shoot(pw, github, message, 'webhook-push.png');
    });

    /**
     * @objective Verify the post for a branch created webhook renders like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for branch created webhook post', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Deliver a create webhook for a branch
        await seedWork(github);
        const {message} = await webhookPost(pw, github, {event: 'create', ref: 'polish-widgets'}, 'created by', 'creates');

        // * Verify the post matches the baseline
        await shoot(pw, github, message, 'webhook-create-branch.png');
    });

    /**
     * @objective Verify a webhook post in the reply thread renders like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for webhook post in reply thread', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Deliver a pull request opened webhook and open its reply thread
        const pr = await seedWork(github);
        const {channelsPage, post} = await webhookPost(pw, github, {event: 'pull_request', action: 'opened', number: pr.number}, '#new-pull-request');
        await (await channelsPage.centerView.getPostById(post.id)).reply();
        const rhsPost = await channelsPage.sidebarRight.getPostById(post.id);

        // * Verify the thread post matches the baseline
        await shoot(pw, github, rhsPost.container.locator('.post-message__text'), 'reply-thread.png');
    });

    /**
     * @objective Verify the review request DM from the bot renders like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for review request DM', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Deliver a review requested webhook naming the connected user and open the bot DM
        await seedWork(github);
        await fakeGitHub.createHook('acme', 'widgets', webhookURL, webhookSecret);
        await fakeGitHub.webhook({event: 'pull_request', action: 'review_requested', repo, number: 2, sender: 'octo-author', requestedReviewer: github.login});
        const dm = await waitForPost(() => botDMPosts(github.adminClient, github.user.id), (p) => p.message.includes('requested your review'), 'review request DM');
        const {channelsPage} = await openChannel(pw, github.user, github.team.name, '@github');

        // * Verify the DM matches the baseline
        const post = await channelsPage.centerView.getPostById(dm.id);
        await shoot(pw, github, post.container.locator('.post-message__text'), 'review-request-dm.png');
    });

    /**
     * @objective Verify the GitHub account attribute in the profile popover renders like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for GitHub account in profile popover', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Connect GitHub, post a message and open the author's profile popover
        await github.connect();
        await github.userClient.createPost({channel_id: github.channel.id, message: 'Who am I on GitHub?'} as any);
        const {channelsPage} = await openChannel(pw, github.user, github.team.name);
        const popover = await channelsPage.openProfilePopover(await channelsPage.getLastPost());
        const attribute = popover.container.locator('a[href^="https://github.com/"]').locator('xpath=..');

        // * Verify the GitHub attribute matches the baseline
        await shoot(pw, github, attribute, 'profile-popover.png');
    });

    /**
     * @objective Verify the create issue modal opened from a post renders like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for create issue modal', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Post a message and open the create issue modal from its menu
        await seedWork(github);
        await github.userClient.createPost({channel_id: github.channel.id, message: 'Make this an issue'} as any);
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);
        const modal = await openCreateIssueModal(channelsPage, page);

        // * Verify the modal matches the baseline
        await shoot(pw, github, modal.locator('.modal-content'), 'create-issue-modal.png');
    });

    /**
     * @objective Verify the create issue modal with a missing title error renders like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for create issue modal with validation error', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Open the create issue modal from a post and submit it empty
        await seedWork(github);
        await github.userClient.createPost({channel_id: github.channel.id, message: 'Submit me without a title'} as any);
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);
        const modal = await openCreateIssueModal(channelsPage, page);
        await modal.getByRole('button', {name: 'Submit'}).click();

        // * Verify the missing title error is shown and the modal matches the baseline
        await expect(selectField(modal, 'Repository')).toContainText('widgets');
        await expect(modal.getByText('This field is required.')).toBeVisible();
        await shoot(pw, github, modal.locator('.modal-content'), 'create-issue-modal-errors.png');
    });

    /**
     * @objective Verify the attach to issue modal opened from a post renders like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for attach to issue modal', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Post a message and open the attach to issue modal from its menu
        await seedWork(github);
        await github.userClient.createPost({channel_id: github.channel.id, message: 'Attach this to an issue'} as any);
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);
        const post = await channelsPage.getLastPost();
        await post.hover();
        await post.postMenu.actionsButton.click();
        await page.getByRole('menuitem', {name: 'Attach to GitHub Issue'}).click();

        // * Verify the modal shows the message and matches the baseline
        const modal = page.getByRole('dialog').filter({hasText: 'Attach Message to GitHub Issue'});
        await expect(modal.getByText('Attach this to an issue')).toBeVisible();
        await shoot(pw, github, modal.locator('.modal-content'), 'attach-issue-modal.png');
    });

    /**
     * @objective Verify the /github todo reply renders like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for /github todo reply', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Run /github todo
        await seedWork(github);
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);
        await runSlashCommand(channelsPage, '/github todo');

        // * Verify the reply matches the baseline
        await shoot(pw, github, postWithText(page, 'Unread Messages').locator('.post-message__text'), 'todo.png');
    });

    /**
     * @objective Verify the /github subscriptions list reply renders like the baseline.
     *
     * @precondition
     * Runs in the pinned dockerized browser (npm run test:visual).
     */
    test('matches baseline for /github subscriptions list reply', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Subscribe the channel to the repository and the organization, then run /github subscriptions list
        await seedWork(github);
        const args = {channel_id: github.channel.id, team_id: github.team.id};
        await github.userClient.executeCommand(`/github subscriptions add ${repo} --features pulls,issues,label:"bug" --render-style collapsed`, args);
        await github.userClient.executeCommand('/github subscriptions add acme --features creates,deletes,releases', args);
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);
        await runSlashCommand(channelsPage, '/github subscriptions list');

        // * Verify the reply matches the baseline
        const reply = postWithText(page, 'Subscriptions in this channel').locator('.post-message__text');
        await expect(reply).toContainText('acme/widgets');
        await shoot(pw, github, reply, 'subscriptions-list.png');
    });
});
