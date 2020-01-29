// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

export default class ChangesRequestedIcon extends React.PureComponent {
    render() {
        return (
            <svg 
                viewBox='0 0 16 15'
                version='1.1'
                width='16'
                height='15'
                role='img'
            >
                <path
                    fill-rule='evenodd'
                    d='M0 1a1 1 0 011-1h14a1 1 0 011 1v10a1 1 0 01-1 1H7.5L4 15.5V12H1a1 1 0 01-1-1V1zm1 0v10h4v2l2-2h8V1H1zm7.5 3h2v1h-2v2h-1V5h-2V4h2V2h1v2zm2 5h-5V8h5v1z'
                />
            </svg>
        );
    }
}
