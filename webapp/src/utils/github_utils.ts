// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export function isUrlCanPreview(url: string) {
    const {hostname, pathname} = new URL(url);
    if (hostname.includes('github.com') && pathname.split('/')[1]) {
        const [_, owner, repo, type, number] = pathname.split('/');
        return Boolean(owner && repo && type && number);
    }
    return false;
}
