// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import crypto from 'node:crypto';
import {type IncomingMessage, type Server, type ServerResponse, createServer} from 'node:http';

import {GraphQLError, execute} from 'graphql';

import {graphQLSchema, validateGraphQLQuery, validateRequest, validateResponse, validateWebhook} from './contract/validator';
import {type AuthCode, type Hook, type Item, type Org, type Repo, State, type Token, WEB, fullName, hasScope, normalizeScopes, setAvatarBase, sha} from './state';
import {type Delivery, type WebhookRequest, buildPayload, deliver, pingPayload} from './webhooks';

export type OAuthApp = {clientId: string; clientSecret: string; callbackURL: string; name: string; codeTTLms?: number};
export type JournalEntry = {method: string; path: string; query: Record<string, string>; status: number; body?: unknown};
type Fault = {method: string; path: string; pattern: RegExp; status: number; body?: unknown; headers?: Record<string, string>; times: number};

type Ctx = {
    req: IncomingMessage; res: ServerResponse; url: URL; params: string[]; body: any;
    token: Token | null; send: (status: number, body?: unknown, headers?: Record<string, string>) => void;
};
type Handler = (ctx: Ctx) => void | Promise<void>;

const docs = 'https://docs.github.com/rest';
const ssoMessage = 'Resource protected by organization SAML enforcement. You must grant your OAuth token access to this organization.';

export type FakeGitHub = {state: State; close: () => Promise<void>};

