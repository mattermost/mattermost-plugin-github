// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {Locator, Page} from '@playwright/test';

import {realGitHub} from '../env';

import {generateTotp} from './totp';

// Walks github.com sign-in, 2FA and OAuth consent until isDone. GitHub skips the emailed device check for 2FA accounts.
export async function completeGitHubSignIn(page: Page, isDone: () => boolean, timeoutMs = 2 * 60_000) {
    const {username, password, totpSecret} = realGitHub.login;
    let lastCode = '';

    const steps: Array<[Locator, (el: Locator) => Promise<void>]> = [
        [page.locator('input#login_field'), async (el) => {
            await el.fill(username!);
            await page.locator('input#password').fill(password!);
            await page.locator('input[name="commit"]').click();
        }],
        [page.locator('a[href="/sessions/two-factor/app"]'), (el) => el.click()],
        [page.locator('input#app_totp'), async (el) => {
            let code = generateTotp(totpSecret!);
            while (code === lastCode) {
                await page.waitForTimeout(2000);
                code = generateTotp(totpSecret!);
            }
            lastCode = code;

            // The field submits itself once six digits are in.
            await el.fill(code);
        }],
        [page.getByRole('button', {name: /^(remind me later|not now|skip)/i}).first(), (el) => el.click()],
        [page.locator('button#js-oauth-authorize-btn:not([disabled])'), (el) => el.click()],
    ];

    const deadline = Date.now() + timeoutMs;
    while (Date.now() < deadline) {
        if (isDone()) {
            return;
        }
        if (new URL(page.url()).pathname.startsWith('/sessions/verified-device')) {
            throw new Error('GitHub asked for an emailed device code; the login account needs 2FA with an authenticator app');
        }
        for (const [el, act] of steps) {
            if (await el.isVisible().catch(() => false)) {
                await act(el);
                await page.waitForLoadState('domcontentloaded').catch(() => undefined);
                break;
            }
        }
        await page.waitForTimeout(1000);
    }
    throw new Error(`GitHub sign-in did not finish; last page was ${new URL(page.url()).pathname}`);
}
