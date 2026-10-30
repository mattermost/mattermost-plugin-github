// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test} from '../support/fixture';
import {fakeGitHub} from '../support/fake_github';
import {openChannel, postWithText, runSlashCommand} from '../support/ui';

test('/github me reports the connected GitHub account', async ({pw, github}) => {
    await github.connect();
    const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);

    await runSlashCommand(channelsPage, '/github me');

    await expect(postWithText(page, 'You are connected to GitHub as')).toContainText(github.login);
    expect(await fakeGitHub.calls()).toContain('GET /api/user');
});
