// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import crypto from 'node:crypto';

import {merge, skeleton} from './contract/validator';

export const API = 'https://api.github.com';
export const WEB = 'https://github.com';

export type Account = {id: number; login: string; type: 'User' | 'Organization'; name: string | null; email: string | null; createdAt: string};
export type Org = Account & {members: Set<string>; admins: Set<string>; ssoEnforced: boolean};
export type Team = {id: number; org: string; slug: string; name: string; members: Set<string>};
export type Label = {id: number; name: string; color: string; description: string | null};
export type Milestone = {id: number; number: number; title: string; state: 'open' | 'closed'};
export type Hook = {id: number; owner: string; repo?: string; url: string; secret: string; events: string[]; active: boolean; contentType: string; createdAt: string};
export type Comment = {id: number; author: string; body: string; createdAt: string};
export type Repo = {
    id: number; owner: string; name: string; private: boolean; archived: boolean; description: string | null;
    admins: Set<string>; writers: Set<string>; labels: Label[]; milestones: Milestone[]; nextNumber: number; createdAt: string;
};
export type Item = {
    id: number; repo: string; number: number; kind: 'issue' | 'pull'; title: string; body: string;
    state: 'open' | 'closed'; merged: boolean; draft: boolean; author: string; assignees: string[]; labels: string[];
    milestone: number | null; reviewers: string[]; teamReviewers: string[]; head: string; base: string;
    additions: number; deletions: number; changedFiles: number; comments: Comment[]; createdAt: string; updatedAt: string;
    reviews: Review[]; ciState: 'success' | 'pending' | 'failure';
};
export type Review = {id: number; author: string; state: 'APPROVED' | 'CHANGES_REQUESTED' | 'COMMENTED'; submittedAt: string};
export type Thread = {id: string; login: string; repo: string; number: number | null; title: string; type: 'Issue' | 'PullRequest' | 'Commit' | 'Release'; reason: string; unread: boolean; updatedAt: string};
export type Token = {value: string; login: string; clientId: string; scopes: string[]; revoked: boolean};
export type AuthCode = {code: string; login: string; clientId: string; redirectUri: string; scopes: string[]; expiresAt: number; used: boolean};

// GitHub folds scopes that another granted scope already covers.
const impliedBy: Record<string, string> = {
    public_repo: 'repo', 'repo:status': 'repo', repo_deployment: 'repo', 'repo:invite': 'repo', security_events: 'repo',
    'read:org': 'write:org', 'write:org': 'admin:org', 'read:user': 'user', 'user:email': 'user', 'user:follow': 'user',
    'read:repo_hook': 'write:repo_hook', 'write:repo_hook': 'admin:repo_hook', 'read:public_key': 'write:public_key', 'write:public_key': 'admin:public_key',
};

export function normalizeScopes(scopes: Iterable<string>): string[] {
    const set = new Set([...scopes].map((s) => s.trim()).filter(Boolean));
    const covered = (scope: string): boolean => {
        const parent = impliedBy[scope];
        return Boolean(parent) && (set.has(parent) || covered(parent));
    };
    return [...set].filter((s) => !covered(s)).sort();
}

export function hasScope(granted: string[], ...accepted: string[]): boolean {
    return accepted.some((scope) => {
        for (let s: string | undefined = scope; s; s = impliedBy[s]) {
            if (granted.includes(s)) {
                return true;
            }
        }
        return false;
    });
}

export class State {
    accounts = new Map<string, Account>();
    orgs = new Map<string, Org>();
    teams: Team[] = [];
    repos = new Map<string, Repo>();
    items: Item[] = [];
    hooks: Hook[] = [];
    threads: Thread[] = [];
    tokens = new Map<string, Token>();
    codes = new Map<string, AuthCode>();
    grants = new Map<string, string[]>();
    signedIn: string | null = null;
    private nextId = 1000;

    id(): number {
        return this.nextId++;
    }

    now(): string {
        return new Date().toISOString().replace(/\.\d+Z$/, 'Z');
    }

    addUser(login: string, extra: Partial<Account> = {}): Account {
        const account: Account = {id: this.id(), login, type: 'User', name: null, email: null, createdAt: this.now(), ...extra};
        this.accounts.set(login.toLowerCase(), account);
        return account;
    }

