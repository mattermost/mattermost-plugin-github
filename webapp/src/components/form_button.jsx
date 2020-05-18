// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

export default class FormButton extends PureComponent {
    static propTypes = {
        executing: PropTypes.bool,
        disabled: PropTypes.bool,
        executingMessage: PropTypes.node,
        defaultMessage: PropTypes.node,
        btnClass: PropTypes.string,
        extraClasses: PropTypes.string,
        saving: PropTypes.bool,
        savingMessage: PropTypes.string,
        type: PropTypes.string,
    };

    static defaultProps = {
        disabled: false,
        savingMessage: 'Creating',
        defaultMessage: 'Create',
        btnClass: 'btn-primary',
        extraClasses: '',
    };

    render() {
        const {saving, disabled, savingMessage, defaultMessage, btnClass, extraClasses, ...props} = this.props;

        let contents;
        if (saving) {
            contents = (
                <span>
                    <span
                        className='fa fa-spin fa-spinner'
                        title={'Loading Icon'}
                    />
                    {savingMessage}
                </span>
            );
        } else {
            contents = defaultMessage;
        }

        let className = 'save-button btn ' + btnClass;

        if (extraClasses) {
            className += ' ' + extraClasses;
        }

        return (
            <button
                id='saveSetting'
                className={className}
                disabled={disabled}
                {...props}
            >
                {contents}
            </button>
        );
    }
}