export async function startFakeGitHub({port, publicURL, app}: {port: number; publicURL: string; app: OAuthApp}): Promise<FakeGitHub> {
    let state = new State();
    let oauthApp = app;
    let journal: JournalEntry[] = [];
    let violations: string[] = [];
    let faults: Fault[] = [];
    let pending: Promise<unknown>[] = [];
    let rateLimit = new Map<string, number>();
    setAvatarBase(`${publicURL}/avatars`);

    const fail = (ctx: Ctx, status: number, message: string, headers?: Record<string, string>) => ctx.send(status, {message, documentation_url: docs, status: String(status)}, headers);
    const notFound = (ctx: Ctx) => fail(ctx, 404, 'Not Found');

    const requireToken = (ctx: Ctx): Token | null => {
        if (!ctx.token) {
            fail(ctx, 401, ctx.req.headers.authorization ? 'Bad credentials' : 'Requires authentication');
            return null;
        }
        return ctx.token;
    };

    const ssoBlocked = (ctx: Ctx, owner: string): boolean => {
        const org = state.orgs.get(owner.toLowerCase());
        if (!org?.ssoEnforced || !ctx.token || !org.members.has(ctx.token.login)) {
            return false;
        }
        fail(ctx, 403, ssoMessage, {'X-GitHub-SSO': `required; url=${WEB}/orgs/${org.login}/sso?authorization_request=${crypto.randomBytes(12).toString('hex')}`});
        return true;
    };

    const visibleRepo = (ctx: Ctx, owner: string, name: string): Repo | null => {
        const repo = state.repo(owner, name);
        if (ssoBlocked(ctx, owner)) {
            return null;
        }
        if (!repo || !state.canSee(ctx.token, repo)) {
            notFound(ctx);
            return null;
        }
        return repo;
    };

    const paginate = (ctx: Ctx, all: unknown[], {defaultSize = 30, maxSize = 100} = {}) => {
        const perPage = Math.min(Number(ctx.url.searchParams.get('per_page') ?? defaultSize) || defaultSize, maxSize);
        const page = Math.max(Number(ctx.url.searchParams.get('page') ?? 1) || 1, 1);
        const last = Math.max(Math.ceil(all.length / perPage), 1);
        const link = (p: number, rel: string) => {
            const next = new URL(`${publicURL}${ctx.url.pathname}${ctx.url.search}`);
            next.searchParams.set('page', String(p));
            return `<${next}>; rel="${rel}"`;
        };
        const links = [page < last && link(page + 1, 'next'), page < last && link(last, 'last'), page > 1 && link(1, 'first'), page > 1 && link(page - 1, 'prev')].filter(Boolean);
        const headers: Record<string, string> = links.length ? {Link: links.join(', ')} : {};
        return {items: all.slice((page - 1) * perPage, page * perPage), headers};
    };

    const sendPage = (ctx: Ctx, all: unknown[], sizes?: {defaultSize?: number; maxSize?: number}) => {
        const {items, headers} = paginate(ctx, all, sizes);
        ctx.send(200, items, headers);
    };

    const track = (delivery: Promise<unknown>) => pending.push(delivery.catch((err) => violations.push(`fake-github failed to deliver a webhook: ${err}`)));

    const hooksFor = (repo: Repo): Hook[] => state.hooks.filter((h) => h.owner.toLowerCase() === repo.owner.toLowerCase() && (!h.repo || h.repo.toLowerCase() === repo.name.toLowerCase()));

    const emit = (repo: Repo, request: WebhookRequest) => {
        const hooks = hooksFor(repo);
        if (!hooks.length) {
            return;
        }
        const payload = buildPayload(state, request);
        violations.push(...validateWebhook(request.event, payload));
        track(deliver(hooks, request.event, payload));
    };

    const routes: [string, RegExp, Handler][] = [];
    const route = (method: string, pattern: string, handler: Handler) => {
        routes.push([method, new RegExp(`^${pattern.replace(/:\w+/g, '([^/]+)')}$`), handler]);
    };

    route('GET', '/login/oauth/authorize', (ctx) => {
        const q = ctx.url.searchParams;
        if (q.get('client_id') !== oauthApp.clientId) {
            return ctx.send(404, htmlPage('404', '<h1>Not Found</h1>'));
        }
        const state_ = q.get('state') ?? '';
        const redirectURI = q.get('redirect_uri') || oauthApp.callbackURL;
        if (!redirectMatches(redirectURI, oauthApp.callbackURL)) {
            return redirect(ctx, oauthApp.callbackURL, {error: 'redirect_uri_mismatch', error_description: 'The redirect_uri MUST match the registered callback URL for this application.', error_uri: 'https://docs.github.com/apps/managing-oauth-apps/troubleshooting-authorization-request-errors/#redirect-uri-mismatch', state: state_});
        }
        if (!state.signedIn) {
            return ctx.send(200, htmlPage('Sign in to GitHub', `<form method="post" action="/session"><input type="hidden" name="return_to" value="${escape(ctx.url.pathname + ctx.url.search)}"><label for="login_field">Username or email address</label><input id="login_field" name="login"><label for="password">Password</label><input id="password" name="password" type="password"><button type="submit">Sign in</button></form>`));
        }
        const scopes = parseScopeParam(q.get('scope'));
        const previous = state.grants.get(`${oauthApp.clientId}:${state.signedIn.toLowerCase()}`);
        if (previous && scopes.every((s) => hasScope(previous, s))) {
            return issueCode(ctx, redirectURI, scopes, state_);
        }
        return ctx.send(200, htmlPage(`Authorize ${oauthApp.name}`, `<h1>Authorize ${escape(oauthApp.name)}</h1><p>Signed in as <strong>${escape(state.signedIn)}</strong></p><ul>${scopes.map((s) => `<li>${escape(s)}</li>`).join('')}</ul><form method="post" action="/login/oauth/authorize">${hidden({client_id: oauthApp.clientId, redirect_uri: redirectURI, scope: scopes.join(' '), state: state_})}<button type="submit" name="authorize" value="0">Cancel</button><button type="submit" name="authorize" value="1">Authorize ${escape(oauthApp.name)}</button></form>`));
    });

    route('POST', '/login/oauth/authorize', (ctx) => {
        const {client_id: clientId, redirect_uri: redirectURI, scope, state: state_, authorize} = ctx.body;
        if (clientId !== oauthApp.clientId || !state.signedIn || !redirectURI || !redirectMatches(redirectURI, oauthApp.callbackURL)) {
            return ctx.send(422, htmlPage('Unprocessable', '<h1>Unprocessable request</h1>'));
        }
        if (authorize !== '1') {
            return redirect(ctx, redirectURI, {error: 'access_denied', error_description: 'The user has denied your application access.', error_uri: 'https://docs.github.com/apps/managing-oauth-apps/troubleshooting-authorization-request-errors/#access-denied', state: state_});
        }
        return issueCode(ctx, redirectURI, parseScopeParam(scope), state_);
    });

    route('POST', '/session', (ctx) => {
        state.signedIn = state.login(String(ctx.body.login)).login;
        ctx.send(302, '', {Location: String(ctx.body.return_to || '/')});
    });

    // GitHub answers token-exchange failures with 200 and an error body.
    route('POST', '/login/oauth/access_token', (ctx) => {
        const basic = /^Basic (.+)$/.exec(ctx.req.headers.authorization ?? '')?.[1];
        const [headerId, headerSecret] = basic ? Buffer.from(basic, 'base64').toString().split(':').map(decodeURIComponent) : [];
        const clientId = ctx.body.client_id ?? headerId;
        const clientSecret = ctx.body.client_secret ?? headerSecret;
        const json = (ctx.req.headers.accept ?? '').includes('application/json');
        const reply = (fields: Record<string, string>) => (json ? ctx.send(200, fields) : ctx.send(200, new URLSearchParams(fields).toString(), {'Content-Type': 'application/x-www-form-urlencoded; charset=utf-8'}));
        const oauthError = (error: string, description: string) => reply({error, error_description: description, error_uri: `https://docs.github.com/apps/managing-oauth-apps/troubleshooting-oauth-app-access-token-request-errors/#${error.replaceAll('_', '-')}`});

        if (clientId !== oauthApp.clientId || clientSecret !== oauthApp.clientSecret) {
            return oauthError('incorrect_client_credentials', 'The client_id and/or client_secret passed are incorrect.');
        }
        const code = state.codes.get(String(ctx.body.code));
        if (!code || code.used || code.expiresAt <= Date.now() || code.clientId !== clientId) {
            return oauthError('bad_verification_code', 'The code passed is incorrect or expired.');
        }
        if (ctx.body.redirect_uri && ctx.body.redirect_uri !== code.redirectUri) {
            return oauthError('redirect_uri_mismatch', 'The redirect_uri MUST match the registered callback URL for this application.');
        }
        code.used = true;
        const token = state.issueToken(code.login, clientId, state.grant(code.login, clientId, code.scopes));
        return reply({access_token: token.value, token_type: 'bearer', scope: token.scopes.join(',')});
    });

    route('GET', '/api/user', (ctx) => {
        const token = requireToken(ctx);
        if (token) {
            ctx.send(200, state.userJSON(state.login(token.login), 'private-user'));
        }
    });

    route('GET', '/api/users/:username', (ctx) => {
        const account = state.account(ctx.params[0]);
        return account ? ctx.send(200, state.userJSON(account, 'public-user')) : notFound(ctx);
    });

    route('GET', '/api/user/orgs', (ctx) => {
        const token = requireToken(ctx);
        if (!token) {
            return;
        }
        if (!hasScope(token.scopes, 'read:org', 'user')) {
            return fail(ctx, 403, 'You need at least read:org scope or user scope to list your organizations.');
        }
        sendPage(ctx, [...state.orgs.values()].filter((o) => o.members.has(token.login)).map((o) => state.orgJSON(o)));
    });

    route('GET', '/api/user/repos', (ctx) => {
        const token = requireToken(ctx);
        if (!token) {
            return;
        }
        const affiliation = (ctx.url.searchParams.get('affiliation') ?? 'owner,collaborator,organization_member').split(',');
        const repos = [...state.repos.values()].filter((r) => state.canSee(token, r) && (
            (affiliation.includes('owner') && r.owner === token.login) ||
            (affiliation.includes('collaborator') && r.writers.has(token.login) && r.owner !== token.login) ||
            (affiliation.includes('organization_member') && Boolean(state.orgs.get(r.owner.toLowerCase())?.members.has(token.login)))));
        sendPage(ctx, sortRepos(repos).map((r) => state.repoJSON(r, token)));
    });

    route('GET', '/api/users/:username/repos', (ctx) => {
        const account = state.account(ctx.params[0]);
        if (!account) {
            return notFound(ctx);
        }
        sendPage(ctx, sortRepos([...state.repos.values()].filter((r) => r.owner === account.login && !r.private)).map((r) => state.repoJSON(r, ctx.token, 'minimal-repository')));
    });

    route('GET', '/api/orgs/:org', (ctx) => {
        const org = state.orgs.get(ctx.params[0].toLowerCase());
        if (!org) {
            return notFound(ctx);
        }
        if (!ssoBlocked(ctx, org.login)) {
            ctx.send(200, state.orgJSON(org, 'organization-full'));
        }
    });

    route('GET', '/api/orgs/:org/repos', (ctx) => {
        const org = state.orgs.get(ctx.params[0].toLowerCase());
        if (!org) {
            return notFound(ctx);
        }
        if (!ssoBlocked(ctx, org.login)) {
            sendPage(ctx, sortRepos([...state.repos.values()].filter((r) => r.owner === org.login && state.canSee(ctx.token, r))).map((r) => state.repoJSON(r, ctx.token, 'minimal-repository')));
        }
    });

    // GitHub redirects non-members to the public membership check; members get a direct answer.
    route('GET', '/api/orgs/:org/members/:username', (ctx) => {
        const org = state.orgs.get(ctx.params[0].toLowerCase());
        if (!org) {
            return notFound(ctx);
        }
        if (!ctx.token || !org.members.has(ctx.token.login)) {
            return ctx.send(302, '', {Location: `${publicURL}/api/orgs/${org.login}/public_members/${ctx.params[1]}`});
        }
        return org.members.has(state.account(ctx.params[1])?.login ?? '') ? ctx.send(204) : ctx.send(404, {message: 'User does not exist or is not a member of the organization', documentation_url: docs, status: '404'});
    });

    route('GET', '/api/orgs/:org/public_members/:username', (ctx) => notFound(ctx));

    route('GET', '/api/orgs/:org/hooks', (ctx) => {
        const org = orgHookAccess(ctx);
        if (org) {
            sendPage(ctx, state.hooks.filter((h) => !h.repo && h.owner === org.login).map((h) => state.hookJSON(h, 'org-hook')));
        }
    });

    route('POST', '/api/orgs/:org/hooks', (ctx) => {
        const org = orgHookAccess(ctx);
        if (org) {
            createHook(ctx, org.login, undefined);
        }
    });

    route('GET', '/api/repos/:owner/:repo', (ctx) => {
        const repo = visibleRepo(ctx, ctx.params[0], ctx.params[1]);
        if (repo) {
            ctx.send(200, state.repoJSON(repo, ctx.token, 'full-repository'));
        }
    });

    route('GET', '/api/repos/:owner/:repo/hooks', (ctx) => {
        const repo = repoHookAccess(ctx, 'read:repo_hook');
        if (repo) {
            sendPage(ctx, state.hooks.filter((h) => h.repo === repo.name && h.owner === repo.owner).map((h) => state.hookJSON(h, 'hook')));
        }
    });

    route('POST', '/api/repos/:owner/:repo/hooks', (ctx) => {
        const repo = repoHookAccess(ctx, 'write:repo_hook');
        if (repo) {
            createHook(ctx, repo.owner, repo.name);
        }
    });

    route('GET', '/api/repos/:owner/:repo/issues/:number', (ctx) => {
        const repo = visibleRepo(ctx, ctx.params[0], ctx.params[1]);
        const item = repo && state.item(repo.owner, repo.name, Number(ctx.params[2]));
        if (repo) {
            return item ? ctx.send(200, state.issueJSON(item)) : notFound(ctx);
        }
    });

    route('POST', '/api/repos/:owner/:repo/issues', (ctx) => {
        const token = requireToken(ctx);
        const repo = token && visibleRepo(ctx, ctx.params[0], ctx.params[1]);
        if (!token || !repo) {
            return;
        }
        if (!hasScope(token.scopes, repo.private ? 'repo' : 'public_repo')) {
            return notFound(ctx);
        }
        if (!ctx.body?.title) {
            return ctx.send(422, {message: 'Validation Failed', errors: [{resource: 'Issue', code: 'missing_field', field: 'title'}], documentation_url: docs, status: '422'});
        }
        const canPush = repo.writers.has(token.login);
        const item = state.addItem(repo, {
            kind: 'issue', title: ctx.body.title, body: ctx.body.body ?? '', author: token.login,
            ...(canPush ? {
                labels: (ctx.body.labels ?? []).map((l: string | {name: string}) => (typeof l === 'string' ? l : l.name)),
                assignees: [...(ctx.body.assignees ?? []), ...(ctx.body.assignee ? [ctx.body.assignee] : [])].filter((a: string) => repo.writers.has(a)),
                milestone: ctx.body.milestone ?? null,
            } : {}),
        });
        ctx.send(201, state.issueJSON(item));
        emit(repo, {event: 'issues', action: 'opened', repo: fullName(repo), number: item.number, sender: token.login});
    });

    route('POST', '/api/repos/:owner/:repo/issues/:number/comments', (ctx) => {
        const token = requireToken(ctx);
        const repo = token && visibleRepo(ctx, ctx.params[0], ctx.params[1]);
        const item = repo && state.item(repo.owner, repo.name, Number(ctx.params[2]));
        if (!token || !repo) {
            return;
        }
        if (!item || !hasScope(token.scopes, repo.private ? 'repo' : 'public_repo')) {
            return notFound(ctx);
        }
        const comment = {id: state.id(), author: token.login, body: String(ctx.body?.body ?? ''), createdAt: state.now()};
        if (!comment.body) {
            return ctx.send(422, {message: 'Validation Failed', errors: [{resource: 'IssueComment', code: 'missing_field', field: 'body'}], documentation_url: docs, status: '422'});
        }
        item.comments.push(comment);
        item.updatedAt = comment.createdAt;
        ctx.send(201, state.commentJSON(item, comment));
        emit(repo, {event: 'issue_comment', action: 'created', repo: fullName(repo), number: item.number, sender: token.login, commentId: comment.id});
    });

    route('GET', '/api/repos/:owner/:repo/labels', (ctx) => {
        const repo = visibleRepo(ctx, ctx.params[0], ctx.params[1]);
        if (repo) {
            sendPage(ctx, repo.labels.map((l) => state.labelJSON(repo, l)));
        }
    });

    route('GET', '/api/repos/:owner/:repo/assignees', (ctx) => {
        const repo = visibleRepo(ctx, ctx.params[0], ctx.params[1]);
        if (repo) {
            sendPage(ctx, [...repo.writers].sort().map((login) => state.userJSON(state.login(login))));
        }
    });

    route('GET', '/api/repos/:owner/:repo/milestones', (ctx) => {
        const repo = visibleRepo(ctx, ctx.params[0], ctx.params[1]);
        if (repo) {
            const wanted = ctx.url.searchParams.get('state') ?? 'open';
            sendPage(ctx, repo.milestones.filter((m) => wanted === 'all' || m.state === wanted).map((m) => state.milestoneJSON(repo, m)));
        }
    });

    route('GET', '/api/repos/:owner/:repo/pulls/:number', (ctx) => {
        const repo = visibleRepo(ctx, ctx.params[0], ctx.params[1]);
        const item = repo && state.item(repo.owner, repo.name, Number(ctx.params[2]));
        if (repo) {
            return item?.kind === 'pull' ? ctx.send(200, state.pullJSON(item, ctx.token)) : notFound(ctx);
        }
    });

    route('GET', '/api/repos/:owner/:repo/pulls/:number/reviews', (ctx) => {
        const repo = visibleRepo(ctx, ctx.params[0], ctx.params[1]);
        const item = repo && state.item(repo.owner, repo.name, Number(ctx.params[2]));
        if (repo) {
            return item?.kind === 'pull' ? sendPage(ctx, item.reviews.map((r) => state.reviewJSON(item, r))) : notFound(ctx);
        }
    });

    route('GET', '/api/repos/:owner/:repo/commits/:ref/status', (ctx) => {
        const repo = visibleRepo(ctx, ctx.params[0], ctx.params[1]);
        const item = repo && state.items.find((i) => i.repo === fullName(repo) && [sha(i, 'head'), i.head].includes(ctx.params[2]));
        if (repo) {
            return item ? ctx.send(200, state.combinedStatusJSON(item)) : notFound(ctx);
        }
    });

    route('GET', '/api/search/issues', (ctx) => {
        const q = ctx.url.searchParams.get('q');
        if (!q) {
            return ctx.send(422, {message: 'Validation Failed', errors: [{resource: 'Search', field: 'q', code: 'missing'}], documentation_url: docs, status: '422'});
        }
        const {visible, ssoHeader} = searchVisible(ctx, q);
        const {items, headers} = paginate(ctx, visible.map((i) => state.issueJSON(i, 'issue-search-result-item')));
        ctx.send(200, {total_count: visible.length, incomplete_results: false, search_type: 'lexical', items}, {...headers, ...ssoHeader});
    });

    route('GET', '/api/notifications', (ctx) => {
        const token = requireToken(ctx);
        if (!token) {
            return;
        }
        if (!hasScope(token.scopes, 'notifications', 'repo')) {
            return fail(ctx, 403, "Missing the 'notifications' scope.");
        }
        const all = ctx.url.searchParams.get('all') === 'true';
        const threads = state.threads.filter((t) => t.login === token.login && (all || t.unread) && state.canSee(token, state.repos.get(t.repo.toLowerCase())));
        sendPage(ctx, threads.sort((a, b) => b.updatedAt.localeCompare(a.updatedAt)).map((t) => state.threadJSON(t, token)), {defaultSize: 50, maxSize: 50});
    });

    route('POST', '/api/graphql', async (ctx) => {
        if (!ctx.token) {
            return ctx.send(401, {message: 'This endpoint requires you to be authenticated.', documentation_url: 'https://docs.github.com/graphql/guides/forming-calls-with-graphql#authenticating-with-graphql', status: '401'});
        }
        const {document, errors} = validateGraphQLQuery(String(ctx.body?.query ?? ''));
        if (!document || errors.length) {
            violations.push(...errors.map((e) => `graphql query: ${e}`));
            return ctx.send(200, {errors: errors.map((message) => ({message}))});
        }
        const result = await execute({schema: graphQLSchema, document, variableValues: ctx.body.variables, rootValue: graphQLRoot(ctx)});
        for (const error of result.errors ?? []) {
            if (!(error.originalError instanceof GitHubGraphQLError)) {
                violations.push(`graphql answer: ${error.message} at ${error.path?.join('.')}`);
            }
        }
        ctx.send(200, result);
    });

    route('GET', '/avatars/u/:id', (ctx) => {
        const hue = (Number(ctx.params[0]) * 47) % 360;
        ctx.send(200, `<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64"><rect width="64" height="64" fill="hsl(${hue},55%,55%)"/></svg>`, {'Content-Type': 'image/svg+xml', 'Cache-Control': 'max-age=3600'});
    });

    route('GET', '/', (ctx) => ctx.send(200, 'ok', {'Content-Type': 'text/plain'}));

    route('POST', '/__control/reset', async (ctx) => {
        await Promise.allSettled(pending);
        state = new State();
        journal = [];
        violations = [];
        faults = [];
        pending = [];
        rateLimit = new Map();
        oauthApp = {...app, ...ctx.body};
        ctx.send(200, {});
    });
    route('POST', '/__control/oauth-app', (ctx) => {
        oauthApp = {...oauthApp, ...ctx.body};
        ctx.send(200, {});
    });
    route('GET', '/__control/requests', (ctx) => ctx.send(200, journal));
    route('GET', '/__control/contract-violations', (ctx) => ctx.send(200, violations));
    route('POST', '/__control/users', (ctx) => ctx.send(200, state.addUser(ctx.body.login, {name: ctx.body.name ?? null, email: ctx.body.email ?? null})));
    route('POST', '/__control/orgs', (ctx) => {
        const org = state.addOrg(ctx.body.login, ctx.body.members ?? [], ctx.body.admins ?? []);
        org.ssoEnforced = Boolean(ctx.body.ssoEnforced);
        ctx.send(200, {...org, members: [...org.members], admins: [...org.admins]});
    });
    route('POST', '/__control/sso', (ctx) => {
        state.orgs.get(String(ctx.body.org).toLowerCase())!.ssoEnforced = Boolean(ctx.body.enforced);
        ctx.send(200, {});
    });
    route('POST', '/__control/teams', (ctx) => {
        const team = {id: state.id(), org: ctx.body.org, slug: ctx.body.slug, name: ctx.body.name ?? ctx.body.slug, members: new Set<string>(ctx.body.members ?? [])};
        state.teams.push(team);
        ctx.send(200, {...team, members: [...team.members]});
    });
    route('POST', '/__control/repos', (ctx) => {
        const repo = state.addRepo(ctx.body.fullName, ctx.body);
        ctx.send(200, {fullName: fullName(repo), id: repo.id});
    });
    route('POST', '/__control/items', (ctx) => {
        const [owner, name] = String(ctx.body.repo).split('/');
        const {repo: _, ...fields} = ctx.body;
        ctx.send(200, state.addItem(state.repo(owner, name)!, fields));
    });
    route('PATCH', '/__control/items', (ctx) => {
        const [owner, name] = String(ctx.body.repo).split('/');
        const item = state.item(owner, name, ctx.body.number);
        if (!item) {
            return ctx.send(404, {});
        }
        const {repo: _, number: __, ...patch} = ctx.body;
        Object.assign(item, patch, {updatedAt: state.now()});
        ctx.send(200, item);
    });
    route('GET', '/__control/items', (ctx) => ctx.send(200, state.items));
    route('POST', '/__control/threads', (ctx) => {
        const item = ctx.body.number ? state.item(...(String(ctx.body.repo).split('/') as [string, string]), ctx.body.number) : undefined;
        const thread = {
            id: String(state.id()), login: ctx.body.login, repo: ctx.body.repo, number: ctx.body.number ?? null, reason: ctx.body.reason ?? 'mention',
            title: ctx.body.title ?? item?.title ?? 'Notification', type: ctx.body.type ?? (item?.kind === 'pull' ? 'PullRequest' : 'Issue'), unread: true, updatedAt: state.now(),
        };
        state.threads.push(thread);
        ctx.send(200, thread);
    });
    route('POST', '/__control/hooks', (ctx) => {
        const hook: Hook = {id: state.id(), owner: ctx.body.owner, repo: ctx.body.repo, url: ctx.body.url, secret: ctx.body.secret ?? '', events: ctx.body.events ?? ['*'], active: true, contentType: ctx.body.contentType ?? 'json', createdAt: state.now()};
        state.hooks.push(hook);
        ctx.send(200, hook);
    });
    route('GET', '/__control/hooks', (ctx) => ctx.send(200, state.hooks));
    route('POST', '/__control/tokens', (ctx) => {
        const {login} = state.login(ctx.body.login);
        const scopes = state.grant(login, oauthApp.clientId, ctx.body.scopes ?? defaultScopes);
        ctx.send(200, state.issueToken(login, oauthApp.clientId, scopes));
    });
    route('POST', '/__control/grants', (ctx) => ctx.send(200, {scopes: state.grant(ctx.body.login, oauthApp.clientId, ctx.body.scopes)}));
    route('POST', '/__control/revoke', (ctx) => {
        const token = state.tokens.get(ctx.body.token);
        if (token) {
            token.revoked = true;
        }
        ctx.send(200, {});
    });
    route('POST', '/__control/sign-in', (ctx) => {
        state.signedIn = ctx.body.login ? state.login(ctx.body.login).login : null;
        ctx.send(200, {});
    });
    route('POST', '/__control/fault', (ctx) => {
        let pattern: RegExp;
        try {
            pattern = new RegExp(`^${ctx.body.path}$`);
        } catch (err) {
            return ctx.send(400, {message: String(err)});
        }
        faults.push({times: Infinity, ...ctx.body, pattern});
        return ctx.send(200, {});
    });
    route('POST', '/__control/webhooks', async (ctx) => {
        const request = ctx.body as WebhookRequest & {deliver?: boolean};
        const payload = buildPayload(state, request);
        violations.push(...validateWebhook(request.event, payload));
        let deliveries: Delivery[] = [];
        if (request.deliver !== false) {
            const [owner, name] = request.repo.split('/');
            deliveries = await deliver(hooksFor(state.repo(owner, name)!), request.event, payload);
        }
        ctx.send(200, {payload, deliveries});
    });
    route('POST', '/__control/settle', async (ctx) => {
        await Promise.allSettled(pending);
        ctx.send(200, {});
    });

    function orgHookAccess(ctx: Ctx): Org | null {
        const token = requireToken(ctx);
        const org = state.orgs.get(ctx.params[0].toLowerCase());
        if (!token || ssoBlocked(ctx, ctx.params[0])) {
            return null;
        }
        if (!org || !org.admins.has(token.login) || !hasScope(token.scopes, 'admin:org_hook')) {
            notFound(ctx);
            return null;
        }
        return org;
    }

    function repoHookAccess(ctx: Ctx, scope: string): Repo | null {
        const token = requireToken(ctx);
        const repo = token && visibleRepo(ctx, ctx.params[0], ctx.params[1]);
        if (!token || !repo) {
            return null;
        }
        if (!repo.admins.has(token.login) || !hasScope(token.scopes, scope, 'repo')) {
            notFound(ctx);
            return null;
        }
        return repo;
    }

    function createHook(ctx: Ctx, owner: string, repo: string | undefined) {
        const config = ctx.body?.config ?? {};
        if (!config.url) {
            return ctx.send(422, {message: 'Validation Failed', errors: [{resource: 'Hook', code: 'custom', message: 'Config must contain URL for webhooks'}], documentation_url: docs, status: '422'});
        }
        const existing = state.hooks.find((h) => h.owner === owner && h.repo === repo && h.url === config.url);
        if (existing) {
            return ctx.send(422, {message: 'Validation Failed', errors: [{resource: 'Hook', code: 'custom', message: `Hook already exists on this ${repo ? 'repository' : 'organization'}`}], documentation_url: docs, status: '422'});
        }
        const hook: Hook = {id: state.id(), owner, repo, url: config.url, secret: config.secret ?? '', events: ctx.body.events ?? ['push'], active: ctx.body.active ?? true, contentType: config.content_type ?? 'form', createdAt: state.now()};
        state.hooks.push(hook);
        ctx.send(201, state.hookJSON(hook, repo ? 'hook' : 'org-hook'));
        const payload = pingPayload(state, hook, ctx.token!.login);
        violations.push(...validateWebhook('ping', payload));
        track(deliver([hook], 'ping', payload));
    }

    function issueCode(ctx: Ctx, redirectURI: string, scopes: string[], stateParam: string) {
        const code: AuthCode = {code: crypto.randomBytes(10).toString('hex'), login: state.signedIn!, clientId: oauthApp.clientId, redirectUri: redirectURI, scopes, expiresAt: Date.now() + (oauthApp.codeTTLms ?? 10 * 60_000), used: false};
        state.codes.set(code.code, code);
        redirect(ctx, redirectURI, {code: code.code, state: stateParam});
    }

    // Search drops results from SAML-protected orgs the token isn't authorized for, and says so in a header.
    function searchVisible(ctx: Ctx, query: string): {visible: Item[]; ssoHeader: Record<string, string>} {
        const hits = state.search(query, ctx.token);
        const blocked = new Set<Org>();
        const visible = hits.filter((item) => {
            const org = state.orgs.get(state.repoOf(item).owner.toLowerCase());
            if (org?.ssoEnforced && ctx.token && org.members.has(ctx.token.login)) {
                blocked.add(org);
                return false;
            }
            return true;
        });
        return {visible, ssoHeader: blocked.size ? {'X-GitHub-SSO': `partial-results; organizations=${[...blocked].map((o) => o.id).join(',')}`} : {}};
    }

    function graphQLRoot(ctx: Ctx) {
        return {
            search: ({query, type, first, last, after}: {query: string; type: string; first?: number; last?: number; after?: string}) => {
                if (first === undefined && last === undefined) {
                    throw new GitHubGraphQLError('You must provide a `first` or `last` value to properly paginate the `search` connection.');
                }
                if (last !== undefined) {
                    throw new Error('fake-github does not implement `last` on search; add it if the plugin starts using it');
                }
                const limit = first!;
                if (limit > 100) {
                    throw new GitHubGraphQLError(`Requesting ${limit} records on the \`search\` connection exceeds the \`${first === undefined ? 'last' : 'first'}\` limit of 100 records.`);
                }
                const hits = type === 'ISSUE' ? searchVisible(ctx, query).visible : [];
                const start = after ? Number(Buffer.from(after, 'base64').toString().replace('cursor:', '')) : 0;
                const nodes = hits.slice(start, start + limit);
                const end = start + nodes.length;
                return {
                    issueCount: hits.length,
                    nodes: nodes.map(graphQLItem),
                    pageInfo: {endCursor: nodes.length ? Buffer.from(`cursor:${end}`).toString('base64') : null, hasNextPage: end < hits.length},
                };
            },
        };
    }

    function graphQLItem(item: Item) {
        const repo = state.repoOf(item);
        const actor = (login: string) => ({__typename: state.account(login)?.type === 'Organization' ? 'Organization' : 'User', login});
        const issue = state.issueJSON(item);
        const milestone = repo.milestones.find((m) => m.number === item.milestone);
        return {
            __typename: item.kind === 'pull' ? 'PullRequest' : 'Issue',
            number: item.number, title: item.title, body: item.body, url: issue.html_url,
            state: item.merged ? 'MERGED' : item.state.toUpperCase(), createdAt: item.createdAt, updatedAt: item.updatedAt,
            authorAssociation: state.association(repo, item.author), author: actor(item.author),
            repository: {name: repo.name, nameWithOwner: fullName(repo), url: `${WEB}/${fullName(repo)}`, owner: actor(repo.owner)},
            milestone: milestone ? {title: milestone.title} : null,
            labels: ({first}: {first?: number}) => ({nodes: item.labels.slice(0, first ?? 100).map((name) => ({name, color: repo.labels.find((l) => l.name === name)?.color ?? 'ededed'}))}),
            additions: item.additions, deletions: item.deletions, changedFiles: item.changedFiles,
            reviewRequests: ({first}: {first?: number}) => ({
                nodes: [
                    ...item.reviewers.map((login) => ({requestedReviewer: {__typename: 'User', login}})),
                    ...item.teamReviewers.map((slug) => ({requestedReviewer: {__typename: 'Team', slug, organization: {__typename: 'Organization', login: repo.owner}}})),
                ].slice(0, first ?? 100),
            }),
        };
    }

    const listener = async (req: IncomingMessage, res: ServerResponse) => {
        const url = new URL(req.url ?? '/', publicURL);
        const rawBody = await readBody(req);
        const isControl = url.pathname.startsWith('/__control/');
        let body: any = {};
        if (rawBody) {
            body = (req.headers['content-type'] ?? '').includes('json') ? safeJSON(rawBody) : Object.fromEntries(new URLSearchParams(rawBody));
        }
        const bearer = /^(?:Bearer|token) (.+)$/i.exec(req.headers.authorization ?? '')?.[1];
        const known = bearer ? state.tokens.get(bearer) : undefined;
        const token = known && !known.revoked ? known : null;
        const apiPath = url.pathname.startsWith('/api/') ? url.pathname.slice(4) : null;

        let skipContract = false;
        const send: Ctx['send'] = (status, payload, headers = {}) => {
            if (res.headersSent) {
                return;
            }
            const isText = typeof payload === 'string';
            const out = payload === undefined ? '' : isText ? payload : JSON.stringify(payload);
            if (apiPath && apiPath !== '/graphql' && !isText && !skipContract) {
                violations.push(...validateResponse(req.method!, apiPath, status, payload));
            }
            res.writeHead(status, {
                'Content-Type': isText ? 'text/html; charset=utf-8' : 'application/json; charset=utf-8',
                ...(apiPath ? apiHeaders(token, rateLimit) : {}),
                ...headers,
            });
            res.end(out);
            if (!isControl) {
                journal.push({method: req.method!, path: url.pathname, query: Object.fromEntries(url.searchParams), status, body: rawBody ? body : undefined});
            }
        };
        const ctx: Ctx = {req, res, url, params: [], body, token, send};

        try {
            if (apiPath && bearer && !token) {
                return send(401, {message: 'Bad credentials', documentation_url: docs, status: '401'});
            }
            if (body === invalidJSON) {
                violations.push(`${req.method} ${url.pathname}: the plugin sent a body that is not JSON`);
                skipContract = true;
                return send(400, {message: 'Problems parsing JSON', documentation_url: docs, status: '400'});
            }
            if (apiPath && apiPath !== '/graphql' && !isControl && req.method !== 'GET' && rawBody) {
                violations.push(...validateRequest(req.method!, apiPath, body));
            }
            const fault = !isControl && faults.find((f) => f.times > 0 && f.method === req.method && f.pattern.test(url.pathname));
            if (fault) {
                fault.times--;
                skipContract = true;
                return send(fault.status, fault.body ?? {message: 'Injected fault', documentation_url: docs, status: String(fault.status)}, fault.headers);
            }
            for (const [method, pattern, handler] of routes) {
                const match = method === req.method && pattern.exec(url.pathname);
                if (match) {
                    ctx.params = match.slice(1).map(decodeURIComponent);
                    await handler(ctx);
                    return;
                }
            }
            if (apiPath) {
                violations.push(`${req.method} ${apiPath} is not served by fake-github`);
                skipContract = true;
            }
            send(404, {message: 'Not Found', documentation_url: docs, status: '404'});
        } catch (err) {
            violations.push(`fake-github crashed on ${req.method} ${url.pathname}: ${(err as Error).stack}`);
            send(500, {message: String(err)});
        }
    };

    const server: Server = createServer((req, res) => {
        listener(req, res).catch((err) => {
            res.statusCode = 500;
            res.end(String(err));
        });
    });
    await new Promise<void>((resolve, reject) => {
        server.once('error', reject);
        server.listen(port, resolve);
    });

    return {
        get state() {
            return state;
        },
        close: () => new Promise((resolve) => server.close(() => resolve())),
    };
}

