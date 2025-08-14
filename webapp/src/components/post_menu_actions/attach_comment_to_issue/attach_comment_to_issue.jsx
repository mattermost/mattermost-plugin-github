// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import GitHubIcon from '../../icon';

export default function AttachCommentToIssuePostMenuAction() {
    return (
        <li
            className='MenuItem'
            role='menuitem'
        >
            <button className='style--none'>
                <span className='MenuItem__icon'>
                    <GitHubIcon type='menu'/>
                </span>
                {'Attach to GitHub Issue'}
            </button>
        </li>
    );
}