    addOrg(login: string, members: string[] = [], admins: string[] = []): Org {
        const org: Org = {id: this.id(), login, type: 'Organization', name: null, email: null, createdAt: this.now(), members: new Set([...members, ...admins]), admins: new Set(admins), ssoEnforced: false};
        this.accounts.set(login.toLowerCase(), org);
        this.orgs.set(login.toLowerCase(), org);
        return org;
    }

    account(login: string): Account | undefined {
        return this.accounts.get(login.toLowerCase());
    }

    addRepo(repoName: string, options: Partial<Pick<Repo, 'private' | 'archived' | 'description'>> & {admins?: string[]; writers?: string[]; labels?: string[]; milestones?: string[]} = {}): Repo {
        const [owner, name] = repoName.split('/');
        const ownerAccount = this.account(owner);
        if (!ownerAccount) {
            throw new Error(`unknown owner ${owner}`);
        }
        const admins = new Set(options.admins ?? (ownerAccount.type === 'User' ? [owner] : [...this.orgs.get(owner.toLowerCase())!.admins]));
        const repo: Repo = {
            id: this.id(), owner: ownerAccount.login, name, private: options.private ?? false, archived: options.archived ?? false,
            description: options.description ?? null, admins, writers: new Set([...admins, ...(options.writers ?? [])]),
            labels: (options.labels ?? ['bug', 'enhancement']).map((l, i) => ({id: this.id(), name: l, color: ['d73a4a', 'a2eeef', '0075ca', '7057ff'][i % 4], description: null})),
            milestones: (options.milestones ?? []).map((title, i) => ({id: this.id(), number: i + 1, title, state: 'open' as const})),
            nextNumber: 1, createdAt: this.now(),
        };
        this.repos.set(repoName.toLowerCase(), repo);
        return repo;
    }

    repo(owner: string, name: string): Repo | undefined {
        return this.repos.get(`${owner}/${name}`.toLowerCase());
    }

    addItem(repo: Repo, fields: Partial<Item> & Pick<Item, 'kind' | 'title' | 'author'>): Item {
        const now = this.now();
        const number = repo.nextNumber++;
        const item: Item = {
            id: this.id(), repo: fullName(repo), number, body: '', state: 'open', merged: false, draft: false,
            assignees: [], labels: [], milestone: null, reviewers: [], teamReviewers: [], head: `feature-${number}`, base: 'main',
            additions: 10, deletions: 2, changedFiles: 1, comments: [], reviews: [], ciState: 'success', createdAt: now, updatedAt: now, ...fields,
        };
        this.items.push(item);
        return item;
    }

    item(owner: string, name: string, number: number): Item | undefined {
        const key = `${owner}/${name}`.toLowerCase();
        return this.items.find((i) => i.repo.toLowerCase() === key && i.number === number);
    }

    repoOf(item: Item): Repo {
        return this.repos.get(item.repo.toLowerCase())!;
    }

    issueToken(login: string, clientId: string, scopes: string[]): Token {
        const value = `gho_${crypto.randomBytes(27).toString('base64url').replace(/[-_]/g, 'x').slice(0, 36)}`;
        const token: Token = {value, login, clientId, scopes: normalizeScopes(scopes), revoked: false};
        this.tokens.set(value, token);
        return token;
    }

    // A classic OAuth app's token carries every scope the user ever granted that app.
    grant(login: string, clientId: string, requested: string[]): string[] {
        const key = `${clientId}:${login.toLowerCase()}`;
        const scopes = normalizeScopes([...(this.grants.get(key) ?? []), ...requested]);
        this.grants.set(key, scopes);
        return scopes;
    }

    canSee(viewer: Token | null, repo: Repo | undefined): boolean {
        if (!repo) {
            return false;
        }
        if (!repo.private) {
            return true;
        }
        if (!viewer || !hasScope(viewer.scopes, 'repo')) {
            return false;
        }
        const org = this.orgs.get(repo.owner.toLowerCase());
        return repo.writers.has(viewer.login) || repo.owner === viewer.login || Boolean(org?.members.has(viewer.login));
    }

    permissions(viewer: Token | null, repo: Repo) {
        const login = viewer?.login ?? '';
        const admin = repo.admins.has(login);
        const push = repo.writers.has(login);
        return {admin, maintain: admin, push, triage: push, pull: true};
    }

