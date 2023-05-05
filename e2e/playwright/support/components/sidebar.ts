// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Locator, Page} from '@playwright/test';

import {GithubRHSCategory} from './todo_message';

export default class Sidebar {
    readonly counters: Map<GithubRHSCategory, Locator>;
    readonly container: Locator;
    readonly containerUnconnected: Locator;
    readonly refresh: Locator;

    constructor(readonly page: Page) {
        this.page = page;
        this.container = page.getByTestId('sidebar-github');
        this.containerUnconnected = page.getByTestId('sidebar-github-unconnected');
        this.refresh = this.container.getByTestId('sidebar-github-refresh');

        this.counters = new Map<GithubRHSCategory, Locator>();
        this.counters.set(GithubRHSCategory.UNREAD, this.container.getByTestId('sidebar-github-unreads'));
        this.counters.set(GithubRHSCategory.OPEN_PR, this.container.getByTestId('sidebar-github-openpr'));
        this.counters.set(GithubRHSCategory.REVIEW_PR, this.container.getByTestId('sidebar-github-reviewpr'));
        this.counters.set(GithubRHSCategory.ASSIGNMENTS, this.container.getByTestId('sidebar-github-assignments'));
    }

    getCounter(kind: GithubRHSCategory): Locator {
        return this.counters.get(kind) ?? this.container.locator('notfound');
    }
}
