// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test} from '../support/fixture';
import {fakeGitHub} from '../support/fake_github';
import {openChannel, postWithText, runSlashCommand} from '../support/ui';

/**
 * @objective Verify /github todo lists unread notifications, review requests, open pull requests and assignments from GitHub search.
 */
test('lists unread notifications, review requests, open PRs and assignments when running /github todo', {tag: '@github_plugin'}, async ({pw, github}) => {
    // # Seed a review request, an own pull request, an assigned issue and a notification, then run /github todo
    const me = github.login;
    await fakeGitHub.createUser('octo-author');
    await fakeGitHub.createRepo(`${me}/widgets`, {writers: ['octo-author']});
    const review = await fakeGitHub.createItem(`${me}/widgets`, {kind: 'pull', title: 'Please review the gizmo', author: 'octo-author', reviewers: [me]});
    await fakeGitHub.createItem(`${me}/widgets`, {kind: 'pull', title: 'My own widget PR', author: me});
    await fakeGitHub.createItem(`${me}/widgets`, {kind: 'issue', title: 'Assigned widget bug', author: 'octo-author', assignees: [me]});
    await fakeGitHub.createThread(me, `${me}/widgets`, {number: review.number, reason: 'review_requested'});
    await github.connect();
    const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);

    await runSlashCommand(channelsPage, '/github todo');

    // * Verify the reply lists each section with its item
    const todo = postWithText(page, 'Unread Messages');
    await expect(todo).toContainText('Review Requests');
    await expect(todo).toContainText('Please review the gizmo');
    await expect(todo).toContainText('Your Open Pull Requests');
    await expect(todo).toContainText('My own widget PR');
    await expect(todo).toContainText('Your Assignments');
    await expect(todo).toContainText('Assigned widget bug');

    // * Verify GitHub was searched for review requests, authored items and assignments
    const searches = (await fakeGitHub.requests()).filter((r) => r.path === '/api/search/issues').map((r) => r.query.q);
    expect(searches).toEqual(expect.arrayContaining([
        expect.stringContaining(`review-requested:${me}`),
        expect.stringContaining(`author:${me}`),
        expect.stringContaining(`assignee:${me}`),
    ]));
});
