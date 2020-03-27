// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

export default class CrossIcon extends React.PureComponent {
    render() {
        return (
            <svg
                viewBox='0 0 12 16'
                version='1.1'
                width='12px'
                height='16px'
                role='img'
            >
                <path
                    fillRule='evenodd'
                    d='M7.48 8l3.75 3.75-1.48 1.48L6 9.48l-3.75 3.75-1.48-1.48L4.52 8 .77 4.25l1.48-1.48L6 6.52l3.75-3.75 1.48 1.48L7.48 8z'
                />
            </svg>
        );
    }
}
