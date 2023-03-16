// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, Locator, Page} from '@playwright/test';


export default class GithubPage {

    readonly page: Page;
    readonly url: string;
    readonly username: Locator;
    readonly password: Locator;
    readonly submit: Locator;

    constructor(page: Page) {
        this.page = page;

        this.url = "https://github.com/login";
        this.username = page.locator('#login_field');
        this.password = page.locator('#password');
        this.submit = page.locator('input[type="submit"]');

    }

    async toBeVisible() {
        await this.page.waitForLoadState('networkidle');
        await expect(this.username).toBeVisible();
        await expect(this.password).toBeVisible();
        await expect(this.submit).toBeVisible();
    }

    async login(username: string, password: string) {
        await this.username.fill(username);
        await this.password.fill(password);
        await this.submit.click();
    }
}

export {GithubPage};
