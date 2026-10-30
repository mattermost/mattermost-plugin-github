// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {validateGraphQLQuery, validateResponse, validateWebhook} from '../fake-github/contract/validator';
import {signatures} from '../fake-github/webhooks';
import {mattermostURL, webhookURL} from '../support/constants';
import {readUserInfo} from '../support/kv';
import {channelPosts, chooseOption, openChannel, openCreateIssueModal, runSlashCommand, selectField, sidebarButton, waitForPost} from '../support/ui';

import {gh, ghOK} from './api';
import {canSignInWithPassword, realGitHub} from './env';
import {expect, test, type RealGitHubFixture} from './fixture';
import {readState} from './fixtures';
import {completeGitHubSignIn} from './login/github_login';
import {reserveWrite} from './rate_limit';

const {fullName} = realGitHub;

type Delivery = {id: number; event: string; action: string | null; delivered_at: string};
type DeliveryDetail = Delivery & {request: {headers: Record<string, string>; payload: Record<string, any>}};

// GitHub can't reach this Mattermost, so the hook's recorded delivery is fetched and posted to the plugin.
async function replayDelivery(real: RealGitHubFixture, event: string, match: (payload: Record<string, any>) => boolean) {
    const path = `/repos/${fullName}/hooks/${real.state.hookId}/deliveries`;
    let detail: DeliveryDetail | undefined;
    await expect.poll(async () => {
        const recent = await ghOK<Delivery[]>('GET', `${path}?per_page=20`);
        for (const d of recent.filter((r) => r.event === event)) {
            const candidate = await ghOK<DeliveryDetail>('GET', `${path}/${d.id}`);
            if (match(candidate.request.payload)) {
                detail = candidate;
                return true;
            }
        }
        return false;
    }, {message: `no ${event} delivery recorded`, timeout: 2 * 60_000, intervals: [5_000]}).toBe(true);

    const {payload, headers} = detail!.request;

    // The API hands back the parsed payload, not the bytes GitHub signed, so it is re-signed with the hook's secret.
    const body = JSON.stringify(payload);
    const res = await fetch(webhookURL, {
        method: 'POST',
        body,
        headers: {'Content-Type': 'application/json', 'X-GitHub-Event': event, 'X-GitHub-Delivery': headers['X-GitHub-Delivery'] ?? '', ...signatures(real.state.webhookSecret, body)},
    });
    return {status: res.status, payload};
}