    userJSON(account: Account, schema: 'simple-user' | 'public-user' | 'private-user' = 'simple-user') {
        const login = account.login;
        const detail = schema === 'simple-user' ? {} : {
            name: account.name, email: account.email, created_at: account.createdAt, updated_at: account.createdAt,
            public_repos: [...this.repos.values()].filter((r) => r.owner === login && !r.private).length,
        };
        return merge(skeleton(schema), {
            login, id: account.id, node_id: nodeId(account.type === 'User' ? 'U' : 'O', account.id), type: account.type, gravatar_id: '',
            avatar_url: `${avatarBase}/u/${account.id}?v=4`, html_url: `${WEB}/${login}`, url: `${API}/users/${login}`,
            repos_url: `${API}/users/${login}/repos`, organizations_url: `${API}/users/${login}/orgs`,
            ...detail,
        });
    }

    login(login: string): Account {
        return this.account(login) ?? this.addUser(login);
    }

    orgJSON(org: Org, schema: 'organization-simple' | 'organization-full' = 'organization-simple') {
        return merge(skeleton(schema), {
            login: org.login, id: org.id, node_id: nodeId('O', org.id), url: `${API}/orgs/${org.login}`,
            repos_url: `${API}/orgs/${org.login}/repos`, hooks_url: `${API}/orgs/${org.login}/hooks`,
            avatar_url: `${avatarBase}/u/${org.id}?v=4`, description: null,
            ...(schema === 'organization-full' ? {html_url: `${WEB}/${org.login}`, type: 'Organization', created_at: org.createdAt, updated_at: org.createdAt} : {}),
        });
    }

    repoJSON(repo: Repo, viewer: Token | null, schema: 'minimal-repository' | 'repository' | 'full-repository' = 'repository') {
        const name = fullName(repo);
        return merge(skeleton(schema), {
            id: repo.id, node_id: nodeId('R', repo.id), name: repo.name, full_name: name, private: repo.private, archived: repo.archived,
            visibility: repo.private ? 'private' : 'public', description: repo.description, default_branch: 'main',
            owner: this.userJSON(this.account(repo.owner)!),
            html_url: `${WEB}/${name}`, url: `${API}/repos/${name}`, hooks_url: `${API}/repos/${name}/hooks`,
            issues_url: `${API}/repos/${name}/issues{/number}`, pulls_url: `${API}/repos/${name}/pulls{/number}`,
            clone_url: `${WEB}/${name}.git`, svn_url: `${WEB}/${name}`, git_url: `git://github.com/${name}.git`, ssh_url: `git@github.com:${name}.git`,
            has_issues: true, open_issues_count: this.items.filter((i) => i.repo === name && i.state === 'open').length,
            created_at: repo.createdAt, updated_at: repo.createdAt, pushed_at: repo.createdAt,
            permissions: this.permissions(viewer, repo),
        });
    }

    labelJSON(repo: Repo, label: Label) {
        return merge(skeleton('label'), {id: label.id, node_id: nodeId('LA', label.id), url: `${API}/repos/${fullName(repo)}/labels/${encodeURIComponent(label.name)}`, name: label.name, color: label.color, description: label.description});
    }

    milestoneJSON(repo: Repo, milestone: Milestone) {
        const base = `${API}/repos/${fullName(repo)}/milestones/${milestone.number}`;
        return merge(skeleton('milestone'), {
            id: milestone.id, node_id: nodeId('MI', milestone.id), number: milestone.number, title: milestone.title, state: milestone.state,
            url: base, labels_url: `${base}/labels`, html_url: `${WEB}/${fullName(repo)}/milestone/${milestone.number}`,
            created_at: repo.createdAt, updated_at: repo.createdAt,
        });
    }

