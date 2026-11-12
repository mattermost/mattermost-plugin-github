// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test} from '../support/fixture';
import {fakeGitHub} from '../support/fake_github';
import {openChannel, postWithText, runSlashCommand} from '../support/ui';

/**
 * @objective Verify /github me replies with the connected GitHub account fetched from the GitHub API.
 */
test('reports connected GitHub account when running /github me', {tag: '@github_plugin'}, async ({pw, github}) => {
    // # Connect GitHub and run /github me
    await github.connect();
    const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);

    await runSlashCommand(channelsPage, '/github me');

    // * Verify the reply names the GitHub login and the user was fetched from GitHub
    await expect(postWithText(page, 'You are connected to GitHub as')).toContainText(github.login);
    expect(await fakeGitHub.calls()).toContain('GET /api/user');
});
