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
    /**
     * @objective Verify /github connect goes through GitHub's consent page, stores the classic token and sends the welcome DM.
     */
    test('connects GitHub account through the OAuth consent page and stores the classic token', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Sign in to GitHub and start /github connect
        await fakeGitHub.signIn(github.login);
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);
        const {popup, complete} = await startConnect(page, channelsPage);

        // * Verify the consent page lists the default scopes
        await expect(popup.getByRole('heading', {name: 'Authorize Mattermost E2E'})).toBeVisible();
        await expect(popup.getByRole('listitem')).toHaveText(defaultScopes.slice().sort());

        // # Authorize the app
        await popup.getByRole('button', {name: 'Authorize Mattermost E2E'}).click();

        // * Verify the callback succeeds and the classic token is stored
        expect((await complete).status()).toBe(200);
        const stored = readUserInfo(github.user.id);
        expect(stored).toMatchObject({GitHubUsername: github.login, AllowedPrivateRepos: false, MM34646ResetTokenDone: true, Token: {token_type: 'bearer'}});
        expect(stored!.Token.access_token).toMatch(/^gho_/);

        // * Verify the welcome DM is sent and the expected GitHub calls were made
        await waitForPost(() => botDMPosts(github.adminClient, github.user.id), (p) => (p.type as string) === 'custom_git_welcome', 'welcome DM');
        expect(await fakeGitHub.calls()).toEqual(expect.arrayContaining(['GET /login/oauth/authorize', 'POST /login/oauth/access_token', 'GET /api/user']));
    });

    /**
     * @objective Verify /github connect completes without the consent page when the app already holds the requested scopes.
     */
    test('skips the OAuth consent page when the app was already authorized for the requested scopes', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Pre-authorize the app and start /github connect
        await fakeGitHub.signIn(github.login);
        await fakeGitHub.grant(github.login, defaultScopes);
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);
        const {complete} = await startConnect(page, channelsPage);

        // * Verify the user is connected without submitting the consent page
        expect((await complete).status()).toBe(200);
        expect(readUserInfo(github.user.id)?.GitHubUsername).toBe(github.login);
        expect(await fakeGitHub.calls()).not.toContain('POST /login/oauth/authorize');
    });

    // GitHub grants the union of every scope the user ever approved for the app, so an old `repo` grant comes back.
    /**
     * @objective Verify the plugin refuses a token whose scopes include private repo access left over from an earlier grant.
     */
    test('rejects OAuth token that carries private repo scope from an earlier authorization', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Grant the app repo scope earlier, then connect and authorize
        await fakeGitHub.signIn(github.login);
        await fakeGitHub.grant(github.login, ['repo']);
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);
        const {popup, complete} = await startConnect(page, channelsPage);
        await popup.getByRole('button', {name: 'Authorize Mattermost E2E'}).click();
        const response = await complete;

        // * Verify the callback is rejected, nothing is stored and the user gets a scope mismatch DM
        expect(response.status()).toBe(403);
        expect(await response.text()).toContain('OAuth token scope does not match the requested permissions');
        expect(readUserInfo(github.user.id)).toBeUndefined();
        await waitForPost(() => botDMPosts(github.adminClient, github.user.id), (p) => JSON.stringify(p.props).includes('OAuth token scope does not match'), 'scope mismatch DM');
    });

    /**
     * @objective Verify the OAuth callback rejects a state token the plugin never issued and skips the code exchange.
     */
    test('rejects OAuth callback whose state token was never issued', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Hit the OAuth callback with a forged state
        const {page} = await openChannel(pw, github.user, github.team.name);
        const response = await page.goto('/plugins/github/oauth/complete?code=0123456789abcdef0123&state=forgedstate0000');

        // * Verify the callback fails and no token exchange happens
        expect(response?.status()).toBe(400);
        expect(await response?.text()).toContain('invalid state token');
        expect(await fakeGitHub.calls()).not.toContain('POST /login/oauth/access_token');
    });

    // GitHub reports token-exchange errors with HTTP 200 and an error field.
    /**
     * @objective Verify an expired authorization code fails the OAuth callback without storing a token.
     */
    test('fails OAuth connect cleanly when GitHub rejects an expired authorization code', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Make authorization codes expire immediately, then connect and authorize
        await fakeGitHub.signIn(github.login);
        await fakeGitHub.setOAuthApp({codeTTLms: 0});
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);
        const {popup, complete} = await startConnect(page, channelsPage);
        await popup.getByRole('button', {name: 'Authorize Mattermost E2E'}).click();
        const response = await complete;

        // * Verify the callback fails and nothing is stored
        expect(response.status()).toBe(500);
        expect(await response.text()).toContain('failed to exchange oauth code into token');
        expect(readUserInfo(github.user.id)).toBeUndefined();

        // * Verify GitHub answered the token exchange with HTTP 200
        const exchange = (await fakeGitHub.requests()).filter((r) => r.path === '/login/oauth/access_token');
        expect(exchange.length).toBeGreaterThan(0);
        expect(exchange.every((r) => r.status === 200)).toBe(true);
    });

    /**
     * @objective Verify /github disconnect removes the user's stored token and username mapping.
     */
    test('removes the stored token when disconnecting through /github disconnect', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Connect, then run /github disconnect
        await github.connect();
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);
        await runSlashCommand(channelsPage, '/github disconnect');

        // * Verify the confirmation post and that the KV entries are gone
        await expect(postWithText(page, 'Disconnected your GitHub account.')).toBeVisible();
        expect(hasKey(`${github.user.id}_githubtoken`)).toBe(false);
        expect(hasKey(`${github.login}_githubusername`)).toBe(false);
    });
});