test.describe.serial('real GitHub with a KV-seeded machine token', () => {
    test('GitHub still behaves the way fake-github models it', async () => {
        const hooks = await gh('GET', `/repos/${fullName}/hooks`);
        const accepted = (hooks.headers.get('x-accepted-oauth-scopes') ?? '').split(',').map((s) => s.trim()).filter(Boolean);
        expect.soft(accepted, 'fake-github answers repo hooks with 404 unless the token has one of these scopes').toEqual(expect.arrayContaining(['admin:repo_hook', 'read:repo_hook', 'repo', 'write:repo_hook']));
        expect.soft(accepted).not.toContain('public_repo');

        const query = `{search(type: ISSUE, first: 1, query: "repo:${fullName} label:e2e-fixture") {nodes {... on Issue {state} ... on PullRequest {state}}}}`;
        expect(validateGraphQLQuery(query).errors, 'the validator lets this shape conflict through').toEqual([]);
        const graphql = await ghOK<{data?: unknown; errors?: unknown[]}>('POST', '/graphql', {body: {query}});
        expect.soft(graphql.errors, 'GitHub answers overlapping Issue/PullRequest state fields').toBeUndefined();

        const token = await fetch(`${realGitHub.webURL}/login/oauth/access_token`, {
            method: 'POST',
            headers: {Accept: 'application/json', 'Content-Type': 'application/json'},
            body: JSON.stringify({client_id: realGitHub.oauth.clientId, client_secret: realGitHub.oauth.clientSecret, code: 'not-a-real-code'}),
        });
        expect.soft(token.status, 'token errors come back as HTTP 200').toBe(200);
        expect.soft(await token.json()).toMatchObject({error: 'bad_verification_code'});

        const {issue, pull} = readState();
        for (const path of ['/user', `/repos/${fullName}`, `/repos/${fullName}/issues/${issue.number}`, `/repos/${fullName}/pulls/${pull.number}`, `/repos/${fullName}/labels`]) {
            expect.soft(validateResponse('GET', path, 200, await ghOK('GET', path)), `real ${path} matches the committed spec`).toEqual([]);
        }
    });

    test('the sidebar and tooltip show the front-loaded issue and pull request', async ({pw, real}) => {
        real.connectMachine();
        const {page, channelsPage} = await openChannel(pw, real.user, real.team.name);

        await runSlashCommand(channelsPage, '/github me');
        await expect(page.getByText(real.state.login).last()).toBeVisible();

        const rhs = page.locator('#rhsContainer');
        await sidebarButton(page, 'fa-compress').click();
        await expect(rhs.getByRole('link', {name: real.state.pull.title})).toBeVisible();
        await sidebarButton(page, 'fa-list-ol').click();
        await expect(rhs.getByRole('link', {name: real.state.issue.title})).toBeVisible();

        const url = `${realGitHub.webURL}/${fullName}/issues/${real.state.issue.number}`;
        await real.userClient.createPost({channel_id: real.channel.id, message: url} as any);
        await page.getByRole('link', {name: url}).last().hover();
        await expect(page.locator('.github-tooltip .box').first()).toContainText(real.state.issue.title);
    });

    test('an issue created from a post reaches GitHub, and its webhook posts back to the channel', async ({pw, real}) => {
        real.connectMachine();
        const title = `E2E journey ${Date.now()}`;
        await real.userClient.executeCommand(`/github subscriptions add ${fullName} --features issues`, {channel_id: real.channel.id, team_id: real.team.id});
        await real.userClient.createPost({channel_id: real.channel.id, message: 'File this from Mattermost'} as any);
        const {channelsPage, page} = await openChannel(pw, real.user, real.team.name);

        const modal = await openCreateIssueModal(channelsPage, page);
        await expect(selectField(modal, 'Organization')).toContainText(realGitHub.org);
        await chooseOption(page, selectField(modal, 'Repository'), realGitHub.repo);
        await modal.getByLabel('Title for the GitHub Issue').fill(title);
        await reserveWrite();
        await modal.getByRole('button', {name: 'Submit'}).click();
        await expect(modal).toBeHidden();

        const reply = await waitForPost(() => channelPosts(real.adminClient, real.channel.id), (p) => p.message.startsWith('Created GitHub issue'), 'issue created reply');
        const number = Number(/\/issues\/(\d+)/.exec(reply.message)?.[1]);
        try {
            expect(await ghOK('GET', `/repos/${fullName}/issues/${number}`)).toMatchObject({title, user: {login: real.state.login}});

            const {status, payload} = await replayDelivery(real, 'issues', (p) => p.action === 'opened' && p.issue?.number === number);
            expect.soft(validateWebhook('issues', payload), 'the real payload matches the schema fake-github validates against').toEqual([]);
            expect(status).toBe(200);
            const post = await waitForPost(() => channelPosts(real.adminClient, real.channel.id), (p) => p.message.includes('#new-issue') && p.message.includes(title), 'issues.opened post');
            expect(post.message).toContain(`/${fullName}/issues/${number}`);
        } finally {
            await ghOK('PATCH', `/repos/${fullName}/issues/${number}`, {body: {state: 'closed', state_reason: 'not_planned'}});
        }
    });
});

test.describe('real GitHub sign-in', () => {
    test.skip(!canSignInWithPassword(), 'needs E2E_GITHUB_LOGIN_USERNAME, _PASSWORD and _TOTP_SECRET');

    test('a user connects through github.com with password and TOTP', async ({pw, real}) => {
        const {page} = await openChannel(pw, real.user, real.team.name);
        const complete = `${mattermostURL}/plugins/github/oauth/complete`;

        await page.goto(`${mattermostURL}/plugins/github/oauth/connect`);
        await completeGitHubSignIn(page, () => page.url().startsWith(complete));
        await expect(page.getByText(/Completed connecting to GitHub/i)).toBeVisible();

        const info = readUserInfo(real.user.id);
        expect(info?.GitHubUsername.toLowerCase()).toBe(realGitHub.login.username!.toLowerCase());
        const token = info!.Token.access_token;
        try {
            const me = await gh('GET', '/user', {token});
            const scopes = (me.headers.get('x-oauth-scopes') ?? '').split(',').map((s) => s.trim()).filter(Boolean).sort();
            expect(scopes, 'the plugin asks for these scopes without /github connect private').toEqual(['admin:org_hook', 'notifications', 'public_repo', 'read:org']);

            const hooks = await gh('GET', `/repos/${fullName}/hooks`, {token});
            expect.soft(hooks.status, 'fake-github hides repo hooks from public_repo tokens, so /github subscriptions add always notes a missing webhook').toBe(404);
        } finally {
            // Revoking the grant brings back the consent screen and drops any widened scopes before the next run.
            const basic = `Basic ${Buffer.from(`${realGitHub.oauth.clientId}:${realGitHub.oauth.clientSecret}`).toString('base64')}`;
            await gh('DELETE', `/applications/${realGitHub.oauth.clientId}/grant`, {auth: basic, body: {access_token: token}});
        }
    });
});
