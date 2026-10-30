// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {JournalEntry} from '../fake-github/server';
import type {Account, Hook, Item, Thread, Token} from '../fake-github/state';
import type {Delivery, WebhookRequest} from '../fake-github/webhooks';

import {fakeGitHubControlURL} from './constants';

async function control<T>(method: string, path: string, body?: unknown): Promise<T> {
    const res = await fetch(`${fakeGitHubControlURL}/__control${path}`, {
        method,
        headers: {'Content-Type': 'application/json'},
        body: body === undefined ? undefined : JSON.stringify(body),
    });
    if (!res.ok) {
        throw new Error(`fake-github control ${method} ${path} failed: ${res.status} ${await res.text()}`);
    }
    return res.json() as Promise<T>;
}

type ItemFields = Partial<Omit<Item, 'id' | 'repo' | 'number'>> & Pick<Item, 'kind' | 'title' | 'author'>;

export const fakeGitHub = {
    reset: () => control('POST', '/reset'),
    requests: () => control<JournalEntry[]>('GET', '/requests'),
    calls: async () => (await fakeGitHub.requests()).map((r) => `${r.method} ${r.path}`),
    contractViolations: () => control<string[]>('GET', '/contract-violations'),
    settle: () => control('POST', '/settle'),
    createUser: (login: string, extra: {name?: string; email?: string} = {}) => control<Account>('POST', '/users', {login, ...extra}),
    createOrg: (login: string, options: {members?: string[]; admins?: string[]; ssoEnforced?: boolean} = {}) => control('POST', '/orgs', {login, ...options}),
    setSSO: (org: string, enforced: boolean) => control('POST', '/sso', {org, enforced}),
    createTeam: (org: string, slug: string, members: string[]) => control('POST', '/teams', {org, slug, members}),
    createRepo: (fullName: string, options: {private?: boolean; archived?: boolean; admins?: string[]; writers?: string[]; labels?: string[]; milestones?: string[]} = {}) => control<{fullName: string; id: number}>('POST', '/repos', {fullName, ...options}),
    createItem: (repo: string, fields: ItemFields) => control<Item>('POST', '/items', {repo, ...fields}),
    updateItem: (repo: string, number: number, patch: Partial<Item>) => control<Item>('PATCH', '/items', {repo, number, ...patch}),
    items: () => control<Item[]>('GET', '/items'),
    createThread: (login: string, repo: string, fields: Partial<Thread> = {}) => control<Thread>('POST', '/threads', {login, repo, ...fields}),
    createHook: (owner: string, repo: string | undefined, url: string, secret: string, events = ['*']) => control<Hook>('POST', '/hooks', {owner, repo, url, secret, events}),
    hooks: () => control<Hook[]>('GET', '/hooks'),
    issueToken: (login: string, scopes?: string[]) => control<Token>('POST', '/tokens', {login, scopes}),
    grant: (login: string, scopes: string[]) => control<{scopes: string[]}>('POST', '/grants', {login, scopes}),
    revoke: (token: string) => control('POST', '/revoke', {token}),
    signIn: (login: string | null) => control('POST', '/sign-in', {login}),
    setOAuthApp: (patch: {codeTTLms?: number; clientSecret?: string}) => control('POST', '/oauth-app', patch),
    fault: (method: string, path: string, status: number, options: {body?: unknown; headers?: Record<string, string>; times?: number} = {}) => control('POST', '/fault', {method, path, status, ...options}),
    webhook: (request: WebhookRequest & {deliver?: boolean}) => control<{payload: Record<string, unknown>; deliveries: Delivery[]}>('POST', '/webhooks', request),
};
