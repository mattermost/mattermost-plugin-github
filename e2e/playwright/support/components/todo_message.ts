// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, Locator} from '@playwright/test';

export default class TodoMessage {
        readonly unreadTitle: Locator;
        readonly openPRTitle: Locator;
        readonly reviewPRTitle: Locator;
        readonly assignmentsTitle: Locator;
        readonly unreadDesc: Locator;
        readonly openPRDesc: Locator;
        readonly reviewPRDesc: Locator;
        readonly assignmentsDesc: Locator;
        readonly zeroResRegex: RegExp;


        constructor(readonly container: Locator) {
            this.container = container;
            this.unreadTitle = container.locator('h5').filter({hasText: 'Unread Messages'});
            this.openPRTitle = container.locator('h5').filter({hasText: 'Your Open Pull Requests'});
            this.reviewPRTitle = container.locator('h5').filter({hasText: 'Review Requests'});
            this.assignmentsTitle = container.locator('h5').filter({hasText: 'Your Assignments'});
            this.unreadDesc = container.locator('p').filter({hasText: 'unread messages'});
            this.openPRDesc = container.locator('p').filter({hasText: 'open pull requests:'});
            this.reviewPRDesc = container.locator('p').filter({hasText: 'pull requests awaiting your review:'});
            this.assignmentsDesc = container.locator('p').filter({hasText: 'assignments:'});
            this.zeroResRegex = /don\'t have any/;
        }

        getTitle(kind: 'unread' | 'assignments' | 'reviewpr' | 'openpr') {
            switch (kind) {
                case 'unread': return this.unreadTitle;
                case 'assignments': return this.assignmentsTitle;
                case 'reviewpr': return this.reviewPRTitle;
                case 'openpr': return this.openPRTitle;
            }
        }

        getDesc(kind: 'unread' | 'assignments' | 'reviewpr' | 'openpr') {
            switch (kind) {
                case 'unread': return this.unreadDesc;
                case 'assignments': return this.assignmentsDesc;
                case 'reviewpr': return this.reviewPRDesc;
                case 'openpr': return this.openPRDesc;
            }
        }

        // this func match elements based on layout, not the most reliable selector :(
        async getList(kind: 'unread' | 'assignments' | 'reviewpr' | 'openpr') {

            // if desc says there's no items, don't check the list (or will return the next one)
            const desc = await this.getDesc(kind).innerText();
            if (desc.match(this.zeroResRegex)) {
                return this.container.locator('notfound'); // temp trick
            }

            switch (kind) {
                case 'unread': return this.container.locator('ul:below(h5:text("Unread Messages"))').first()
                case 'assignments': return this.container.locator('ul:below(h5:text("Your Assignments"))').first()
                case 'reviewpr': return this.container.locator('ul:below(h5:text("Review Requests"))').first()
                case 'openpr': return this.container.locator('ul:below(h5:text("Your Open Pull Requests"))').first()
            }
        }

        async toBeVisible() {
            await expect(this.container).toBeVisible();
        }
    }
export {TodoMessage};
