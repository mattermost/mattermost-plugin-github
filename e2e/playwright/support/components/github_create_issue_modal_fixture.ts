// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Page} from '@playwright/test';

export default class CreateIssueForm {
    readonly formElement = this.page.locator('#github-create-issue-form');

    readonly issueTitle = this.page.locator('#github-issue-title input');
    readonly issueDescription = this.page.locator('#github-issue-description textarea');

    readonly header = this.page.getByRole('heading', {name: 'Create GitHub Issue'});

    constructor(private readonly page: Page) {}

    selectRepo = async (repoSearch: string, repoName: string) => {
        const selector = this.page.locator('#github-repo-selector');
        await selector.click();
        await selector.type(repoSearch);
        await this.clickReactSelectOption(repoName);
    }

    selectLabels = async (labels: string[]) => {
        const selector = this.page.locator('#github-label-selector');
        await selector.click();

        for (const label of labels) {
            await this.clickReactSelectOption(label);
        }
    }

    selectAssignees = async (assignees: string[]) => {
        const selector = this.page.locator('#github-assignee-selector');
        await selector.click();

        for (const assignee of assignees) {
            await this.clickReactSelectOption(assignee);
        }
    }

    selectMilestone = async (milestone: string) => {
        const selector = this.page.locator('#github-milestone-selector');
        await selector.click();
        await this.clickReactSelectOption(milestone);
    }

    submit = async () => {
        await this.page.getByRole('button', {name: 'Submit'}).click();
    }

    clickReactSelectOption = async (optionText: string) => {
        const selector = 'div[id^="react-select-"]';
        await this.page.waitForSelector(selector);

        const arrayOfLocators = this.page.locator(selector);
        const elementsCount = await arrayOfLocators.count();

        for (let index = 0; index < elementsCount; index++) {
            const element = arrayOfLocators.nth(index);
            const innerText = await element.innerText();
            if (innerText === optionText) {
                await element.click();
                await this.header.click();
                return;
            }
        }

        throw new Error(`Could not find React Select option with text "${optionText}"`)
    }
}