    issueJSON(item: Item, schema: 'issue' | 'issue-search-result-item' = 'issue') {
        const repo = this.repoOf(item);
        const name = fullName(repo);
        const kindPath = item.kind === 'pull' ? 'pull' : 'issues';
        const milestone = repo.milestones.find((m) => m.number === item.milestone);
        return merge(skeleton(schema), {
            id: item.id, node_id: nodeId(item.kind === 'pull' ? 'PR' : 'I', item.id), number: item.number, title: item.title, body: item.body,
            state: item.state, state_reason: null, draft: item.kind === 'pull' ? item.draft : undefined, locked: false, comments: item.comments.length,
            user: this.userJSON(this.login(item.author)), assignee: item.assignees[0] ? this.userJSON(this.login(item.assignees[0])) : null,
            assignees: item.assignees.map((a) => this.userJSON(this.login(a))),
            labels: item.labels.map((l) => this.labelJSON(repo, repo.labels.find((x) => x.name === l) ?? {id: 0, name: l, color: 'ededed', description: null})),
            milestone: milestone ? this.milestoneJSON(repo, milestone) : null,
            url: `${API}/repos/${name}/issues/${item.number}`, html_url: `${WEB}/${name}/${kindPath}/${item.number}`,
            repository_url: `${API}/repos/${name}`, comments_url: `${API}/repos/${name}/issues/${item.number}/comments`,
            labels_url: `${API}/repos/${name}/issues/${item.number}/labels{/name}`, events_url: `${API}/repos/${name}/issues/${item.number}/events`,
            author_association: this.association(repo, item.author),
            closed_at: item.state === 'closed' ? item.updatedAt : null, created_at: item.createdAt, updated_at: item.updatedAt,
            ...(item.kind === 'pull' ? {pull_request: {url: `${API}/repos/${name}/pulls/${item.number}`, html_url: `${WEB}/${name}/pull/${item.number}`, diff_url: `${WEB}/${name}/pull/${item.number}.diff`, patch_url: `${WEB}/${name}/pull/${item.number}.patch`, merged_at: item.merged ? item.updatedAt : null}} : {}),
            ...(schema === 'issue-search-result-item' ? {score: 1} : {}),
        });
    }

    pullJSON(item: Item, viewer: Token | null) {
        const repo = this.repoOf(item);
        const name = fullName(repo);
        const base = `${API}/repos/${name}/pulls/${item.number}`;
        const ref = (branch: string, commit: string) => ({label: `${repo.owner}:${branch}`, ref: branch, sha: commit, user: this.userJSON(this.account(repo.owner)!), repo: this.repoJSON(repo, viewer)});
        const link = (href: string) => ({href});
        const issue = this.issueJSON(item);
        return merge(skeleton('pull-request'), {
            ...issue, pull_request: undefined, repository_url: undefined, labels_url: undefined, events_url: undefined, comments: item.comments.length,
            url: base, html_url: `${WEB}/${name}/pull/${item.number}`, diff_url: `${WEB}/${name}/pull/${item.number}.diff`, patch_url: `${WEB}/${name}/pull/${item.number}.patch`,
            issue_url: `${API}/repos/${name}/issues/${item.number}`, commits_url: `${base}/commits`, review_comments_url: `${base}/comments`,
            review_comment_url: `${API}/repos/${name}/pulls/comments{/number}`, comments_url: `${API}/repos/${name}/issues/${item.number}/comments`,
            statuses_url: `${API}/repos/${name}/statuses/${sha(item, 'head')}`,
            head: ref(item.head, sha(item, 'head')), base: ref(item.base, sha(item, 'base')),
            _links: {self: link(base), html: link(`${WEB}/${name}/pull/${item.number}`), issue: link(`${API}/repos/${name}/issues/${item.number}`), comments: link(`${API}/repos/${name}/issues/${item.number}/comments`), review_comments: link(`${base}/comments`), review_comment: link(`${API}/repos/${name}/pulls/comments{/number}`), commits: link(`${base}/commits`), statuses: link(`${API}/repos/${name}/statuses/${sha(item, 'head')}`)},
            requested_reviewers: item.reviewers.map((r) => this.userJSON(this.login(r))),
            requested_teams: item.teamReviewers.map((slug) => this.teamJSON(this.teams.find((t) => t.org.toLowerCase() === repo.owner.toLowerCase() && t.slug === slug)!)),
            merged: item.merged, merged_at: item.merged ? item.updatedAt : null, mergeable: item.state === 'open' ? true : null, mergeable_state: 'clean',
            merge_commit_sha: null, draft: item.draft, auto_merge: null, merged_by: null,
            additions: item.additions, deletions: item.deletions, changed_files: item.changedFiles, commits: 1, review_comments: 0, maintainer_can_modify: false,
        });
    }

    reviewJSON(item: Item, review: Review) {
        const html = `${WEB}/${item.repo}/pull/${item.number}#pullrequestreview-${review.id}`;
        return merge(skeleton('pull-request-review'), {
            id: review.id, node_id: nodeId('PRR', review.id), user: this.userJSON(this.login(review.author)), body: '', state: review.state,
            html_url: html, pull_request_url: `${API}/repos/${item.repo}/pulls/${item.number}`, submitted_at: review.submittedAt, commit_id: sha(item, 'head'),
            author_association: this.association(this.repoOf(item), review.author),
            _links: {html: {href: html}, pull_request: {href: `${API}/repos/${item.repo}/pulls/${item.number}`}},
        });
    }

