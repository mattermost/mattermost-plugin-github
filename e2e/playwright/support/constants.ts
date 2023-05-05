// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {GithubRHSCategory} from './components/todo_message';

export const messages = {
    UNCONNECTED: 'You must connect your account to GitHub first. Either click on the GitHub logo in the bottom left of the screen or enter /github connect.',
    NOSETUP: "Before using this plugin, you'll need to configure it by running /github setup: must have a github oauth client id",
};

export const expectedData = {
    [GithubRHSCategory.UNREAD]: '1',
    [GithubRHSCategory.ASSIGNMENTS]: '32',
    [GithubRHSCategory.REVIEW_PR]: '1',
    [GithubRHSCategory.OPEN_PR]: '1',
};

export const username = 'MM-Github-Plugin';
