// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Locator, Page} from '@playwright/test';

export default class RHS {
    readonly container: Locator;
    readonly header_title: Locator;
    readonly title: Locator;
    readonly items: Locator;

    constructor(readonly page: Page) {
        this.page = page;
        this.container = page.locator('#rhsContainer');
        this.header_title = this.container.locator('.sidebar--right__title');
        this.title = this.container.getByTestId('github-rhs-title').locator('a');
        this.items = this.container.getByTestId('github-rhs-item');
    }

    async getItem(nth: number) {
        return new RHSItem(this.items.nth(nth));
    }
}

export class RHSItem {
    readonly header_title: Locator;
    readonly title: Locator;
    readonly id: Locator;
    readonly repo: Locator;
    readonly description: Locator;

    constructor(readonly container: Locator) {
        this.header_title = this.container.locator('.sidebar--right__title');
        this.title = this.container.getByTestId('github-item-title');
        this.id = this.container.getByTestId('github-item-id');
        this.repo = this.container.getByTestId('github-item-repo');
        this.description = this.container.getByTestId('github-item-description');
    }
}
