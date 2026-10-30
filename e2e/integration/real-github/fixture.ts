// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {Client4} from '@mattermost/client';
import type {Channel} from '@mattermost/types/channels';
import type {Team} from '@mattermost/types/teams';
import type {UserProfile} from '@mattermost/types/users';
import {expect, test as base} from '@mattermost/playwright-lib';

import {seedUserInfo} from '../support/kv';

import {realGitHub} from './env';
import {readState, type RealState} from './fixtures';

export {expect};

export type RealGitHubFixture = {
    adminClient: Client4;
    user: UserProfile;
    userClient: Client4;
    team: Team;
    channel: Channel;
    state: RealState;
    connectMachine: () => void;
};

// pw.initSetup() is deliberately avoided: it resets server config, which drops the plugin's settings.
export const test = base.extend<{real: RealGitHubFixture}>({
    real: async ({pw}, use) => {
        const {adminClient} = await pw.getAdminClient();
        const team = await pw.createNewTeam(adminClient, {name: 'github', displayName: 'GitHub', type: 'O', unique: true});
        const user = await pw.createNewUserProfile(adminClient, {prefix: 'github', disableTutorial: true, disableOnboarding: true});
        await adminClient.addToTeam(team.id, user.id);
        const {client: userClient} = await pw.makeClient(user);
        if (!userClient) {
            throw new Error(`Failed to log in ${user.username}`);
        }
        const channel = await adminClient.getChannelByName(team.id, 'town-square');
        const state = readState();

        const connectMachine = () => seedUserInfo(user.id, state.login, realGitHub.machineToken, {allowedPrivateRepos: true});

        await use({adminClient, user, userClient, team, channel, state, connectMachine});
    },
});