export const defaultScopes = ['public_repo', 'notifications', 'read:org', 'admin:org_hook'];

class GitHubGraphQLError extends GraphQLError {}

function apiHeaders(token: Token | null, rateLimit: Map<string, number>): Record<string, string> {
    const key = token?.value ?? 'anonymous';
    const used = (rateLimit.get(key) ?? 0) + 1;
    rateLimit.set(key, used);
    const limit = token ? 5000 : 60;
    return {
        'X-GitHub-Media-Type': 'github.v3; format=json',
        'X-GitHub-Api-Version-Selected': '2022-11-28',
        'X-GitHub-Request-Id': crypto.randomBytes(8).toString('hex').toUpperCase(),
        'X-RateLimit-Limit': String(limit),
        'X-RateLimit-Remaining': String(Math.max(limit - used, 0)),
        'X-RateLimit-Reset': String(Math.floor(Date.now() / 1000) + 3600),
        'X-RateLimit-Used': String(used),
        'X-RateLimit-Resource': 'core',
        ...(token ? {'X-OAuth-Scopes': normalizeScopes(token.scopes).join(', '), 'X-Accepted-OAuth-Scopes': ''} : {}),
    };
}

function redirectMatches(requested: string, registered: string): boolean {
    try {
        const a = new URL(requested);
        const b = new URL(registered);
        return a.protocol === b.protocol && a.host === b.host && (a.pathname === b.pathname || a.pathname.startsWith(`${b.pathname.replace(/\/$/, '')}/`));
    } catch {
        return false;
    }
}