    combinedStatusJSON(item: Item) {
        const repo = this.repoOf(item);
        const head = sha(item, 'head');
        const url = `${API}/repos/${item.repo}/commits/${head}/status`;
        return merge(skeleton('combined-commit-status'), {
            state: item.ciState, sha: head, total_count: 1, commit_url: `${API}/repos/${item.repo}/commits/${head}`, url,
            repository: this.repoJSON(repo, null, 'minimal-repository'),
            statuses: [merge(skeleton('simple-commit-status'), {id: item.id, node_id: nodeId('SC', item.id), state: item.ciState, context: 'ci/e2e', description: null, target_url: null, avatar_url: null, url, created_at: item.updatedAt, updated_at: item.updatedAt})],
        });
    }

    teamJSON(team: Team) {
        const url = `${API}/orgs/${team.org}/teams/${team.slug}`;
        return merge(skeleton('team-simple'), {id: team.id, node_id: nodeId('T', team.id), name: team.name, slug: team.slug, url, html_url: `${WEB}/orgs/${team.org}/teams/${team.slug}`, members_url: `${url}/members{/member}`, repositories_url: `${url}/repos`, permission: 'pull', type: 'organization', description: null});
    }

    hookJSON(hook: Hook, schema: 'hook' | 'org-hook') {
        const base = hook.repo ? `${API}/repos/${hook.owner}/${hook.repo}/hooks/${hook.id}` : `${API}/orgs/${hook.owner}/hooks/${hook.id}`;
        return merge(skeleton(schema), {
            id: hook.id, type: hook.repo ? 'Repository' : 'Organization', name: 'web', active: hook.active, events: hook.events,
            config: {url: hook.url, content_type: hook.contentType, insecure_ssl: '0', ...(hook.secret ? {secret: '********'} : {})},
            url: base, ping_url: `${base}/pings`, deliveries_url: `${base}/deliveries`, created_at: hook.createdAt, updated_at: hook.createdAt,
            ...(schema === 'hook' ? {test_url: `${base}/test`, last_response: {code: null, status: 'unused', message: null}} : {}),
        });
    }

    threadJSON(thread: Thread, viewer: Token | null) {
        const repo = this.repos.get(thread.repo.toLowerCase())!;
        const subjectURL = thread.number ? `${API}/repos/${thread.repo}/${thread.type === 'PullRequest' ? 'pulls' : 'issues'}/${thread.number}` : `${API}/repos/${thread.repo}`;
        return merge(skeleton('thread'), {
            id: thread.id, unread: thread.unread, reason: thread.reason, updated_at: thread.updatedAt, last_read_at: null,
            subject: {title: thread.title, type: thread.type, url: subjectURL, latest_comment_url: subjectURL},
            repository: this.repoJSON(repo, viewer, 'minimal-repository'),
            url: `${API}/notifications/threads/${thread.id}`, subscription_url: `${API}/notifications/threads/${thread.id}/subscription`,
        });
    }

    commentJSON(item: Item, comment: Comment) {
        const name = item.repo;
        return merge(skeleton('issue-comment'), {
            id: comment.id, node_id: nodeId('IC', comment.id), body: comment.body, user: this.userJSON(this.login(comment.author)),
            url: `${API}/repos/${name}/issues/comments/${comment.id}`, html_url: `${WEB}/${name}/${item.kind === 'pull' ? 'pull' : 'issues'}/${item.number}#issuecomment-${comment.id}`,
            issue_url: `${API}/repos/${name}/issues/${item.number}`, author_association: this.association(this.repoOf(item), comment.author),
            created_at: comment.createdAt, updated_at: comment.createdAt,
        });
    }

    association(repo: Repo, login: string): string {
        if (repo.owner === login) {
            return 'OWNER';
        }
        if (this.orgs.get(repo.owner.toLowerCase())?.members.has(login)) {
            return 'MEMBER';
        }
        return repo.writers.has(login) ? 'COLLABORATOR' : 'NONE';
    }

    search(query: string, viewer: Token | null): Item[] {
        const terms = parseSearch(query, viewer?.login ?? '');
        return this.items.
            filter((item) => this.canSee(viewer, this.repoOf(item)) && terms.every((term) => term.negated !== this.matches(item, term))).
            sort((a, b) => b.updatedAt.localeCompare(a.updatedAt) || b.id - a.id);
    }

