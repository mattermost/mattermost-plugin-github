// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

export default class TickIcon extends React.PureComponent {
    render() {
        return (
            <svg
                viewBox='0 0 12 16'
                version='1.1'
                width='12'
                height='16'
                role='img'
            >
                <path
                    fillRule='evenodd'
                    d='M12 5l-8 8-4-4 1.5-1.5L4 10l6.5-6.5L12 5z'
                />
            </svg>
        );
    }
}
