// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import crypto from 'node:crypto';

import {merge, skeleton, webhookSchemaName, webhookSkeleton} from './contract/validator';
import {type Hook, type Item, type Repo, type State, WEB, fullName, sha} from './state';

export type WebhookRequest = {event: string; action?: string; repo: string; number?: number; sender: string; requestedReviewer?: string; comment?: string; commentId?: number; ref?: string; overrides?: Record<string, unknown>};

export function buildPayload(state: State, req: WebhookRequest): Record<string, unknown> {
    const [owner, name] = req.repo.split('/');
    const repo = state.repo(owner, name);
    if (!repo) {
        throw new Error(`unknown repo ${req.repo}`);
    }
    const schemaName = webhookSchemaName(req.event, req.action);
    if (!schemaName) {
        throw new Error(`webhook ${req.event}.${req.action} is not in operations.json`);
    }
    const item = req.number === undefined ? undefined : state.item(owner, name, req.number);
    const common = {
        repository: repositoryWebhook(state, repo),
        sender: state.userJSON(state.login(req.sender)),
        ...(state.orgs.has(repo.owner.toLowerCase()) ? {organization: merge(skeleton('organization-simple-webhooks'), state.orgJSON(state.orgs.get(repo.owner.toLowerCase())!))} : {}),
    };

    let specific: Record<string, unknown> = {};
    switch (req.event) {
    case 'pull_request':
        specific = {
            action: req.action, number: item!.number, pull_request: merge(skeleton('pull-request-webhook'), state.pullJSON(item!, null)),
            ...(req.requestedReviewer ? {requested_reviewer: state.userJSON(state.login(req.requestedReviewer))} : {}),
        };
        break;
    case 'issues':
        specific = {action: req.action, issue: issueWebhook(state, item!)};
        break;
    case 'issue_comment': {
        let comment = item!.comments.find((c) => c.id === req.commentId);
        if (!comment) {
            comment = {id: state.id(), author: req.sender, body: req.comment ?? 'A comment', createdAt: state.now()};
            item!.comments.push(comment);
        }
        specific = {action: req.action, issue: issueWebhook(state, item!), comment: state.commentJSON(item!, comment)};
        break;
    }
    case 'push':
        specific = push(state, repo, req.sender, req.ref ?? 'refs/heads/main');
        break;
    default:
        throw new Error(`no payload builder for ${req.event}`);
    }

    return merge(merge(webhookSkeleton(schemaName), {...common, ...specific}), req.overrides ?? {});
}

export function pingPayload(state: State, hook: Hook, sender: string): Record<string, unknown> {
    const repo = hook.repo ? state.repo(hook.owner, hook.repo) : undefined;
    const org = state.orgs.get(hook.owner.toLowerCase());
    return merge(webhookSkeleton('ping'), {
        zen: 'Keep it logically awesome.',
        hook_id: hook.id,
        hook: state.hookJSON(hook, hook.repo ? 'hook' : 'org-hook'),
        ...(repo ? {repository: repositoryWebhook(state, repo)} : {}),
        ...(org ? {organization: merge(skeleton('organization-simple-webhooks'), state.orgJSON(org))} : {}),
        sender: state.userJSON(state.login(sender)),
    });
}

export type Delivery = {hookId: number; url: string; status: number; body: string};

export async function deliver(hooks: Hook[], event: string, payload: Record<string, unknown>): Promise<Delivery[]> {
    const body = JSON.stringify(payload);
    return Promise.all(hooks.filter((h) => h.active && (event === 'ping' || h.events.includes(event) || h.events.includes('*'))).map(async (hook) => {
        const sent = hook.contentType === 'form' ? `payload=${encodeURIComponent(body)}` : body;
        const headers: Record<string, string> = {
            'Content-Type': hook.contentType === 'form' ? 'application/x-www-form-urlencoded' : 'application/json',
            'User-Agent': 'GitHub-Hookshot/e2e0000',
            'X-GitHub-Event': event,
            'X-GitHub-Delivery': crypto.randomUUID(),
            'X-GitHub-Hook-ID': String(hook.id),
            'X-GitHub-Hook-Installation-Target-Type': hook.repo ? 'repository' : 'organization',
        };
        if (hook.secret) {
            Object.assign(headers, signatures(hook.secret, sent));
        }
        try {
            const res = await fetch(hook.url, {method: 'POST', headers, body: sent, signal: AbortSignal.timeout(10_000)});
            return {hookId: hook.id, url: hook.url, status: res.status, body: await res.text()};
        } catch (err) {
            return {hookId: hook.id, url: hook.url, status: 0, body: String(err)};
        }
    }));
}

export function signatures(secret: string, body: string) {
    return {
        'X-Hub-Signature': `sha1=${crypto.createHmac('sha1', secret).update(body).digest('hex')}`,
        'X-Hub-Signature-256': `sha256=${crypto.createHmac('sha256', secret).update(body).digest('hex')}`,
    };
}

function repositoryWebhook(state: State, repo: Repo) {
    return merge(skeleton('repository-webhooks'), state.repoJSON(repo, null));
}

function issueWebhook(state: State, item: Item) {
    const issue = state.issueJSON(item);
    return {...issue, reactions: merge(skeleton('reaction-rollup'), {url: `${issue.url}/reactions`})};
}

function push(state: State, repo: Repo, pusher: string, ref: string) {
    const name = fullName(repo);
    const after = sha({repo: name, number: Date.now()} as Item, 'head');
    const before = sha({repo: name, number: Date.now() - 1} as Item, 'base');
    const account = state.login(pusher);
    const person = {name: account.name ?? account.login, email: account.email ?? `${account.login}@users.noreply.github.com`, username: account.login};
    const commit = {
        id: after, tree_id: after, distinct: true, message: 'Update README', timestamp: state.now(), url: `${WEB}/${name}/commit/${after}`,
        author: person, committer: person, added: [], removed: [], modified: ['README.md'],
    };
    return {
        ref, before, after, created: false, deleted: false, forced: false, base_ref: null, compare: `${WEB}/${name}/compare/${before.slice(0, 12)}...${after.slice(0, 12)}`,
        commits: [commit], head_commit: commit, pusher: {name: account.login, email: person.email},
        repository: {...repositoryWebhook(state, repo), url: `${WEB}/${name}`, created_at: Math.floor(Date.parse(repo.createdAt) / 1000), pushed_at: Math.floor(Date.now() / 1000), stargazers: 0, master_branch: 'main'},
    };
}
