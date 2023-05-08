// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {GithubRHSCategory} from './components/todo_message';

export const messages = {
    UNCONNECTED: 'You must connect your account to GitHub first. Either click on the GitHub logo in the bottom left of the screen or enter /github connect.',
    NOSETUP: "Before using this plugin, you'll need to configure it by running /github setup: must have a github oauth client id",
};

export const expectedData = {
    [GithubRHSCategory.UNREAD]: {
        count: '1',
        firstItem: {
            title: 'Update README.md',
            link: 'https://github.com/MM-Github-Testorg/testrepo/issues/3#issuecomment-1485227142',
            repo: 'MM-Github-Testorg/testrepo',
            descriptionRegex: [/You were requested to review a pull request./, /days ago/],
        }},
    [GithubRHSCategory.ASSIGNMENTS]: {
        count: '32',
        firstItem: {
            title: 'The title',
            id: '#34',
            link: 'https://github.com/MM-Github-Testorg/testrepo/issues/34',
            repo: 'MM-Github-Testorg/testrepo',
            descriptionRegex: [/Opened/, /days ago by MM-Github-Plugin/],
        }},
    [GithubRHSCategory.REVIEW_PR]: {
        count: '1',
        firstItem: {
            title: 'Update README.md',
            id: '#3',
            link: 'https://github.com/MM-Github-Testorg/testrepo/pull/3',
            repo: 'MM-Github-Testorg/testrepo',
            descriptionRegex: [/Opened/, /days ago by trilopin/],
        }},
    [GithubRHSCategory.OPEN_PR]: {
        count: '1',
        firstItem: {
            title: 'Update README.md',
            id: '#1',
            link: 'https://github.com/MM-Github-Testorg/testrepo/pull/1',
            repo: 'MM-Github-Testorg/testrepo',
            descriptionRegex: [/Opened/, /days ago by MM-Github-Plugin/],
        }},
};

export const username = 'MM-Github-Plugin';
