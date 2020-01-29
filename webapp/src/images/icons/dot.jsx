// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

export default class DotIcon extends React.PureComponent {
    render() {
        return (
            <svg
                viewBox='0 0 8 16'
                version='1.1'
                width='8'
                height='16'
                role='img'
            >
                <path
                    fillRule='evenodd'
                    d='M0 8c0-2.2 1.8-4 4-4s4 1.8 4 4-1.8 4-4 4-4-1.8-4-4z'
                />
            </svg>
        );
    }
}
