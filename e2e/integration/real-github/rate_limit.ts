// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import fs from 'node:fs';
import path from 'node:path';

import {realGitHub} from './env';

// GitHub's secondary limits for content-creating requests. Kept on disk so back-to-back local runs share one budget.
const limits = [
    {windowMs: 60_000, max: 80},
    {windowMs: 60 * 60_000, max: 500},
];

function readLog(): number[] {
    try {
        return JSON.parse(fs.readFileSync(realGitHub.writeLogPath, 'utf8'));
    } catch {
        return [];
    }
}

// Waits until one more write fits in every window, then records it. Call before each write, including ones the plugin makes.
export async function reserveWrite() {
    for (;;) {
        const now = Date.now();
        const log = readLog().filter((t) => now - t < limits.at(-1)!.windowMs);
        const waitMs = Math.max(0, ...limits.map(({windowMs, max}) => {
            const inWindow = log.filter((t) => now - t < windowMs);
            return inWindow.length < max ? 0 : inWindow[inWindow.length - max] + windowMs - now;
        }));
        if (waitMs === 0) {
            fs.mkdirSync(path.dirname(realGitHub.writeLogPath), {recursive: true});
            fs.writeFileSync(realGitHub.writeLogPath, JSON.stringify([...log, now]));
            return;
        }
        if (waitMs > 5 * 60_000) {
            throw new Error(`GitHub write budget exhausted for ${Math.ceil(waitMs / 60_000)} more minutes`);
        }
        await new Promise((r) => setTimeout(r, waitMs));
    }
}
