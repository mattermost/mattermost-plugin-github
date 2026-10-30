// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test, type GitHubFixture} from '../support/fixture';
import {fakeGitHub} from '../support/fake_github';
import {channelPosts, chooseOption, openChannel, openCreateIssueModal, postWithText, selectField, waitForPost} from '../support/ui';

const repo = 'acme/widgets';

async function seedRepo(github: GitHubFixture, options: {canPush?: boolean} = {}) {
    await fakeGitHub.createUser('octo-author');
    await fakeGitHub.createOrg('acme', {members: [github.login, 'octo-author']});
    await fakeGitHub.createRepo(repo, {writers: options.canPush === false ? ['octo-author'] : [github.login, 'octo-author'], labels: ['bug', 'enhancement'], milestones: ['v1.0']});
}

test.describe('link tooltip', () => {
    test('shows pull request details for a github.com link', async ({pw, github}) => {
        await seedRepo(github);
        const pr = await fakeGitHub.createItem(repo, {kind: 'pull', title: 'Add widget polishing', author: 'octo-author', labels: ['enhancement'], head: 'polish', base: 'main'});
        await github.connect();
        const url = `https://github.com/${repo}/pull/${pr.number}`;
        await github.userClient.createPost({channel_id: github.channel.id, message: `Have a look: ${url}`} as any);
        const {page} = await openChannel(pw, github.user, github.team.name);

        await postWithText(page, 'Have a look').getByRole('link', {name: url}).hover();

        const tooltip = page.locator('.github-tooltip').first();
        await expect(tooltip).toContainText('Add widget polishing');
        await expect(tooltip).toContainText(`#${pr.number}`);
        await expect(tooltip.locator('.opened-by')).toContainText('octo-author');
        await expect(tooltip.locator('.commit-ref')).toHaveText(['main', 'polish']);
        await expect(tooltip.locator('.labels')).toContainText('enhancement');
        expect(await fakeGitHub.calls()).toContain(`GET /api/repos/${repo}/pulls/${pr.number}`);
    });

    test('shows issue details for a github.com link', async ({pw, github}) => {
        await seedRepo(github);
        const issue = await fakeGitHub.createItem(repo, {kind: 'issue', title: 'Widgets squeak on Mondays', author: 'octo-author', body: 'Every Monday, without fail.'});
        await github.connect();
        const url = `https://github.com/${repo}/issues/${issue.number}`;
        await github.userClient.createPost({channel_id: github.channel.id, message: url} as any);
        const {page} = await openChannel(pw, github.user, github.team.name);

        await page.getByRole('link', {name: url}).last().hover();

        const tooltip = page.locator('.github-tooltip').first();
        await expect(tooltip).toContainText('Widgets squeak on Mondays');
        await expect(tooltip).toContainText('Every Monday, without fail.');
    });
});

test.describe('create issue from a post', () => {
    async function openCreateIssue(pw: Parameters<typeof openChannel>[0], github: GitHubFixture, message: string) {
        await github.userClient.createPost({channel_id: github.channel.id, message} as any);
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);
        return {page, modal: await openCreateIssueModal(channelsPage, page)};
    }

    test('creates the issue with labels, assignee and milestone and links back to the post', async ({pw, github}) => {
        await seedRepo(github);
        await github.connect();
        const {page, modal} = await openCreateIssue(pw, github, 'The widget squeaks when it rains');

        await expect(selectField(modal, 'Organization')).toContainText(github.login);
        await chooseOption(page, selectField(modal, 'Repository'), 'widgets');
        await modal.getByLabel('Title for the GitHub Issue').fill('Widget squeaks in the rain');
        await chooseOption(page, selectField(modal, 'Labels'), 'bug');
        await chooseOption(page, selectField(modal, 'Assignees'), github.login);
        await chooseOption(page, selectField(modal, 'Milestone'), 'v1.0');
        await modal.getByRole('button', {name: 'Submit'}).click();
        await expect(modal).toBeHidden();

        const created = (await fakeGitHub.items()).find((i) => i.title === 'Widget squeaks in the rain');
        expect(created).toMatchObject({kind: 'issue', author: github.login, labels: ['bug'], assignees: [github.login], milestone: 1});
        expect(created!.body).toContain('The widget squeaks when it rains');
        const reply = await waitForPost(() => channelPosts(github.adminClient, github.channel.id), (p) => p.message.startsWith('Created GitHub issue'), 'issue created reply');
        expect(reply.message).toContain(`https://github.com/${repo}/issues/${created!.number}`);
    });

    // GitHub silently drops labels, assignees and milestones from users without push access; the modal hides them.
    test('hides issue attributes for a repository the user cannot push to', async ({pw, github}) => {
        await seedRepo(github, {canPush: false});
        await github.connect();
        const {page, modal} = await openCreateIssue(pw, github, 'Read-only repo message');

        await chooseOption(page, selectField(modal, 'Repository'), 'widgets');

        await expect(modal.getByText('Labels', {exact: true})).toHaveCount(0);
        await expect(modal.getByText('Assignees', {exact: true})).toHaveCount(0);
        await modal.getByLabel('Title for the GitHub Issue').fill('Filed without push access');
        await modal.getByRole('button', {name: 'Submit'}).click();
        await expect(modal).toBeHidden();
        expect((await fakeGitHub.items()).find((i) => i.title === 'Filed without push access')).toMatchObject({labels: [], assignees: []});
    });

    test('switching organization lists that organization\'s repositories', async ({pw, github}) => {
        await seedRepo(github);
        await fakeGitHub.createOrg('gizmo-co', {members: [github.login]});
        await fakeGitHub.createRepo('gizmo-co/gizmos', {writers: [github.login]});
        await github.connect();
        const {page, modal} = await openCreateIssue(pw, github, 'Which org?');

        await chooseOption(page, selectField(modal, 'Organization'), 'gizmo-co');

        await expect.poll(() => fakeGitHub.calls(), {timeout: 5_000}).toContain('GET /api/orgs/gizmo-co/repos');
        await chooseOption(page, selectField(modal, 'Repository'), 'gizmos');
    });
});
