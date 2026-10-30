// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test} from '@playwright/test';

import {skeleton, validateGraphQLQuery, validateResponse, validateWebhook, webhookSkeleton} from '../fake-github/contract/validator';
import {type FakeGitHub, startFakeGitHub} from '../fake-github/server';
import {State, normalizeScopes, parseSearch} from '../fake-github/state';
import {buildPayload, signatures} from '../fake-github/webhooks';

function seededState() {
    const state = new State();
    state.addUser('alice');
    state.addOrg('acme', [], ['alice']);
    state.addItem(state.addRepo('acme/widgets'), {kind: 'issue', title: 'Broken widget', author: 'alice'});
    return state;
}

// These check the fake against GitHub's published schemas without a Mattermost server.
const port = 8193;
const base = `http://localhost:${port}`;
const app = {clientId: 'Iv1.contract', clientSecret: 'contract-secret', callbackURL: 'http://localhost:1/plugins/github/oauth/complete', name: 'Contract'};

test.describe('contract', () => {
    let fake: FakeGitHub;
    let token: string;

    const control = async (path: string, body: unknown) => (await fetch(`${base}/__control${path}`, {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(body)})).json();
    const api = (path: string, init: RequestInit = {}) => fetch(`${base}/api${path}`, {...init, headers: {Authorization: `Bearer ${token}`, 'Content-Type': 'application/json', ...init.headers}});

    test.beforeAll(async () => {
        fake = await startFakeGitHub({port, publicURL: base, app});
    });
    test.afterAll(() => fake.close());

    test.beforeEach(async () => {
        await control('/reset', {});
        await control('/users', {login: 'alice'});
        await control('/users', {login: 'bob'});
        await control('/orgs', {login: 'acme', members: ['alice', 'bob'], admins: ['alice']});
        await control('/teams', {org: 'acme', slug: 'core', members: ['bob']});
        await control('/repos', {fullName: 'acme/widgets', milestones: ['v1']});
        await control('/items', {repo: 'acme/widgets', kind: 'pull', title: 'Add widget', author: 'alice', reviewers: ['bob'], teamReviewers: ['core'], labels: ['bug']});
        await control('/items', {repo: 'acme/widgets', kind: 'issue', title: 'Broken widget', author: 'bob', assignees: ['alice'], milestone: 1});
        await control('/threads', {login: 'alice', repo: 'acme/widgets', number: 1, reason: 'review_requested'});
        token = (await control('/tokens', {login: 'alice'})).value;
    });

    test.afterEach(async () => {
        await fetch(`${base}/__control/settle`, {method: 'POST'});
        expect(await (await fetch(`${base}/__control/contract-violations`)).json()).toEqual([]);
    });

    test('every REST route the plugin uses answers with a schema-valid body', async () => {
        for (const path of ['/user', '/users/bob', '/user/orgs', '/user/repos', '/users/alice/repos', '/orgs/acme', '/orgs/acme/repos', '/orgs/acme/hooks',
            '/repos/acme/widgets', '/repos/acme/widgets/issues/2', '/repos/acme/widgets/labels', '/repos/acme/widgets/assignees', '/repos/acme/widgets/milestones',
            '/repos/acme/widgets/pulls/1', '/repos/acme/widgets/pulls/1/reviews', '/search/issues?q=is%3Apr+review-requested%3Abob', '/notifications']) {
            expect((await api(path)).status, path).toBe(200);
        }
        expect((await api('/orgs/acme/members/bob')).status).toBe(204);
        expect((await api('/repos/acme/widgets/issues', {method: 'POST', body: JSON.stringify({title: 'New', labels: ['bug']})})).status).toBe(201);
        expect((await api('/repos/acme/widgets/issues/1/comments', {method: 'POST', body: JSON.stringify({body: 'hi'})})).status).toBe(201);
    });

    test('answers like GitHub for bad credentials, missing scopes and SAML', async () => {
        const bad = await fetch(`${base}/api/user`, {headers: {Authorization: 'Bearer gho_nope'}});
        expect(bad.status).toBe(401);
        expect(await bad.json()).toMatchObject({message: 'Bad credentials'});
        await control('/revoke', {token});
        expect((await api('/graphql', {method: 'POST', body: JSON.stringify({query: '{ search(type: ISSUE, query: "x", first: 1) { issueCount } }'})})).status).toBe(401);
        token = (await control('/tokens', {login: 'alice'})).value;

        expect((await api('/repos/acme/widgets/hooks')).status).toBe(404);
        await control('/users', {login: 'dave'});
        const narrow = (await control('/tokens', {login: 'dave', scopes: ['public_repo']})).value;
        expect((await fetch(`${base}/api/user/orgs`, {headers: {Authorization: `Bearer ${narrow}`}})).status).toBe(403);
        expect((await api('/user')).headers.get('x-oauth-scopes')).toBe('admin:org_hook, notifications, public_repo, read:org');

        await control('/sso', {org: 'acme', enforced: true});
        const saml = await api('/repos/acme/widgets');
        expect(saml.status).toBe(403);
        expect(saml.headers.get('x-github-sso')).toMatch(/^required; url=https:\/\/github\.com\/orgs\/acme\/sso\?authorization_request=/);
        const search = await api('/search/issues?q=author%3Aalice');
        expect((await search.json()).total_count).toBe(0);
        expect(search.headers.get('x-github-sso')).toMatch(/^partial-results; organizations=\d+$/);

        await control('/users', {login: 'carol'});
        const outsider = (await control('/tokens', {login: 'carol'})).value;
        expect((await fetch(`${base}/api/repos/acme/widgets`, {headers: {Authorization: `Bearer ${outsider}`}})).status).toBe(200);
    });

    test('commenting through the API delivers issue_comment to the repository hook', async () => {
        await control('/hooks', {owner: 'acme', repo: 'widgets', url: `${base}/`, secret: 's'});
        expect((await api('/repos/acme/widgets/issues/2/comments', {method: 'POST', body: JSON.stringify({body: 'on it'})})).status).toBe(201);
        await fetch(`${base}/__control/settle`, {method: 'POST'});
        const delivered = (await (await fetch(`${base}/__control/requests`)).json()).filter((r: {method: string; path: string; body?: {comment?: {body: string}}}) => r.method === 'POST' && r.path === '/');
        expect(delivered.map((r: {body?: {comment?: {body: string}}}) => r.body?.comment?.body)).toEqual(['on it']);
    });

    test('paginates with Link headers', async () => {
        for (let i = 0; i < 4; i++) {
            await control('/items', {repo: 'acme/widgets', kind: 'issue', title: `Bulk ${i}`, author: 'alice'});
        }
        const first = await api('/search/issues?q=repo%3Aacme%2Fwidgets&per_page=2');
        expect(first.headers.get('link')).toContain('rel="next"');
        const last = await api('/search/issues?q=repo%3Aacme%2Fwidgets&per_page=2&page=3');
        expect(last.headers.get('link')).not.toContain('rel="next"');
        expect((await last.json()).items).toHaveLength(2);
    });

    test('the plugin\'s sidebar and digest queries validate and execute against GitHub\'s schema', async () => {
        const query = `query($q: String!, $c: String) { pullRequest: search(first: 100, after: $c, query: $q, type: ISSUE) { issueCount pageInfo { hasNextPage endCursor }
            nodes { ... on PullRequest { body number authorAssociation createdAt updatedAt repository { name nameWithOwner url owner { login } } state title author { login } url
            labels(first: 100) { nodes { name color } } milestone { title } additions deletions changedFiles
            reviewRequests(first: 100) { nodes { requestedReviewer { __typename ... on User { login } ... on Team { slug organization { login } } } } } }
            ... on Issue { number title state url milestone { title } } } } }`;
        expect(validateGraphQLQuery(query).errors).toEqual([]);

        const res = await api('/graphql', {method: 'POST', body: JSON.stringify({query, variables: {q: 'review-requested:bob is:pr is:OPEN archived:false org:acme'}})});
        const {data} = await res.json();
        expect(data.pullRequest.issueCount).toBe(1);
        expect(data.pullRequest.nodes[0].reviewRequests.nodes.map((n: any) => n.requestedReviewer)).toEqual([
            {__typename: 'User', login: 'bob'},
            {__typename: 'Team', slug: 'core', organization: {login: 'acme'}},
        ]);
    });

    test('GraphQL rejects unknown fields and missing pagination bounds like GitHub', async () => {
        expect(validateGraphQLQuery('{ search(first: 1, query: "x", type: ISSUE) { nodes { ... on Issue { nope } } } }').errors).toHaveLength(1);
        const res = await api('/graphql', {method: 'POST', body: JSON.stringify({query: '{ search(query: "x", type: ISSUE) { issueCount } }'})});
        expect(res.status).toBe(200);
        expect((await res.json()).errors[0].message).toContain('You must provide a `first` or `last` value');
    });

    test('OAuth token exchange errors come back as HTTP 200, as form or JSON by Accept header', async () => {
        const form = await fetch(`${base}/login/oauth/access_token`, {method: 'POST', body: new URLSearchParams({client_id: app.clientId, client_secret: app.clientSecret, code: 'nope'})});
        expect(form.status).toBe(200);
        expect(Object.fromEntries(new URLSearchParams(await form.text()))).toMatchObject({error: 'bad_verification_code'});

        const json = await fetch(`${base}/login/oauth/access_token`, {method: 'POST', headers: {Accept: 'application/json'}, body: new URLSearchParams({client_id: app.clientId, client_secret: 'wrong', code: 'nope'})});
        expect(await json.json()).toMatchObject({error: 'incorrect_client_credentials'});
    });

    test('webhook payloads match GitHub\'s event schemas and carry both signatures', async () => {
        const state = fake.state;
        for (const request of [
            {event: 'pull_request', action: 'opened', number: 1},
            {event: 'pull_request', action: 'review_requested', number: 1, requestedReviewer: 'bob'},
            {event: 'issues', action: 'opened', number: 2},
            {event: 'issue_comment', action: 'created', number: 2},
            {event: 'push'},
        ]) {
            const payload = buildPayload(state, {repo: 'acme/widgets', sender: 'alice', ...request});
            expect(validateWebhook(request.event, payload), `${request.event}.${request.action}`).toEqual([]);
        }
        expect(signatures('secret', '{}')).toEqual({
            'X-Hub-Signature': expect.stringMatching(/^sha1=[0-9a-f]{40}$/),
            'X-Hub-Signature-256': expect.stringMatching(/^sha256=[0-9a-f]{64}$/),
        });
    });

    test('creating a hook sends GitHub\'s ping delivery', async () => {
        const received: string[] = [];
        const {createServer} = await import('node:http');
        const sink = createServer((req, res) => {
            received.push(String(req.headers['x-github-event']));
            res.end();
        });
        await new Promise<void>((resolve) => sink.listen(8194, resolve));
        try {
            const res = await api('/orgs/acme/hooks', {method: 'POST', body: JSON.stringify({name: 'web', config: {url: 'http://localhost:8194/hook', content_type: 'json', secret: 's'}, events: ['*']})});
            expect(res.status).toBe(201);
            await expect.poll(() => received).toEqual(['ping']);
        } finally {
            sink.close();
        }
    });
});

