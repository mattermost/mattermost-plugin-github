// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {useMount} from '../../hooks/useMount';

import type {Props} from '.';

export const UserAttribute = (props: Props) => {
    useMount(() => {
        props.actions.getGitHubUser(props.id);
    });

    const username = props.username;
    if (!username) {
        return null;
    }

    let baseURL = 'https://github.com';
    if (props.enterpriseURL) {
        baseURL = props.enterpriseURL;
    }

    return (
        <div style={style.container}>
            <a
                href={baseURL + '/' + username}
                target='_blank'
                rel='noopener noreferrer'
            >
                <i className='fa fa-github'/>{' ' + username}
            </a>
        </div>
    );
};

const style = {
    container: {
        margin: '5px 0',
    },
};
