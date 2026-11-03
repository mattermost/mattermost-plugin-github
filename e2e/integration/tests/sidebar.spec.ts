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

/**
 * @objective Verify the sidebar counts and RHS lists are populated from GitHub search and notifications.
 */
test('shows sidebar counts and RHS lists from GitHub search and notifications', {tag: '@github_plugin'}, async ({pw, github}) => {
    // # Seed GitHub work, connect and open the channel
    await seedWork(github);
    await github.connect();
    const {page} = await openChannel(pw, github.user, github.team.name);

    // * Verify each sidebar button shows its count
    await expect(sidebarButton(page, 'fa-compress')).toHaveText('2');
    await expect(sidebarButton(page, 'fa-code-fork')).toHaveText('1');
    await expect(sidebarButton(page, 'fa-list-ol')).toHaveText('1');
    await expect(sidebarButton(page, 'fa-envelope')).toHaveText('1');

    // # Open the open pull requests list
    const rhs = page.locator('#rhsContainer');
    await sidebarButton(page, 'fa-compress').click();

    // * Verify only open pull requests are listed with their details
    await expect(rhs.getByRole('link', {name: 'Your Open Pull Requests'})).toBeVisible();
    await expect(rhs.getByRole('link', {name: 'Add widget polishing'})).toBeVisible();
    await expect(rhs.getByRole('link', {name: 'Fix widget wobble'})).toBeVisible();
    await expect(rhs.getByText('Closed widget work')).toHaveCount(0);
    await expect(rhs.getByText('3 Files Changed')).toBeVisible();
    await expect(rhs.getByLabel('Success')).toHaveCount(1);

    // # Open the review requests list
    await sidebarButton(page, 'fa-code-fork').click();

    // * Verify the review request is listed
    await expect(rhs.getByRole('link', {name: 'Pull Requests Needing Review'})).toBeVisible();
    await expect(rhs.getByRole('link', {name: 'Please review the gizmo'})).toBeVisible();

    // # Open the assignments list
    await sidebarButton(page, 'fa-list-ol').click();

    // * Verify the assigned issue is listed with its milestone
    await expect(rhs.getByRole('link', {name: 'Your Assignments'})).toBeVisible();
    await expect(rhs.getByRole('link', {name: 'Widgets squeak on Mondays'})).toBeVisible();
    await expect(rhs.getByRole('note', {name: 'v1.0'})).toBeVisible();

    // # Open the unread messages list
    await sidebarButton(page, 'fa-envelope').click();

    // * Verify the review request notification is listed
    await expect(rhs.getByRole('link', {name: 'Unread Messages'})).toBeVisible();
    await expect(rhs.getByText('Please review the gizmo')).toBeVisible();

    // * Verify the data came from GitHub GraphQL, notifications and pull request APIs
    const calls = await fakeGitHub.calls();
    expect(calls).toEqual(expect.arrayContaining(['POST /api/graphql', 'GET /api/notifications', `GET /api/repos/${repo}/pulls/1/reviews`, `GET /api/repos/${repo}/pulls/1`]));
});

/**
 * @objective Verify locking the plugin to an organization limits the sidebar counts to that organization.
 */
test('limits sidebar counts to the locked organization', {tag: '@github_plugin'}, async ({pw, github}) => {
    // # Seed a pull request outside the acme org, connect and open the channel
    await seedWork(github);
    await fakeGitHub.createOrg('elsewhere', {members: [github.login]});
    await fakeGitHub.createRepo('elsewhere/gadgets', {writers: [github.login]});
    await fakeGitHub.createItem('elsewhere/gadgets', {kind: 'pull', title: 'Gadget PR outside the org', author: github.login});
    await github.connect();
    const {page} = await openChannel(pw, github.user, github.team.name);

    // * Verify pull requests from both orgs are counted
    await expect(sidebarButton(page, 'fa-compress')).toHaveText('3');

    // # Lock the plugin to the acme org and reload
    const {patchPluginSettings} = await import('../support/plugin');
    await patchPluginSettings(github.adminClient, {githuborg: 'acme'});
    try {
        await page.reload();

        // * Verify only acme pull requests are counted and the search is scoped to the org
        await expect(sidebarButton(page, 'fa-compress')).toHaveText('2');
        const graphql = (await fakeGitHub.requests()).filter((r) => r.path === '/api/graphql').at(-1);
        expect(JSON.stringify(graphql?.body)).toContain('org:acme');
    } finally {
        await patchPluginSettings(github.adminClient, {githuborg: ''});
    }
});
