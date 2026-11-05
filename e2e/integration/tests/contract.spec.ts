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

    /**
     * @objective Verify every REST route the plugin calls answers with a status and body valid against GitHub's schema.
     */
    test('answers every plugin REST route with a schema-valid body', {tag: '@github_plugin'}, async () => {
        // # Call each read route the plugin uses
        // * Verify each answers 200
        for (const path of ['/user', '/users/bob', '/user/orgs', '/user/repos', '/users/alice/repos', '/orgs/acme', '/orgs/acme/repos', '/orgs/acme/hooks',
            '/repos/acme/widgets', '/repos/acme/widgets/issues/2', '/repos/acme/widgets/labels', '/repos/acme/widgets/assignees', '/repos/acme/widgets/milestones',
            '/repos/acme/widgets/pulls/1', '/repos/acme/widgets/pulls/1/reviews', '/search/issues?q=is%3Apr+review-requested%3Abob', '/notifications']) {
            expect((await api(path)).status, path).toBe(200);
        }

        // # Check org membership and create an issue and a comment
        // * Verify the membership check and both writes succeed
        expect((await api('/orgs/acme/members/bob')).status).toBe(204);
        expect((await api('/repos/acme/widgets/issues', {method: 'POST', body: JSON.stringify({title: 'New', labels: ['bug']})})).status).toBe(201);
        expect((await api('/repos/acme/widgets/issues/1/comments', {method: 'POST', body: JSON.stringify({body: 'hi'})})).status).toBe(201);
    });

    /**
     * @objective Verify the fake answers bad credentials, missing scopes and SAML enforcement with GitHub's statuses and headers.
     */
    test('rejects bad credentials, missing scopes and SAML-enforced access like GitHub', {tag: '@github_plugin'}, async () => {
        // # Call the API with an unknown token
        const bad = await fetch(`${base}/api/user`, {headers: {Authorization: 'Bearer gho_nope'}});

        // * Verify it answers 401 Bad credentials
        expect(bad.status).toBe(401);
        expect(await bad.json()).toMatchObject({message: 'Bad credentials'});

        // # Revoke the token and query GraphQL
        await control('/revoke', {token});

        // * Verify GraphQL answers 401
        expect((await api('/graphql', {method: 'POST', body: JSON.stringify({query: '{ search(type: ISSUE, query: "x", first: 1) { issueCount } }'})})).status).toBe(401);
        token = (await control('/tokens', {login: 'alice'})).value;

        // # Call routes with scopes that do not cover them
        // * Verify repo hooks answer 404, orgs answer 403 and the granted scopes are reported
        expect((await api('/repos/acme/widgets/hooks')).status).toBe(404);
        await control('/users', {login: 'dave'});
        const narrow = (await control('/tokens', {login: 'dave', scopes: ['public_repo']})).value;
        expect((await fetch(`${base}/api/user/orgs`, {headers: {Authorization: `Bearer ${narrow}`}})).status).toBe(403);
        expect((await api('/user')).headers.get('x-oauth-scopes')).toBe('admin:org_hook, notifications, public_repo, read:org');

        // # Enforce SAML SSO on the org and read a repo
        await control('/sso', {org: 'acme', enforced: true});
        const saml = await api('/repos/acme/widgets');

        // * Verify the read answers 403 with the SSO authorization URL
        expect(saml.status).toBe(403);
        expect(saml.headers.get('x-github-sso')).toMatch(/^required; url=https:\/\/github\.com\/orgs\/acme\/sso\?authorization_request=/);

        // # Search across the SSO-enforced org
        const search = await api('/search/issues?q=author%3Aalice');

        // * Verify search returns partial results with the SSO header
        expect((await search.json()).total_count).toBe(0);
        expect(search.headers.get('x-github-sso')).toMatch(/^partial-results; organizations=\d+$/);

        // # Read the repo as a user outside the org
        await control('/users', {login: 'carol'});
        const outsider = (await control('/tokens', {login: 'carol'})).value;

        // * Verify the outsider is not blocked by SSO
        expect((await fetch(`${base}/api/repos/acme/widgets`, {headers: {Authorization: `Bearer ${outsider}`}})).status).toBe(200);
    });

    /**
     * @objective Verify posting an issue comment through the API delivers an issue_comment event to the repository hook.
     */
    test('delivers issue_comment to the repository hook when commenting through the API', {tag: '@github_plugin'}, async () => {
        // # Register a repository hook and comment on an issue
        await control('/hooks', {owner: 'acme', repo: 'widgets', url: `${base}/`, secret: 's'});
        expect((await api('/repos/acme/widgets/issues/2/comments', {method: 'POST', body: JSON.stringify({body: 'on it'})})).status).toBe(201);
        await fetch(`${base}/__control/settle`, {method: 'POST'});

        // * Verify the hook received exactly that comment
        const delivered = (await (await fetch(`${base}/__control/requests`)).json()).filter((r: {method: string; path: string; body?: {comment?: {body: string}}}) => r.method === 'POST' && r.path === '/');
        expect(delivered.map((r: {body?: {comment?: {body: string}}}) => r.body?.comment?.body)).toEqual(['on it']);
    });

    /**
     * @objective Verify search results paginate with GitHub-style Link headers.
     */
    test('paginates search results with Link headers', {tag: '@github_plugin'}, async () => {
        // # Seed enough issues for three pages
        for (let i = 0; i < 4; i++) {
            await control('/items', {repo: 'acme/widgets', kind: 'issue', title: `Bulk ${i}`, author: 'alice'});
        }

        // # Request the first and last pages
        const first = await api('/search/issues?q=repo%3Aacme%2Fwidgets&per_page=2');
        const last = await api('/search/issues?q=repo%3Aacme%2Fwidgets&per_page=2&page=3');

        // * Verify only the first page links to a next page and the last page holds the remainder
        expect(first.headers.get('link')).toContain('rel="next"');
        expect(last.headers.get('link')).not.toContain('rel="next"');
        expect((await last.json()).items).toHaveLength(2);
    });

    /**
     * @objective Verify the plugin's sidebar and digest GraphQL query validates against GitHub's schema and returns user and team reviewers.
     */
    test('validates and executes the sidebar and digest GraphQL query against GitHub\'s schema', {tag: '@github_plugin'}, async () => {
        const query = `query($q: String!, $c: String) { pullRequest: search(first: 100, after: $c, query: $q, type: ISSUE) { issueCount pageInfo { hasNextPage endCursor }
            nodes { ... on PullRequest { body number authorAssociation createdAt updatedAt repository { name nameWithOwner url owner { login } } state title author { login } url
            labels(first: 100) { nodes { name color } } milestone { title } additions deletions changedFiles
            reviewRequests(first: 100) { nodes { requestedReviewer { __typename ... on User { login } ... on Team { slug organization { login } } } } } }
            ... on Issue { number title state url milestone { title } } } } }`;

        // * Verify the query is valid against the schema
        expect(validateGraphQLQuery(query).errors).toEqual([]);

        // # Run the query for bob's review requests
        const res = await api('/graphql', {method: 'POST', body: JSON.stringify({query, variables: {q: 'review-requested:bob is:pr is:OPEN archived:false org:acme'}})});
        const {data} = await res.json();

        // * Verify it returns the pull request with both user and team reviewers
        expect(data.pullRequest.issueCount).toBe(1);
        expect(data.pullRequest.nodes[0].reviewRequests.nodes.map((n: any) => n.requestedReviewer)).toEqual([
            {__typename: 'User', login: 'bob'},
            {__typename: 'Team', slug: 'core', organization: {login: 'acme'}},
        ]);
    });

    /**
     * @objective Verify GraphQL rejects unknown fields and connections without first or last, as GitHub does.
     */
    test('rejects GraphQL queries with unknown fields or missing pagination bounds', {tag: '@github_plugin'}, async () => {
        // * Verify an unknown field fails validation
        expect(validateGraphQLQuery('{ search(first: 1, query: "x", type: ISSUE) { nodes { ... on Issue { nope } } } }').errors).toHaveLength(1);

        // # Run a search without first or last
        const res = await api('/graphql', {method: 'POST', body: JSON.stringify({query: '{ search(query: "x", type: ISSUE) { issueCount } }'})});

        // * Verify it answers 200 with GitHub's pagination error
        expect(res.status).toBe(200);
        expect((await res.json()).errors[0].message).toContain('You must provide a `first` or `last` value');
    });

    /**
     * @objective Verify OAuth token exchange errors come back as HTTP 200, encoded as form or JSON depending on the Accept header.
     */
    test('returns OAuth token exchange errors as HTTP 200 in form or JSON by Accept header', {tag: '@github_plugin'}, async () => {
        // # Exchange a bad code without an Accept header
        const form = await fetch(`${base}/login/oauth/access_token`, {method: 'POST', body: new URLSearchParams({client_id: app.clientId, client_secret: app.clientSecret, code: 'nope'})});

        // * Verify a form-encoded bad_verification_code error with status 200
        expect(form.status).toBe(200);
        expect(Object.fromEntries(new URLSearchParams(await form.text()))).toMatchObject({error: 'bad_verification_code'});

        // # Exchange with a wrong client secret asking for JSON
        const json = await fetch(`${base}/login/oauth/access_token`, {method: 'POST', headers: {Accept: 'application/json'}, body: new URLSearchParams({client_id: app.clientId, client_secret: 'wrong', code: 'nope'})});

        // * Verify a JSON incorrect_client_credentials error
        expect(await json.json()).toMatchObject({error: 'incorrect_client_credentials'});
    });

    /**
     * @objective Verify the fake's webhook payloads match GitHub's event schemas and are signed with both SHA-1 and SHA-256.
     */
    test('builds webhook payloads that match GitHub\'s event schemas and carry both signatures', {tag: '@github_plugin'}, async () => {
        const state = fake.state;

        // # Build a payload for each event the plugin handles
        // * Verify each payload validates against its event schema
        for (const request of [
            {event: 'pull_request', action: 'opened', number: 1},
            {event: 'pull_request', action: 'review_requested', number: 1, requestedReviewer: 'bob'},
            {event: 'pull_request_review', action: 'submitted', number: 1, comment: 'Looks good'},
            {event: 'issues', action: 'opened', number: 2},
            {event: 'issues', action: 'closed', number: 2},
            {event: 'issues', action: 'labeled', number: 2, label: 'bug'},
            {event: 'issue_comment', action: 'created', number: 2},
            {event: 'push'},
            {event: 'create'},
        ]) {
            const payload = buildPayload(state, {repo: 'acme/widgets', sender: 'alice', ...request});
            expect(validateWebhook(request.event, payload), `${request.event}.${request.action}`).toEqual([]);
        }

        // * Verify both signature headers are produced
        expect(signatures('secret', '{}')).toEqual({
            'X-Hub-Signature': expect.stringMatching(/^sha1=[0-9a-f]{40}$/),
            'X-Hub-Signature-256': expect.stringMatching(/^sha256=[0-9a-f]{64}$/),
        });
    });

    /**
     * @objective Verify creating an org hook sends GitHub's ping delivery to the hook URL.
     */
    test('sends a ping delivery when an org hook is created', {tag: '@github_plugin'}, async () => {
        // # Start a sink to receive deliveries
        const received: string[] = [];
        const {createServer} = await import('node:http');
        const sink = createServer((req, res) => {
            received.push(String(req.headers['x-github-event']));
            res.end();
        });
        await new Promise<void>((resolve) => sink.listen(8194, resolve));
        try {
            // # Create an org hook pointing at the sink
            const res = await api('/orgs/acme/hooks', {method: 'POST', body: JSON.stringify({name: 'web', config: {url: 'http://localhost:8194/hook', content_type: 'json', secret: 's'}, events: ['*']})});

            // * Verify the hook is created and the sink receives a ping
            expect(res.status).toBe(201);
            await expect.poll(() => received).toEqual(['ping']);
        } finally {
            sink.close();
        }
    });
});

