// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {GitHubFixture} from './fixture';
import {fakeGitHub} from './fake_github';

export const repo = 'acme/widgets';
export const labels = ['bug', 'enhancement', 'needs-design-review-from-the-platform-team'];
export const longTitle = 'Refactor the widget polishing pipeline so that it handles extremely long pull request titles gracefully '.repeat(2).trim();

// A fixed date keeps the tooltip's date line identical between runs.
const createdAt = '2024-01-15T12:00:00Z';

export async function seedWork(github: GitHubFixture) {
    const dates = {createdAt, updatedAt: createdAt};
    await fakeGitHub.createUser('octo-author');
    await fakeGitHub.createOrg('acme', {members: [github.login, 'octo-author']});
    await fakeGitHub.createRepo(repo, {writers: [github.login, 'octo-author'], labels, milestones: ['v1.0']});
    const pr = await fakeGitHub.createItem(repo, {kind: 'pull', title: longTitle, author: github.login, labels, body: 'Splits polishing into stages.', ...dates});
    await fakeGitHub.createItem(repo, {kind: 'pull', title: 'Please review the gizmo', author: 'octo-author', reviewers: [github.login], ...dates});
    await fakeGitHub.createItem(repo, {kind: 'issue', title: 'Widgets squeak on Mondays', author: 'octo-author', assignees: [github.login], milestone: 1, ...dates});
    await fakeGitHub.createThread(github.login, repo, {number: pr.number, reason: 'mention'});
    await github.connect();
    return pr;
}
