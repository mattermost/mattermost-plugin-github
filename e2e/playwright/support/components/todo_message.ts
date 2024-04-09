// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, Locator} from '@playwright/test';

export enum GithubRHSCategory {
    UNREAD = 'unread',
    ASSIGNMENTS = 'assignments',
    REVIEW_PR = 'reviewpr',
    OPEN_PR = 'openpr',
}

export default class TodoMessage {
    readonly zeroResRegex: RegExp;
    readonly titles: Map<GithubRHSCategory, Locator>;
    readonly descriptions: Map<GithubRHSCategory, Locator>;
    readonly lists: Map<GithubRHSCategory, Locator>;

    constructor(readonly container: Locator) {
        this.container = container;
        this.zeroResRegex = /don't have any/;

        this.titles = new Map<GithubRHSCategory, Locator>();
        this.titles.set(GithubRHSCategory.UNREAD, container.locator('h5').filter({hasText: 'Unread Messages'}));
        this.titles.set(GithubRHSCategory.OPEN_PR, container.locator('h5').filter({hasText: 'Your Open Pull Requests'}));
        this.titles.set(GithubRHSCategory.REVIEW_PR, container.locator('h5').filter({hasText: 'Review Requests'}));
        this.titles.set(GithubRHSCategory.ASSIGNMENTS, container.locator('h5').filter({hasText: 'Your Assignments'}));

        this.descriptions = new Map<GithubRHSCategory, Locator>();
        this.descriptions.set(GithubRHSCategory.UNREAD, container.locator('p').filter({hasText: 'unread messages'}));
        this.descriptions.set(GithubRHSCategory.OPEN_PR, container.locator('p').filter({hasText: 'open pull requests:'}));
        this.descriptions.set(GithubRHSCategory.REVIEW_PR, container.locator('p').filter({hasText: 'pull requests awaiting your review:'}));
        this.descriptions.set(GithubRHSCategory.ASSIGNMENTS, container.locator('p').filter({hasText: 'assignments:'}));

        this.lists = new Map<GithubRHSCategory, Locator>();
        this.lists.set(GithubRHSCategory.UNREAD, container.locator('ul:below(h5:text("Unread Messages"))').first());
        this.lists.set(GithubRHSCategory.OPEN_PR, container.locator('ul:below(h5:text("Your Open Pull Requests"))').first());
        this.lists.set(GithubRHSCategory.REVIEW_PR, container.locator('ul:below(h5:text("Review Requests"))').first());
        this.lists.set(GithubRHSCategory.ASSIGNMENTS, container.locator('ul:below(h5:text("Your Assignments"))').first());
    }

    getTitle(kind: GithubRHSCategory): Locator {
        return this.titles.get(kind) ?? this.container.locator('notfound');
    }

    getDesc(kind: GithubRHSCategory): Locator {
        return this.descriptions.get(kind) ?? this.container.locator('notfound');
    }

    // this func match elements based on layout, not the most reliable selector :(
    async getList(kind: GithubRHSCategory) {
        // if desc says there's no items, don't check the list (or will return the next one)
        const desc = await this.getDesc(kind)?.innerText() ?? '';
        if (desc.match(this.zeroResRegex)) {
            return this.container.locator('notfound'); // temp trick
        }
        return this.lists.get(kind) ?? this.container.locator('notfound');
    }

    async toBeVisible() {
        await expect(this.container).toBeVisible();
    }
}
