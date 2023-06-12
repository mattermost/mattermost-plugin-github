// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Locator} from '@playwright/test';

export default class SlashCommandSuggestions {
    constructor(readonly container: Locator) {
        this.container = container;
    }

    getItems() {
        return this.container.getByRole('button');
    }

    getItemNth(n: number) {
        return this.container.getByRole('button').nth(n);
    }
    getItemTitleNth(n: number) {
        return this.getItemNth(n).locator('.slash-command__title');
    }
    getItemDescNth(n: number) {
        return this.getItemNth(n).locator('.slash-command__desc');
    }

    // The text must be exact and complete, otherwise won't match the item
    getItemByText(text: string) {
        return this.container.getByRole('button', {name: text});
    }
}
export {SlashCommandSuggestions};
