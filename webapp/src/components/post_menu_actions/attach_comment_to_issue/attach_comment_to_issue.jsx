// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import GitHubIcon from '../../icon';

export default function AttachCommentToIssuePostMenuAction() {
    return (
        <>
            <GitHubIcon type='menu'/>
            {'Attach to GitHub Issue'}
        </>
    );
}
