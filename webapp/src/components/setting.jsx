// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import PropTypes from 'prop-types';

export default class Setting extends React.PureComponent {
    static propTypes = {
        inputId: PropTypes.string,
        label: PropTypes.node,
        children: PropTypes.node.isRequired,
        helpText: PropTypes.node,
        required: PropTypes.bool,
        hideRequiredStar: PropTypes.bool,
    };

    render() {
        const {
            children,
            helpText,
            inputId,
            label,
            required,
            hideRequiredStar,
        } = this.props;

        return (
            <div
                className='form-group less'
                style={{marginBottom: '8px'}}
            >
                {label && (
                    <label
                        className='control-label margin-bottom x2'
                        htmlFor={inputId}
                    >
                        {label}
                    </label>)
                }
                {required && !hideRequiredStar && (
                    <span
                        className='error-text'
                        style={{marginLeft: '3px'}}
                    >
                        {'*'}
                    </span>
                )
                }
                <div>
                    {children}
                    <div
                        className='help-text'
                        style={{margin: '0px'}}
                    >
                        {helpText}
                    </div>
                </div>
            </div>
        );
    }
}
