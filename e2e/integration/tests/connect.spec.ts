// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {Page} from '@playwright/test';

import {defaultScopes} from '../fake-github/server';
import {expect, test} from '../support/fixture';
import {fakeGitHub} from '../support/fake_github';
import {hasKey, readUserInfo} from '../support/kv';
import {botDMPosts, openChannel, postWithText, runSlashCommand, waitForPost} from '../support/ui';

const completePath = /\/plugins\/github\/oauth\/complete/;

// `/github connect` answers with a link that opens the OAuth flow in a new tab, which closes itself on success.
async function startConnect(page: Page, channelsPage: Parameters<typeof runSlashCommand>[0]) {
    await runSlashCommand(channelsPage, '/github connect');
    const complete = page.context().waitForEvent('response', {predicate: (r) => completePath.test(r.url()), timeout: 30_000});
    const [popup] = await Promise.all([
        page.waitForEvent('popup'),
        postWithText(page, 'Click here to link your GitHub account').getByRole('link', {name: 'Click here to link your GitHub account'}).click(),
    ]);
    return {popup, complete};
}

test.describe('OAuth connect', () => {
    test('connects through GitHub\'s consent page and stores the classic token', async ({pw, github}) => {
        await fakeGitHub.signIn(github.login);
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);

        const {popup, complete} = await startConnect(page, channelsPage);
        await expect(popup.getByRole('heading', {name: 'Authorize Mattermost E2E'})).toBeVisible();
        await expect(popup.getByRole('listitem')).toHaveText(defaultScopes.slice().sort());

        await popup.getByRole('button', {name: 'Authorize Mattermost E2E'}).click();
        expect((await complete).status()).toBe(200);

        const stored = readUserInfo(github.user.id);
        expect(stored).toMatchObject({GitHubUsername: github.login, AllowedPrivateRepos: false, MM34646ResetTokenDone: true, Token: {token_type: 'bearer'}});
        expect(stored!.Token.access_token).toMatch(/^gho_/);

        await waitForPost(() => botDMPosts(github.adminClient, github.user.id), (p) => (p.type as string) === 'custom_git_welcome', 'welcome DM');
        expect(await fakeGitHub.calls()).toEqual(expect.arrayContaining(['GET /login/oauth/authorize', 'POST /login/oauth/access_token', 'GET /api/user']));
    });

    test('skips the consent page when the app was already authorized for these scopes', async ({pw, github}) => {
        await fakeGitHub.signIn(github.login);
        await fakeGitHub.grant(github.login, defaultScopes);
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);

        const {complete} = await startConnect(page, channelsPage);

        expect((await complete).status()).toBe(200);
        expect(readUserInfo(github.user.id)?.GitHubUsername).toBe(github.login);
        expect(await fakeGitHub.calls()).not.toContain('POST /login/oauth/authorize');
    });

    // GitHub grants the union of every scope the user ever approved for the app, so an old `repo` grant comes back.
    test('rejects a token that carries private repo scope from an earlier authorization', async ({pw, github}) => {
        await fakeGitHub.signIn(github.login);
        await fakeGitHub.grant(github.login, ['repo']);
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);

        const {popup, complete} = await startConnect(page, channelsPage);
        await popup.getByRole('button', {name: 'Authorize Mattermost E2E'}).click();
        const response = await complete;

        expect(response.status()).toBe(403);
        expect(await response.text()).toContain('OAuth token scope does not match the requested permissions');
        expect(readUserInfo(github.user.id)).toBeUndefined();
        await waitForPost(() => botDMPosts(github.adminClient, github.user.id), (p) => JSON.stringify(p.props).includes('OAuth token scope does not match'), 'scope mismatch DM');
    });

    test('rejects a callback whose state was never issued', async ({pw, github}) => {
        const {page} = await openChannel(pw, github.user, github.team.name);

        const response = await page.goto('/plugins/github/oauth/complete?code=0123456789abcdef0123&state=forgedstate0000');

        expect(response?.status()).toBe(400);
        expect(await response?.text()).toContain('invalid state token');
        expect(await fakeGitHub.calls()).not.toContain('POST /login/oauth/access_token');
    });

    // GitHub reports token-exchange errors with HTTP 200 and an error field.
    test('fails cleanly when GitHub rejects an expired authorization code', async ({pw, github}) => {
        await fakeGitHub.signIn(github.login);
        await fakeGitHub.setOAuthApp({codeTTLms: 0});
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);

        const {popup, complete} = await startConnect(page, channelsPage);
        await popup.getByRole('button', {name: 'Authorize Mattermost E2E'}).click();
        const response = await complete;

        expect(response.status()).toBe(500);
        expect(await response.text()).toContain('failed to exchange oauth code into token');
        expect(readUserInfo(github.user.id)).toBeUndefined();
        const exchange = (await fakeGitHub.requests()).filter((r) => r.path === '/login/oauth/access_token');
        expect(exchange.length).toBeGreaterThan(0);
        expect(exchange.every((r) => r.status === 200)).toBe(true);
    });

    test('/github disconnect removes the stored token', async ({pw, github}) => {
        await github.connect();
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);

        await runSlashCommand(channelsPage, '/github disconnect');

        await expect(postWithText(page, 'Disconnected your GitHub account.')).toBeVisible();
        expect(hasKey(`${github.user.id}_githubtoken`)).toBe(false);
        expect(hasKey(`${github.login}_githubusername`)).toBe(false);
    });
});

test.describe('auth failures', () => {
    test('a revoked token disconnects the user and DMs them to reconnect', async ({pw, github}) => {
        const {token} = await github.connect();
        await fakeGitHub.revoke(token.value);
        const {channelsPage} = await openChannel(pw, github.user, github.team.name);

        await runSlashCommand(channelsPage, '/github me');

        await waitForPost(() => botDMPosts(github.adminClient, github.user.id), (p) => (p.type as string) === 'custom_git_revoked_token', 'revoked-token DM');
        expect(hasKey(`${github.user.id}_githubtoken`)).toBe(false);
        expect((await fakeGitHub.requests()).some((r) => r.path.startsWith('/api/') && r.status === 401)).toBe(true);
    });

    test('SAML enforcement keeps the connection and DMs a re-authorize link once', async ({pw, github}) => {
        await fakeGitHub.createOrg('acme-sso', {members: [github.login], ssoEnforced: true});
        await fakeGitHub.createRepo('acme-sso/widgets');
        await github.connect();
        const {channelsPage} = await openChannel(pw, github.user, github.team.name);

        await runSlashCommand(channelsPage, '/github subscriptions add acme-sso/widgets');
        await runSlashCommand(channelsPage, '/github subscriptions add acme-sso/widgets');

        const dm = await waitForPost(() => botDMPosts(github.adminClient, github.user.id), (p) => p.message.includes('SAML SSO session expired'), 'SAML DM');
        expect(dm.message).toContain('https://github.com/orgs/acme-sso/sso?authorization_request=');
        await expect.poll(async () => (await botDMPosts(github.adminClient, github.user.id)).filter((p) => p.message.includes('SAML SSO session expired')).length).toBe(1);
        expect(readUserInfo(github.user.id)?.GitHubUsername).toBe(github.login);
    });
});
