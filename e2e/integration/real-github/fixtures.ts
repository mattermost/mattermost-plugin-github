// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {randomBytes} from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';

import {gh, ghOK, waitForSearch} from './api';
import {realGitHub} from './env';

export type RealState = {
    login: string;
    issue: {number: number; title: string};
    pull: {number: number; title: string};
    hookId: number;
    webhookSecret: string;
};

export const fixtureLabel = 'e2e-fixture';
const fixtureBranch = 'e2e-fixture';
const issueTitle = 'E2E fixture: assigned issue';
const pullTitle = 'E2E fixture: open pull request';

// GitHub refuses hooks to private addresses, so this one never delivers; the tests replay its recorded deliveries instead.
export const fixtureHookURL = 'https://example.com/mattermost-plugin-github-e2e';

export function readState(): RealState {
    return JSON.parse(fs.readFileSync(realGitHub.statePath, 'utf8'));
}

// Long-lived fixtures are created once and reused, so a nightly run costs a handful of writes.
export async function frontLoad(): Promise<RealState> {
    const {fullName} = realGitHub;
    const {login} = await ghOK<{login: string}>('GET', '/user');

    if ((await gh('GET', `/repos/${fullName}/labels/${fixtureLabel}`)).status === 404) {
        await ghOK('POST', `/repos/${fullName}/labels`, {body: {name: fixtureLabel, color: 'ededed', description: 'Kept open for the Mattermost plugin E2E suite'}});
    }

    const issue = await ensureIssue(login);
    const pull = await ensurePull(login);

    await waitForSearch(`repo:${fullName} is:issue is:open assignee:${login} label:${fixtureLabel}`, (i) => i.number === issue.number);
    await waitForSearch(`repo:${fullName} is:pr is:open author:${login} label:${fixtureLabel}`, (i) => i.number === pull.number);

    const hookId = await ensureHook();
    const webhookSecret = randomBytes(20).toString('hex');
    await ghOK('PATCH', `/repos/${fullName}/hooks/${hookId}/config`, {body: {url: fixtureHookURL, content_type: 'json', secret: webhookSecret}});
    const state: RealState = {login, issue, pull, hookId, webhookSecret};

    fs.mkdirSync(path.dirname(realGitHub.statePath), {recursive: true});
    fs.writeFileSync(realGitHub.statePath, JSON.stringify(state));
    return state;
}

async function ensureIssue(login: string) {
    const open = await ghOK<Array<{number: number; title: string; pull_request?: unknown}>>('GET', `/repos/${realGitHub.fullName}/issues?labels=${fixtureLabel}&state=open&per_page=100`);
    const found = open.find((i) => !i.pull_request && i.title === issueTitle);
    if (found) {
        return {number: found.number, title: found.title};
    }
    const created = await ghOK('POST', `/repos/${realGitHub.fullName}/issues`, {body: {title: issueTitle, body: 'Kept open for the E2E suite.', labels: [fixtureLabel], assignees: [login]}});
    return {number: created.number, title: issueTitle};
}

async function ensurePull(login: string) {
    const {fullName, org} = realGitHub;
    const open = await ghOK<Array<{number: number}>>('GET', `/repos/${fullName}/pulls?state=open&head=${org}:${fixtureBranch}`);
    if (open.length) {
        return {number: open[0].number, title: pullTitle};
    }

    const {default_branch: base} = await ghOK('GET', `/repos/${fullName}`);
    if ((await gh('GET', `/repos/${fullName}/git/ref/heads/${fixtureBranch}`)).status === 404) {
        const {object} = await ghOK('GET', `/repos/${fullName}/git/ref/heads/${base}`);
        await ghOK('POST', `/repos/${fullName}/git/refs`, {body: {ref: `refs/heads/${fixtureBranch}`, sha: object.sha}});
    }
    if ((await gh('GET', `/repos/${fullName}/contents/e2e-fixture.md?ref=${fixtureBranch}`)).status === 404) {
        await ghOK('PUT', `/repos/${fullName}/contents/e2e-fixture.md`, {body: {message: 'Add E2E fixture file', content: Buffer.from('Kept open for the E2E suite.\n').toString('base64'), branch: fixtureBranch}});
    }
    const created = await ghOK('POST', `/repos/${fullName}/pulls`, {body: {title: pullTitle, head: fixtureBranch, base, body: 'Kept open for the E2E suite.'}});
    await ghOK('POST', `/repos/${fullName}/issues/${created.number}/labels`, {body: {labels: [fixtureLabel]}});
    if (created.user.login !== login) {
        throw new Error('The fixture pull request must be authored by the machine account');
    }
    return {number: created.number, title: pullTitle};
}

async function ensureHook() {
    const hooks = await ghOK<Array<{id: number; config: {url?: string}}>>('GET', `/repos/${realGitHub.fullName}/hooks?per_page=100`);
    const found = hooks.find((h) => h.config.url === fixtureHookURL);
    if (found) {
        return found.id;
    }
    const created = await ghOK('POST', `/repos/${realGitHub.fullName}/hooks`, {
        body: {name: 'web', active: true, events: ['issues', 'issue_comment', 'pull_request'], config: {url: fixtureHookURL, content_type: 'json'}},
    });
    return created.id as number;
}