test.describe('fake semantics', () => {
    test('search qualifiers are case-insensitive, repeated org: is OR, and @me resolves to the viewer', () => {
        expect(parseSearch('is:pr is:OPEN org:a org:b author:@me "two words"', 'alice')).toEqual([
            {qualifier: 'is', values: ['pr'], negated: false},
            {qualifier: 'is', values: ['OPEN'], negated: false},
            {qualifier: 'org', values: ['a', 'b'], negated: false},
            {qualifier: 'author', values: ['alice'], negated: false},
            {qualifier: 'text', values: ['two words'], negated: false},
        ]);
    });

    test('scopes covered by a broader grant are folded away', () => {
        expect(normalizeScopes(['public_repo', 'repo', 'read:org', 'notifications'])).toEqual(['notifications', 'read:org', 'repo']);
        expect(normalizeScopes(['read:org', 'admin:org'])).toEqual(['admin:org']);
    });

    test('skeletons satisfy their own schemas', () => {
        expect(validateResponse('GET', '/user', 200, skeleton('private-user'))).toEqual([]);
        expect(validateWebhook('ping', webhookSkeleton('ping'))).toEqual([]);
    });

    test('the fake rejects payloads GitHub would never send', () => {
        expect(validateResponse('GET', '/user', 200, {login: 'x'})).not.toEqual([]);
        const payload = buildPayload(seededState(), {event: 'issues', action: 'opened', repo: 'acme/widgets', number: 1, sender: 'alice'});
        expect(validateWebhook('issues', {...payload, action: 'teleported'})).not.toEqual([]);
    });
});
