// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

/* eslint-disable no-process-env */

import path from 'node:path';

import '../support/constants';

const required = [
    'E2E_GITHUB_OAUTH_CLIENT_ID',
    'E2E_GITHUB_OAUTH_CLIENT_SECRET',
    'E2E_GITHUB_REPO',
    'E2E_GITHUB_MACHINE_TOKEN',
] as const;

export function missingRealGitHubEnv() {
    return required.filter((name) => !process.env[name]);
}

const env = (name: (typeof required)[number]) => process.env[name] ?? '';

const [org, repo] = env('E2E_GITHUB_REPO').split('/');

export const realGitHub = {
    apiURL: 'https://api.github.com',
    webURL: 'https://github.com',
    org,
    repo,
    fullName: `${org}/${repo}`,
    oauth: {clientId: env('E2E_GITHUB_OAUTH_CLIENT_ID'), clientSecret: env('E2E_GITHUB_OAUTH_CLIENT_SECRET')},
    machineToken: env('E2E_GITHUB_MACHINE_TOKEN'),

    // Optional: the real-login test skips without them.
    login: {
        username: process.env.E2E_GITHUB_LOGIN_USERNAME || undefined,
        password: process.env.E2E_GITHUB_LOGIN_PASSWORD || undefined,
        totpSecret: process.env.E2E_GITHUB_LOGIN_TOTP_SECRET || undefined,
    },

    // Shared between global setup and the tests, which run in separate processes.
    statePath: path.join(__dirname, '../.auth/real-github-state.json'),
    writeLogPath: path.join(__dirname, '../.auth/real-github-writes.json'),
};

export const canSignInWithPassword = () => Boolean(realGitHub.login.username && realGitHub.login.password && realGitHub.login.totpSecret);