test.describe('fake semantics', () => {
    /**
     * @objective Verify search parsing treats qualifiers case-insensitively, ORs repeated org: and resolves @me to the viewer.
     */
    test('parses search qualifiers case-insensitively, ORs repeated org: and resolves @me to the viewer', {tag: '@github_plugin'}, () => {
        // * Verify the parsed qualifiers
        expect(parseSearch('is:pr is:OPEN org:a org:b author:@me "two words"', 'alice')).toEqual([
            {qualifier: 'is', values: ['pr'], negated: false},
            {qualifier: 'is', values: ['OPEN'], negated: false},
            {qualifier: 'org', values: ['a', 'b'], negated: false},
            {qualifier: 'author', values: ['alice'], negated: false},
            {qualifier: 'text', values: ['two words'], negated: false},
        ]);
    });

    /**
     * @objective Verify scope normalization drops scopes already covered by a broader grant.
     */
    test('folds away scopes covered by a broader grant', {tag: '@github_plugin'}, () => {
        // * Verify repo absorbs public_repo and admin:org absorbs read:org
        expect(normalizeScopes(['public_repo', 'repo', 'read:org', 'notifications'])).toEqual(['notifications', 'read:org', 'repo']);
        expect(normalizeScopes(['read:org', 'admin:org'])).toEqual(['admin:org']);
    });

    /**
     * @objective Verify the response and webhook skeletons validate against the schemas they are generated from.
     */
    test('validates response and webhook skeletons against their own schemas', {tag: '@github_plugin'}, () => {
        // * Verify the user and ping skeletons have no violations
        expect(validateResponse('GET', '/user', 200, skeleton('private-user'))).toEqual([]);
        expect(validateWebhook('ping', webhookSkeleton('ping'))).toEqual([]);
    });

    /**
     * @objective Verify the validator catches responses and payloads GitHub would never send, so a passing check means something.
     */
    test('rejects responses and payloads GitHub would never send', {tag: '@github_plugin'}, () => {
        // # Build an issues payload with an invalid action
        const payload = buildPayload(seededState(), {event: 'issues', action: 'opened', repo: 'acme/widgets', number: 1, sender: 'alice'});

        // * Verify an incomplete user and the invalid action are both reported
        expect(validateResponse('GET', '/user', 200, {login: 'x'})).not.toEqual([]);
        expect(validateWebhook('issues', {...payload, action: 'teleported'})).not.toEqual([]);
    });
});
