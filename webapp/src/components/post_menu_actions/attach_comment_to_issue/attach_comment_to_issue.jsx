// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

import {id as pluginId} from '../../../manifest';

export default class AttachCommentToIssuePostMenuAction extends PureComponent {
    static propTypes = {
        isSystemMessage: PropTypes.bool.isRequired,
        open: PropTypes.func.isRequired,
        postId: PropTypes.string,
        connected: PropTypes.bool.isRequired,
    };

    static defaultTypes = {
        locale: 'en',
    };

    handleClick = (e) => {
        const {open, postId} = this.props;
        e.preventDefault();
        open(postId);
    };

    connectClick = () => {
        window.open('/plugins/' + pluginId + '/user/connect', '_blank');
    };

    render() {
        if (this.props.isSystemMessage || !this.props.connected) {
            return null;
        }

        const content = (
            <button
                className='style--none'
                role='presentation'
                onClick={this.handleClick}
            >
                <span className="MenuItem__icon"><i className='fa fa-github fa-lg'/></span>
                {'Attach to Github Issue'}
            </button>
        );

        return (
            <li
                className='MenuItem'
                role='menuitem'
            >
                {content}
            </li>
        );
    }
}
