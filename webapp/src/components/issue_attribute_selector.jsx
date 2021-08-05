// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import ReactSelect from 'react-select';
import PropTypes from 'prop-types';

import {getStyleForReactSelect} from 'utils/styles';
import Setting from 'components/setting';

export default class IssueAttributeSelector extends PureComponent {
    static propTypes = {
        isMulti: PropTypes.bool.isRequired,
        repoName: PropTypes.string.isRequired,
        theme: PropTypes.object.isRequired,
        onChange: PropTypes.func.isRequired,
        loadOptions: PropTypes.func.isRequired,
        selection: PropTypes.oneOfType([
            PropTypes.array,
            PropTypes.object,
        ]).isRequired,
    };

    constructor(props) {
        super(props);

        this.state = {
            options: [],
            isLoading: false,
            error: null,
        };
    }

    componentDidMount() {
        if (this.props.repoName) {
            this.loadOptions();
        }
    }

    componentDidUpdate(prevProps) {
        if (this.props.repoName && prevProps.repoName !== this.props.repoName) {
            this.loadOptions();
        }
    }

    loadOptions = async () => {
        this.setState({isLoading: true});

        try {
            const options = await this.props.loadOptions();
            this.filterSelection(options);
            this.setState({
                options,
                isLoading: false,
                error: null,
            });
        } catch (err) {
            this.filterSelection([]);
            this.setState({
                error: err,
                isLoading: false,
            });
        }
    };

    filterSelection = (options) => {
        if (this.props.isMulti) {
            const filtered = options.filter((option) => this.props.selection.includes(option.value));
            this.props.onChange(filtered);
            return;
        }

        if (!this.props.selection) {
            this.props.onChange(null);
            return;
        }

        for (const option of options) {
            if (option.value === this.props.selection.value) {
                this.props.onChange(option);
                return;
            }
        }

        this.props.onChange(null);
    }

    onChange = (selection) => {
        if (this.props.isMulti) {
            this.props.onChange(selection || []);
            return;
        }

        this.props.onChange(selection);
    };

    render() {
        let selection;
        if (this.props.isMulti) {
            selection = this.props.selection.map((s) => ({label: s, value: s}));
        } else {
            selection = this.props.selection || {};
        }

        const noOptionsMessage = this.props.repoName ? 'No options' : 'Please select a repository first';

        return (
            <Setting {...this.props}>
                <ReactSelect
                    {...this.props}
                    isClearable={true}
                    placeholder={'Select...'}
                    isDisabled={this.props.repoName === ''}
                    noOptionsMessage={() => noOptionsMessage}
                    closeMenuOnSelect={!this.props.isMulti}
                    hideSelectedOptions={this.props.isMulti}
                    onChange={this.onChange}
                    options={this.state.options}
                    value={selection}
                    isLoading={this.state.isLoading}
                    styles={getStyleForReactSelect(this.props.theme)}
                />

                {this.state.error && (
                    <p className='alert alert-danger'>
                        <i
                            className='fa fa-warning'
                            title='Warning Icon'
                        />
                        <span> {this.state.error.message}</span>
                    </p>
                )}
            </Setting>
        );
    }
}
