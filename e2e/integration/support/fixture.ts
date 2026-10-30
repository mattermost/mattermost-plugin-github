// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {Client4} from '@mattermost/client';
import type {Channel} from '@mattermost/types/channels';
import type {Team} from '@mattermost/types/teams';
import type {UserProfile} from '@mattermost/types/users';
import {expect, test as base} from '@mattermost/playwright-lib';

import type {Token} from '../fake-github/state';

import {fakeGitHub} from './fake_github';
import {seedUserInfo} from './kv';

export {expect};

type GitHubConnection = {login: string; token: Token};

export type GitHubFixture = {
    adminClient: Client4;
    user: UserProfile;
    userClient: Client4;
    team: Team;
    channel: Channel;
    login: string;
    connect: (mattermostUser?: UserProfile, options?: {login?: string; scopes?: string[]; allowedPrivateRepos?: boolean}) => Promise<GitHubConnection>;
};

export type FailureScreenshotOptions = {failureScreenshots: boolean};

// pw.initSetup() is deliberately avoided: it resets server config, which drops the plugin's settings and PluginStates.
export const test = base.extend<{github: GitHubFixture; failureScreenshot: void}, FailureScreenshotOptions>({
    failureScreenshots: [false, {option: true, scope: 'worker'}],

    // Playwright's own failure screenshot runs after pw closes the context and only sees a blank page.
    failureScreenshot: [async ({pw, failureScreenshots}, use, testInfo) => {
        await use();
        if (!failureScreenshots || testInfo.status === testInfo.expectedStatus) {
            return;
        }
        for (const [i, page] of (pw.testBrowser.context?.pages() ?? []).entries()) {
            const body = await page.screenshot({fullPage: true}).catch(() => undefined);
            if (body) {
                await testInfo.attach(`failure-page-${i}`, {body, contentType: 'image/png'});
            }
        }
    }, {auto: true}],

    github: async ({pw}, use) => {
        await fakeGitHub.reset();

        const {adminClient} = await pw.getAdminClient();
        const team = await pw.createNewTeam(adminClient, {name: 'github', displayName: 'GitHub', type: 'O', unique: true});
        const user = await pw.createNewUserProfile(adminClient, {prefix: 'github', disableTutorial: true, disableOnboarding: true});
        await adminClient.addToTeam(team.id, user.id);
        const {client: userClient} = await pw.makeClient(user);
        if (!userClient) {
            throw new Error(`Failed to log in ${user.username}`);
        }
        const channel = await adminClient.getChannelByName(team.id, 'town-square');

        // GitHub logins are global; deriving them from the random Mattermost username keeps tests independent.
        const login = `gh-${user.username}`;
        await fakeGitHub.createUser(login);

        const connect: GitHubFixture['connect'] = async (mattermostUser = user, options = {}) => {
            const ghLogin = options.login ?? (mattermostUser.id === user.id ? login : `gh-${mattermostUser.username}`);
            const token = await fakeGitHub.issueToken(ghLogin, options.scopes);
            seedUserInfo(mattermostUser.id, ghLogin, token.value, {allowedPrivateRepos: options.allowedPrivateRepos});
            return {login: ghLogin, token};
        };

        await use({adminClient, user, userClient, team, channel, login, connect});

        await fakeGitHub.settle();
        expect(await fakeGitHub.contractViolations(), 'fake-github traffic must match GitHub\'s REST, GraphQL and webhook schemas').toEqual([]);
    },
});
