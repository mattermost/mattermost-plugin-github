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

    getItem(nth: number): Locator {
        return this.items.nth(nth);
    }
}
