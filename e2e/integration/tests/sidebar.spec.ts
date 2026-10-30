// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test, type GitHubFixture} from '../support/fixture';
import {fakeGitHub} from '../support/fake_github';
import {openChannel, sidebarButton} from '../support/ui';

const repo = 'acme/widgets';

async function seedWork(github: GitHubFixture) {
    const me = github.login;
    await fakeGitHub.createUser('octo-author');
    await fakeGitHub.createOrg('acme', {members: [me, 'octo-author']});
    await fakeGitHub.createRepo(repo, {writers: [me, 'octo-author'], milestones: ['v1.0']});
    await fakeGitHub.createItem(repo, {kind: 'pull', title: 'Add widget polishing', author: me, labels: ['enhancement'], additions: 42, deletions: 7, changedFiles: 3});
    await fakeGitHub.createItem(repo, {kind: 'pull', title: 'Fix widget wobble', author: me, ciState: 'failure'});
    const review = await fakeGitHub.createItem(repo, {kind: 'pull', title: 'Please review the gizmo', author: 'octo-author', reviewers: [me]});
    await fakeGitHub.createItem(repo, {kind: 'pull', title: 'Closed widget work', author: me, state: 'closed'});
    await fakeGitHub.createItem(repo, {kind: 'issue', title: 'Widgets squeak on Mondays', author: 'octo-author', assignees: [me], milestone: 1});
    await fakeGitHub.createThread(me, repo, {number: review.number, reason: 'review_requested'});
}

test('sidebar counts and RHS lists come from GitHub search and notifications', async ({pw, github}) => {
    await seedWork(github);
    await github.connect();
    const {page} = await openChannel(pw, github.user, github.team.name);

    await expect(sidebarButton(page, 'fa-compress')).toHaveText('2');
    await expect(sidebarButton(page, 'fa-code-fork')).toHaveText('1');
    await expect(sidebarButton(page, 'fa-list-ol')).toHaveText('1');
    await expect(sidebarButton(page, 'fa-envelope')).toHaveText('1');

    const rhs = page.locator('#rhsContainer');
    await sidebarButton(page, 'fa-compress').click();
    await expect(rhs.getByRole('link', {name: 'Your Open Pull Requests'})).toBeVisible();
    await expect(rhs.getByRole('link', {name: 'Add widget polishing'})).toBeVisible();
    await expect(rhs.getByRole('link', {name: 'Fix widget wobble'})).toBeVisible();
    await expect(rhs.getByText('Closed widget work')).toHaveCount(0);
    await expect(rhs.getByText('3 Files Changed')).toBeVisible();
    await expect(rhs.getByLabel('Success')).toHaveCount(1);

    await sidebarButton(page, 'fa-code-fork').click();
    await expect(rhs.getByRole('link', {name: 'Pull Requests Needing Review'})).toBeVisible();
    await expect(rhs.getByRole('link', {name: 'Please review the gizmo'})).toBeVisible();

    await sidebarButton(page, 'fa-list-ol').click();
    await expect(rhs.getByRole('link', {name: 'Your Assignments'})).toBeVisible();
    await expect(rhs.getByRole('link', {name: 'Widgets squeak on Mondays'})).toBeVisible();
    await expect(rhs.getByRole('note', {name: 'v1.0'})).toBeVisible();

    await sidebarButton(page, 'fa-envelope').click();
    await expect(rhs.getByRole('link', {name: 'Unread Messages'})).toBeVisible();
    await expect(rhs.getByText('Please review the gizmo')).toBeVisible();

    const calls = await fakeGitHub.calls();
    expect(calls).toEqual(expect.arrayContaining(['POST /api/graphql', 'GET /api/notifications', `GET /api/repos/${repo}/pulls/1/reviews`, `GET /api/repos/${repo}/pulls/1`]));
});

test('an org lock limits the sidebar to that organization', async ({pw, github}) => {
    await seedWork(github);
    await fakeGitHub.createOrg('elsewhere', {members: [github.login]});
    await fakeGitHub.createRepo('elsewhere/gadgets', {writers: [github.login]});
    await fakeGitHub.createItem('elsewhere/gadgets', {kind: 'pull', title: 'Gadget PR outside the org', author: github.login});
    await github.connect();
    const {page} = await openChannel(pw, github.user, github.team.name);
    await expect(sidebarButton(page, 'fa-compress')).toHaveText('3');

    const {patchPluginSettings} = await import('../support/plugin');
    await patchPluginSettings(github.adminClient, {githuborg: 'acme'});
    try {
        await page.reload();
        await expect(sidebarButton(page, 'fa-compress')).toHaveText('2');
        const graphql = (await fakeGitHub.requests()).filter((r) => r.path === '/api/graphql').at(-1);
        expect(JSON.stringify(graphql?.body)).toContain('org:acme');
    } finally {
        await patchPluginSettings(github.adminClient, {githuborg: ''});
    }
});