function redirect(ctx: Ctx, target: string, params: Record<string, string>) {
    const url = new URL(target);
    for (const [k, v] of Object.entries(params)) {
        url.searchParams.set(k, v);
    }
    ctx.send(302, '', {Location: url.toString()});
}

function parseScopeParam(scope: string | null | undefined): string[] {
    return normalizeScopes((scope ?? '').split(/[\s,]+/));
}

function sortRepos(repos: Repo[]): Repo[] {
    return [...repos].sort((a, b) => fullName(a).localeCompare(fullName(b)));
}

function readBody(req: IncomingMessage): Promise<string> {
    return new Promise((resolve, reject) => {
        const chunks: Buffer[] = [];
        req.on('data', (c) => chunks.push(c));
        req.on('end', () => resolve(Buffer.concat(chunks).toString()));
        req.on('error', reject);
    });
}

const invalidJSON = Symbol('invalid JSON');

function safeJSON(raw: string): unknown {
    try {
        return JSON.parse(raw);
    } catch {
        return invalidJSON;
    }
}

function escape(value: string): string {
    return value.replace(/[&<>"']/g, (c) => `&#${c.charCodeAt(0)};`);
}

function hidden(fields: Record<string, string>): string {
    return Object.entries(fields).map(([k, v]) => `<input type="hidden" name="${k}" value="${escape(v)}">`).join('');
}

function htmlPage(title: string, content: string): string {
    return `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>${escape(title)}</title><style>body{font:14px -apple-system,sans-serif;max-width:420px;margin:48px auto}label,input,button{display:block;margin:8px 0}button{padding:6px 16px}</style></head><body><main>${content}</main></body></html>`;
}
