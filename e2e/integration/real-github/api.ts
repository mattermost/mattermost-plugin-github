// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect} from '@playwright/test';

import {realGitHub} from './env';
import {reserveWrite} from './rate_limit';

export type GitHubResponse<T> = {status: number; body: T; headers: Headers};

export async function gh<T = any>(method: string, path: string, {body, token = realGitHub.machineToken, auth}: {body?: unknown; token?: string; auth?: string} = {}): Promise<GitHubResponse<T>> {
    if (method !== 'GET') {
        await reserveWrite();
    }
    for (let attempt = 0; ; attempt++) {
        const res = await fetch(`${realGitHub.apiURL}${path}`, {
            method,
            headers: {
                Accept: 'application/vnd.github+json',
                Authorization: auth ?? `Bearer ${token}`,
                'X-GitHub-Api-Version': '2022-11-28',
                'User-Agent': 'mattermost-plugin-github-e2e',
                ...(body === undefined ? {} : {'Content-Type': 'application/json'}),
            },
            body: body === undefined ? undefined : JSON.stringify(body),
        });

        // Secondary limits answer 403 or 429 with retry-after; primary ones send x-ratelimit-reset.
        const retryAfter = Number(res.headers.get('retry-after') ?? 0);
        if ((res.status === 429 || (res.status === 403 && (retryAfter || res.headers.get('x-ratelimit-remaining') === '0'))) && attempt < 2) {
            const resetMs = Number(res.headers.get('x-ratelimit-reset') ?? 0) * 1000 - Date.now();
            const waitMs = retryAfter ? retryAfter * 1000 : resetMs;
            if (waitMs > 0 && waitMs <= 2 * 60_000) {
                await new Promise((r) => setTimeout(r, waitMs));
                continue;
            }
        }
        const text = await res.text();
        return {status: res.status, body: (text ? JSON.parse(text) : undefined) as T, headers: res.headers};
    }
}

export async function ghOK<T = any>(method: string, path: string, options?: Parameters<typeof gh>[2]): Promise<T> {
    const res = await gh<T>(method, path, options);
    if (res.status >= 300) {
        throw new Error(`${method} ${path} answered ${res.status}: ${JSON.stringify(res.body).slice(0, 300)}`);
    }
    return res.body;
}

// The search index trails writes by seconds to minutes.
export async function waitForSearch(query: string, matches: (item: {number: number; title: string}) => boolean, timeoutMs = 3 * 60_000) {
    await expect.poll(async () => {
        const res = await gh<{items: Array<{number: number; title: string}>}>('GET', `/search/issues?q=${encodeURIComponent(query)}&per_page=50`);
        return res.status === 200 && res.body.items.some(matches);
    }, {message: `GitHub search never returned a match for "${query}"`, timeout: timeoutMs, intervals: [5_000, 10_000, 15_000]}).toBe(true);
}