    private matches(item: Item, {qualifier, values}: SearchTerm): boolean {
        const repo = this.repoOf(item);
        const eq = (a: string, b: string) => a.toLowerCase() === b.toLowerCase();
        return values.some((value) => {
            switch (qualifier) {
            case 'is':
            case 'type':
            case 'state':
                return matchesIs(item, repo, value.toLowerCase());
            case 'author':
                return eq(item.author, value);
            case 'assignee':
                return item.assignees.some((a) => eq(a, value));
            case 'review-requested':
                return item.reviewers.some((r) => eq(r, value)) ||
                    item.teamReviewers.some((slug) => this.teams.some((t) => t.org.toLowerCase() === repo.owner.toLowerCase() && t.slug === slug && [...t.members].some((m) => eq(m, value))));
            case 'mentions':
                return new RegExp(`@${escapeRegExp(value)}\\b`, 'i').test(`${item.body}\n${item.comments.map((c) => c.body).join('\n')}`);
            case 'involves':
                return eq(item.author, value) || item.assignees.some((a) => eq(a, value)) || new RegExp(`@${escapeRegExp(value)}\\b`, 'i').test(item.body);
            case 'org':
            case 'user':
                return eq(repo.owner, value);
            case 'repo':
                return eq(item.repo, value);
            case 'archived':
                return String(repo.archived) === value.toLowerCase();
            case 'draft':
                return item.kind === 'pull' && String(item.draft) === value.toLowerCase();
            case 'label':
                return item.labels.some((l) => eq(l, value));
            case 'text':
                return `${item.title}\n${item.body}`.toLowerCase().includes(value.toLowerCase());
            default:
                return false;
            }
        });
    }
}

type SearchTerm = {qualifier: string; values: string[]; negated: boolean};

const knownQualifiers = new Set(['is', 'type', 'state', 'author', 'assignee', 'review-requested', 'mentions', 'involves', 'org', 'user', 'repo', 'archived', 'draft', 'label']);
const orQualifiers = new Set(['org', 'user', 'repo']);

// GitHub ANDs qualifiers, except repeated org:/user:/repo:, which it ORs. Qualifier values are case-insensitive.
export function parseSearch(query: string, viewer: string): SearchTerm[] {
    const terms: SearchTerm[] = [];
    for (const raw of query.match(/(?:[^\s"]+|"[^"]*")+/g) ?? []) {
        const negated = raw.startsWith('-');
        const token = negated ? raw.slice(1) : raw;
        const colon = token.indexOf(':');
        const qualifier = colon > 0 ? token.slice(0, colon).toLowerCase() : '';
        let value = (knownQualifiers.has(qualifier) ? token.slice(colon + 1) : token).replace(/^"|"$/g, '');
        if (value === '@me') {
            value = viewer;
        }
        const key = knownQualifiers.has(qualifier) ? qualifier : 'text';
        const existing = orQualifiers.has(key) && !negated ? terms.find((t) => t.qualifier === key && !t.negated) : undefined;
        if (existing) {
            existing.values.push(value);
        } else {
            terms.push({qualifier: key, values: [value], negated});
        }
    }
    return terms;
}

function matchesIs(item: Item, repo: Repo, value: string): boolean {
    switch (value) {
    case 'pr':
    case 'pull-request':
        return item.kind === 'pull';
    case 'issue':
        return item.kind === 'issue';
    case 'open':
        return item.state === 'open';
    case 'closed':
        return item.state === 'closed';
    case 'merged':
        return item.merged;
    case 'unmerged':
        return item.kind === 'pull' && !item.merged;
    case 'draft':
        return item.draft;
    case 'public':
        return !repo.private;
    case 'private':
        return repo.private;
    default:
        return false;
    }
}

export function fullName(repo: Repo): string {
    return `${repo.owner}/${repo.name}`;
}

export function sha(item: Item, side: 'head' | 'base'): string {
    return crypto.createHash('sha1').update(`${item.repo}#${item.number}:${side}`).digest('hex');
}

function nodeId(prefix: string, id: number): string {
    return `${prefix}_kgDO${Buffer.from(String(id)).toString('base64url')}`;
}

let avatarBase = 'http://fake-github:8093/avatars';

export function setAvatarBase(url: string) {
    avatarBase = url;
}

function escapeRegExp(value: string): string {
    return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}
