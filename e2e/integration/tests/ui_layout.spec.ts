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

test('the sidebar buttons fit the channel sidebar header without overlapping', async ({pw, github}) => {
    await seed(github);
    const {page} = await openChannel(pw, github.user, github.team.name);
    const buttons = sidebarButtons(page);
    await expect(buttons[0]).toHaveText('1');
    const sidebar = page.locator('#SidebarContainer');

    for (const button of buttons) {
        await expectInside(button, sidebar);
    }
    for (let i = 1; i < buttons.length; i++) {
        await expectNoOverlap(buttons[i - 1], buttons[i]);
    }
});

test('a long pull request title and many labels wrap inside the right-hand panel', async ({pw, github}) => {
    await seed(github);
    const {page} = await openChannel(pw, github.user, github.team.name);
    await sidebarButton(page, 'fa-compress').click();
    const rhs = page.locator('#rhsContainer');
    const title = rhs.getByRole('link', {name: longTitle});

    await expect(title).toBeVisible();
    await expectInside(title, rhs);
    for (const label of labels) {
        await expectInside(rhs.getByLabel(label), rhs);
    }
    await expectNothingClipped(rhs.locator('.scrollbar--view'));
});

test('the link tooltip stays inside the window and keeps its content', async ({pw, github}) => {
    const pr = await seed(github);
    const url = `https://github.com/${repo}/pull/${pr.number}`;
    await github.userClient.createPost({channel_id: github.channel.id, message: url} as any);
    const {page} = await openChannel(pw, github.user, github.team.name);

    await page.getByRole('link', {name: url}).last().hover();
    const tooltip = page.locator('.github-tooltip .box').first();
    await expect(tooltip).toContainText(longTitle.slice(0, 40));

    const viewport = page.viewportSize()!;
    const box = (await tooltip.boundingBox())!;
    expect(box.x).toBeGreaterThanOrEqual(0);
    expect(box.x + box.width).toBeLessThanOrEqual(viewport.width);
    await expectInside(tooltip.locator('.labels'), tooltip);
});

test('a webhook post with labels fits the post', async ({github, pw}) => {
    const pr = await seed(github);
    await fakeGitHub.createHook('acme', 'widgets', webhookURL, webhookSecret);
    await github.userClient.executeCommand(`/github subscriptions add ${repo}`, {channel_id: github.channel.id, team_id: github.team.id});
    await fakeGitHub.webhook({event: 'pull_request', action: 'opened', repo, number: pr.number, sender: github.login});
    const {page} = await openChannel(pw, github.user, github.team.name);

    const post = postWithText(page, '#new-pull-request');
    await expect(post).toBeVisible();
    await expectInside(post.getByRole('heading', {name: longTitle}), post);
    await expectNothingClipped(post.locator('.post-message__text'));
});

test.describe('accessibility', () => {
    test('the sidebar buttons and right-hand panel have no violations', async ({pw, github, axe}) => {
        await seed(github);
        const {page} = await openChannel(pw, github.user, github.team.name);
        await sidebarButton(page, 'fa-compress').click();
        await expect(page.locator('#rhsContainer').getByRole('link', {name: longTitle})).toBeVisible();
        await sidebarButtonBar(page).evaluate((el) => el.setAttribute('data-e2e-axe', 'github-sidebar-buttons'));

        const results = await axe.builder(page).include('[data-e2e-axe="github-sidebar-buttons"]').include('#rhsContainer .scrollbar--view').analyze();

        // Known plugin bug: the icon-only GitHub profile and refresh links have no accessible name.
        // Drop this entry once they are labelled; any other violation still fails.
        expect(results.violations.flatMap((v) => v.nodes.map((n) => `${v.id}: ${n.html.match(/fa-[\w-]+/)?.[0] ?? n.target}`))).toEqual(['link-name: fa-github', 'link-name: fa-refresh']);
    });

    test('the create issue modal has no violations', async ({pw, github, axe}) => {
        await seed(github);
        await github.userClient.createPost({channel_id: github.channel.id, message: 'Make this an issue'} as any);
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);
        await openCreateIssueModal(channelsPage, page);

        const results = await axe.builder(page).include('[role="dialog"]').analyze();

        // Known plugin bug: the five react-selects and the description textarea have no associated label.
        // Drop this entry once they are labelled; any other violation still fails.
        const knownUnlabelled = 'label: Form elements must have labels (6)';
        expect(results.violations.map((v) => `${v.id}: ${v.help} (${v.nodes.length})`)).toEqual([knownUnlabelled]);
    });
});