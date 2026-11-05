// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

/* eslint-disable no-process-env */

import fs from 'node:fs';
import path from 'node:path';

// docker compose reads the same file, so MM_PORT/MM_IMAGE stay in sync between the stack and the tests.
const envFile = path.join(__dirname, '../.env');
if (fs.existsSync(envFile)) {
    process.loadEnvFile(envFile);
}

export const pluginId = 'github';
export const pluginDistDir = path.join(__dirname, '../../../dist');

// Must match the server's SiteURL in docker-compose.yml.
process.env.PW_BASE_URL ??= `http://127.0.0.1:${process.env.MM_PORT ?? 8069}`;
export const mattermostURL = process.env.PW_BASE_URL;

// Baked into the plugin by `make e2e-dist`. fake-github resolves from the Mattermost container (extra_hosts)
// and from Chromium (--host-resolver-rules); the tests reach it on localhost.
export const fakeGitHubHost = 'fake-github';
export const fakeGitHubPort = 8093;
export const fakeGitHubURL = `http://${fakeGitHubHost}:${fakeGitHubPort}`;
export const fakeGitHubControlURL = `http://localhost:${fakeGitHubPort}`;

export const oauthClientId = 'Iv1.e2e0123456789abc';
export const oauthClientSecret = 'e2e-github-oauth-client-secret-0000000000';
export const webhookSecret = 'e2e-github-webhook-secret';

// AES-256 needs a 32-byte key.
export const encryptionKey = 'e2e-encryption-key-0123456789abc';

export const webhookURL = `${mattermostURL}/plugins/${pluginId}/webhook`;

export const oauthApp = {
    clientId: oauthClientId,
    clientSecret: oauthClientSecret,
    callbackURL: `${mattermostURL}/plugins/${pluginId}/oauth/complete`,
    name: 'Mattermost E2E',
};