test.describe('auth failures', () => {
    /**
     * @objective Verify a GitHub 401 on a revoked token disconnects the user and DMs them to reconnect.
     */
    test('disconnects the user and DMs a reconnect prompt when the GitHub token is revoked', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Connect, revoke the token and run /github me
        const {token} = await github.connect();
        await fakeGitHub.revoke(token.value);
        const {channelsPage} = await openChannel(pw, github.user, github.team.name);
        await runSlashCommand(channelsPage, '/github me');

        // * Verify the revoked-token DM, the removed token and the 401 from GitHub
        await waitForPost(() => botDMPosts(github.adminClient, github.user.id), (p) => (p.type as string) === 'custom_git_revoked_token', 'revoked-token DM');
        expect(hasKey(`${github.user.id}_githubtoken`)).toBe(false);
        expect((await fakeGitHub.requests()).some((r) => r.path.startsWith('/api/') && r.status === 401)).toBe(true);
    });

    /**
     * @objective Verify SAML SSO enforcement keeps the user connected and DMs a single re-authorize link.
     */
    test('keeps the connection and DMs one re-authorize link when SAML SSO is enforced', {tag: '@github_plugin'}, async ({pw, github}) => {
        // # Create an SSO-enforced org and subscribe to its repo twice
        await fakeGitHub.createOrg('acme-sso', {members: [github.login], ssoEnforced: true});
        await fakeGitHub.createRepo('acme-sso/widgets');
        await github.connect();
        const {channelsPage} = await openChannel(pw, github.user, github.team.name);
        await runSlashCommand(channelsPage, '/github subscriptions add acme-sso/widgets');
        await runSlashCommand(channelsPage, '/github subscriptions add acme-sso/widgets');

        // * Verify exactly one SAML DM with the re-authorize link and that the user stays connected
        const dm = await waitForPost(() => botDMPosts(github.adminClient, github.user.id), (p) => p.message.includes('SAML SSO session expired'), 'SAML DM');
        expect(dm.message).toContain('https://github.com/orgs/acme-sso/sso?authorization_request=');
        await expect.poll(async () => (await botDMPosts(github.adminClient, github.user.id)).filter((p) => p.message.includes('SAML SSO session expired')).length).toBe(1);
        expect(readUserInfo(github.user.id)?.GitHubUsername).toBe(github.login);
    });
});
