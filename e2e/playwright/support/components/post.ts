// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {ChannelsPost} from '@e2e-support/ui/components';

// TODO: migrate this helpers to webapp's ChannelsPost after monorepo migration
export const getBotTagFromPost = (post: ChannelsPost) => {
    return post.container.locator('.post__header').locator('.BotTag', {hasText: 'BOT'});
};

export const getPostAuthor = (post: ChannelsPost) => {
    return post.container.locator('.post__header').getByRole('button');
};
