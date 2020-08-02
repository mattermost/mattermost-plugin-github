// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import ReactSelect from 'react-select';
import PropTypes from 'prop-types';

import {getStyleForReactSelect} from 'utils/styles';

export default class GithubLabelSelector extends PureComponent {
    static propTypes = {
        repo: PropTypes.string,
        theme: PropTypes.object.isRequired,
        onChange: PropTypes.func,
        labels: PropTypes.array.isRequired,
        actions: PropTypes.shape({
            getLabels: PropTypes.func.isRequired,
        }).isRequired,
    };

    componentDidMount() {
        this.fetchLabels();
    }

    componentDidUpdate(prevProps) {
        if (prevProps.repo === this.props.repo) {
            return;
        }
        this.fetchLabels();
    }

    fetchLabels = (query) => {
        // no point in searching without a repo ID or a query string
        if (!this.props.repo || !query) {
            return;
        }

        this.props.actions.getLabels(this.props.repo, query);
    };

    // in order to avoid duplicate labels, convert array to set and then back to array
    handleChange = (items) => {
        if (!items || items.length === 0) {
            return;
        }

        const labels = new Set(items.map((i) => i.value));
        this.props.onChange([...labels]);
    };

    render() {
        const options = this.props.labels.map((item) => ({
            value: item.name,
            label: item.name,
        }));

        return (
            <div className='form-group margin-bottom x3'>
                <label className='control-label margin-bottom x2'>
                    {'Labels'}
                </label>
                <ReactSelect
                    isMulti={true}
                    name='colors'
                    className='basic-multi-select'
                    classNamePrefix='select'
                    styles={getStyleForReactSelect(this.props.theme)}
                    onInputChange={(query) => this.fetchLabels(query)}
                    onChange={this.handleChange}
                    options={options}
                    noOptionsMessage={() => 'Start typing...'}
                />
            </div>
        );
    }
}
