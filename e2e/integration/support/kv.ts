// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

/* eslint-disable no-process-env */

import {execFileSync} from 'node:child_process';
import {createCipheriv, createDecipheriv, randomBytes} from 'node:crypto';
import path from 'node:path';

import {encryptionKey, pluginId} from './constants';

export type StoredUserInfo = {
    UserID: string;
    Token: {access_token: string; token_type: string; expiry: string};
    GitHubUsername: string;
    LastToDoPostAt: number;
    Settings: {sidebar_buttons: string; daily_reminder: boolean; daily_reminder_on_change: boolean; notifications: boolean};
    AllowedPrivateRepos: boolean;
    MM34646ResetTokenDone: boolean;
};

const composeDir = path.join(__dirname, '..');

// Keys and format must match storeGitHubUserInfo and storeGitHubToUserIDMapping (server/plugin/plugin.go).
export function seedUserInfo(mattermostUserId: string, githubLogin: string, accessToken: string, options: {allowedPrivateRepos?: boolean; settings?: Partial<StoredUserInfo['Settings']>; lastToDoPostAt?: number; key?: string} = {}) {
    const info: StoredUserInfo = {
        UserID: mattermostUserId,

        // Classic OAuth app tokens never expire; oauth2 stores the zero time.
        Token: {access_token: encrypt(accessToken, options.key), token_type: 'bearer', expiry: '0001-01-01T00:00:00Z'},
        GitHubUsername: githubLogin,

        // A recent value keeps the daily reminder from posting into the DM channel tests assert on.
        LastToDoPostAt: options.lastToDoPostAt ?? Date.now(),
        Settings: {sidebar_buttons: 'team', daily_reminder: true, daily_reminder_on_change: false, notifications: true, ...options.settings},
        AllowedPrivateRepos: options.allowedPrivateRepos ?? false,
        MM34646ResetTokenDone: true,
    };
    upsert({
        [`${mattermostUserId}_githubtoken`]: JSON.stringify(info),
        [`${githubLogin}_githubusername`]: mattermostUserId,
    });
}

export function readUserInfo(mattermostUserId: string): StoredUserInfo | undefined {
    const raw = get(`${mattermostUserId}_githubtoken`);
    if (!raw) {
        return undefined;
    }
    const info = JSON.parse(raw) as StoredUserInfo;
    return {...info, Token: {...info.Token, access_token: decrypt(info.Token.access_token)}};
}

export function seedTokenResetDone() {
    upsert({mm34646_token_reset_done: 'done'});
}

export function hasKey(key: string) {
    return Boolean(get(key));
}

export function deleteKey(key: string) {
    psql(`DELETE FROM pluginkeyvaluestore WHERE pluginid = '${pluginId}' AND pkey = ${quote(key)};`);
}

function get(key: string) {
    return psql(`SELECT convert_from(pvalue, 'UTF8') FROM pluginkeyvaluestore WHERE pluginid = '${pluginId}' AND pkey = ${quote(key)};`, ['-A', '-t']).trim();
}

function upsert(entries: Record<string, string>) {
    const values = Object.entries(entries).map(([key, value]) => `('${pluginId}', ${quote(key)}, convert_to(${quote(value)}, 'UTF8'), 0)`);
    psql(`INSERT INTO pluginkeyvaluestore (pluginid, pkey, pvalue, expireat) VALUES ${values.join(', ')}
        ON CONFLICT (pluginid, pkey) DO UPDATE SET pvalue = EXCLUDED.pvalue, expireat = 0;`);
}

function psql(sql: string, flags: string[] = []) {
    return execFileSync('docker', ['compose', 'exec', '-T', 'postgres', 'psql', '-U', 'mmuser', '-d', 'mattermost_test', '-v', 'ON_ERROR_STOP=1', '-q', ...flags], {
        cwd: composeDir,
        input: sql,
        encoding: 'utf8',
        stdio: ['pipe', 'pipe', 'inherit'],
        env: process.env,
    });
}

function quote(value: string) {
    return `'${value.replaceAll("'", "''")}'`;
}

// Mirrors encrypt/decrypt in server/plugin/utils.go: PKCS#7 padding, AES-CFB with a random IV prefix, padded URL-safe base64.
export function encrypt(text: string, key = encryptionKey) {
    const iv = randomBytes(16);
    const plain = Buffer.from(text);
    const padding = 16 - (plain.length % 16);
    const cipher = createCipheriv('aes-256-cfb', Buffer.from(key), iv);
    const body = Buffer.concat([cipher.update(Buffer.concat([plain, Buffer.alloc(padding, padding)])), cipher.final()]);
    return Buffer.concat([iv, body]).toString('base64').replaceAll('+', '-').replaceAll('/', '_');
}

export function decrypt(text: string, key = encryptionKey) {
    const data = Buffer.from(text.replaceAll('-', '+').replaceAll('_', '/'), 'base64');
    const decipher = createDecipheriv('aes-256-cfb', Buffer.from(key), data.subarray(0, 16));
    const plain = Buffer.concat([decipher.update(data.subarray(16)), decipher.final()]);
    return plain.subarray(0, plain.length - plain[plain.length - 1]).toString();
}
