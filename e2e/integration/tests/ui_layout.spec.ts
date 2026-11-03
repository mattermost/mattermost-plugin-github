// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {Page} from '@playwright/test';

import {webhookSecret, webhookURL} from '../support/constants';
import {expect, test} from '../support/fixture';
import {fakeGitHub} from '../support/fake_github';
import {expectInside, expectNoOverlap, expectNothingClipped} from '../support/layout';
import {labels, longTitle, repo, seedWork as seed} from '../support/seed';
import {openChannel, openCreateIssueModal, postWithText, sidebarButton, sidebarButtonBar} from '../support/ui';

function sidebarButtons(page: Page) {
    return (['fa-compress', 'fa-code-fork', 'fa-list-ol', 'fa-envelope'] as const).map((icon) => sidebarButton(page, icon));
}

/**
 * @objective Verify the plugin's sidebar buttons stay inside the channel sidebar header and do not overlap each other.
 */
test('fits the sidebar buttons inside the channel sidebar header without overlapping', {tag: '@github_plugin'}, async ({pw, github}) => {
    // # Seed work and open the channel
    await seed(github);
    const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);
    const buttons = sidebarButtons(page);

    // * Verify the buttons have loaded their counts
    await expect(buttons[0]).toHaveText('1');

    // * Verify each button sits inside the sidebar and none overlap
    const sidebar = channelsPage.sidebarLeft.container;
    for (const button of buttons) {
        await expectInside(button, sidebar);
    }
    for (let i = 1; i < buttons.length; i++) {
        await expectNoOverlap(buttons[i - 1], buttons[i]);
    }
});

/**
 * @objective Verify a long pull request title and many labels wrap inside the right-hand panel without clipping.
 */
test('wraps a long pull request title and many labels inside the right-hand panel', {tag: '@github_plugin'}, async ({pw, github}) => {
    // # Seed work and open the pull requests list in the right-hand panel
    await seed(github);
    const {page} = await openChannel(pw, github.user, github.team.name);
    await sidebarButton(page, 'fa-compress').click();
    const rhs = page.locator('#rhsContainer');
    const title = rhs.getByRole('link', {name: longTitle});

    // * Verify the title and every label stay inside the panel and nothing is clipped
    await expect(title).toBeVisible();
    await expectInside(title, rhs);
    for (const label of labels) {
        await expectInside(rhs.getByLabel(label), rhs);
    }
    await expectNothingClipped(rhs.locator('.scrollbar--view'));
});

/**
 * @objective Verify the pull request link tooltip stays inside the window and keeps its labels inside the tooltip.
 */
test('keeps the link tooltip inside the window with its content', {tag: '@github_plugin'}, async ({pw, github}) => {
    // # Post a pull request link and hover it
    const pr = await seed(github);
    const url = `https://github.com/${repo}/pull/${pr.number}`;
    await github.userClient.createPost({channel_id: github.channel.id, message: url} as any);
    const {page} = await openChannel(pw, github.user, github.team.name);
    await page.getByRole('link', {name: url}).last().hover();

    // * Verify the tooltip shows the title
    const tooltip = page.locator('.github-tooltip .box').first();
    await expect(tooltip).toContainText(longTitle.slice(0, 40));

    // * Verify the tooltip is horizontally inside the viewport and its labels fit
    const viewport = page.viewportSize()!;
    const box = (await tooltip.boundingBox())!;
    expect(box.x).toBeGreaterThanOrEqual(0);
    expect(box.x + box.width).toBeLessThanOrEqual(viewport.width);
    await expectInside(tooltip.locator('.labels'), tooltip);
});

/**
 * @objective Verify a pull request webhook post with a long title and labels fits inside the post.
 */
test('fits the pull request webhook post with labels inside the post', {tag: '@github_plugin'}, async ({github, pw}) => {
    // # Subscribe the channel and deliver a pull_request opened webhook
    const pr = await seed(github);
    await fakeGitHub.createHook('acme', 'widgets', webhookURL, webhookSecret);
    await github.userClient.executeCommand(`/github subscriptions add ${repo}`, {channel_id: github.channel.id, team_id: github.team.id});
    await fakeGitHub.webhook({event: 'pull_request', action: 'opened', repo, number: pr.number, sender: github.login});
    const {page} = await openChannel(pw, github.user, github.team.name);

    // * Verify the post's title stays inside it and nothing is clipped
    const post = postWithText(page, '#new-pull-request');
    await expect(post).toBeVisible();
    await expectInside(post.getByRole('heading', {name: longTitle}), post);
    await expectNothingClipped(post.locator('.post-message__text'));
});

test.describe('accessibility', () => {
    /**
     * @objective Verify the sidebar buttons and the right-hand panel have no accessibility violations beyond the known ones.
     */
    test('has no accessibility violations in the sidebar buttons and right-hand panel', {tag: '@github_plugin'}, async ({pw, github, axe}) => {
        // # Open the pull requests list in the right-hand panel
        await seed(github);
        const {page} = await openChannel(pw, github.user, github.team.name);
        await sidebarButton(page, 'fa-compress').click();
        await expect(page.locator('#rhsContainer').getByRole('link', {name: longTitle})).toBeVisible();

        // # Run axe on the sidebar buttons and the panel
        await sidebarButtonBar(page).evaluate((el) => el.setAttribute('data-e2e-axe', 'github-sidebar-buttons'));
        const results = await axe.builder(page).include('[data-e2e-axe="github-sidebar-buttons"]').include('#rhsContainer .scrollbar--view').analyze();

        // * Verify only the known violations are reported
        // Known plugin bug: the icon-only GitHub profile and refresh links have no accessible name.
        // Drop this entry once they are labelled; any other violation still fails.
        expect(results.violations.flatMap((v) => v.nodes.map((n) => `${v.id}: ${n.html.match(/fa-[\w-]+/)?.[0] ?? n.target}`))).toEqual(['link-name: fa-github', 'link-name: fa-refresh']);
    });

    /**
     * @objective Verify the create issue modal has no accessibility violations beyond the known ones.
     */
    test('has no accessibility violations in the create issue modal', {tag: '@github_plugin'}, async ({pw, github, axe}) => {
        // # Open the create issue modal from a post
        await seed(github);
        await github.userClient.createPost({channel_id: github.channel.id, message: 'Make this an issue'} as any);
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);
        await openCreateIssueModal(channelsPage, page);

        // # Run axe on the modal
        const results = await axe.builder(page).include('[role="dialog"]').analyze();

        // * Verify only the known violations are reported
        // Known plugin bug: the five react-selects and the description textarea have no associated label.
        // Drop this entry once they are labelled; any other violation still fails.
        const knownUnlabelled = 'label: Form elements must have labels (6)';
        expect(results.violations.map((v) => `${v.id}: ${v.help} (${v.nodes.length})`)).toEqual([knownUnlabelled]);
    });
});
