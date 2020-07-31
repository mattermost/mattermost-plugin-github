// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import CreatableSelect from 'react-select/creatable';
import PropTypes from 'prop-types';

import {getStyleForReactSelect} from 'utils/styles';

export default class GithubLabelSelector extends PureComponent {
    static propTypes = {
        theme: PropTypes.object.isRequired,
        onChange: PropTypes.func,
    };

    handleChange = (items) => {
        const labels = items.map((i) => i.value);
        this.props.onChange(labels);
    };

    render() {
        return (
            <div className={'form-group margin-bottom x3'}>
                <label className='control-label margin-bottom x2'>
                    {'Github Label'}
                </label>
                <CreatableSelect
                    isMulti={true}
                    name='colors'
                    className='basic-multi-select'
                    classNamePrefix='select'
                    styles={getStyleForReactSelect(this.props.theme)}
                    noOptionsMessage={() => 'Start typing...'}
                    formatCreateLabel={(value) => `Add "${value}"`}
                    placeholder=''
                    onChange={this.handleChange}
                />
            </div>
        );
    }
}
