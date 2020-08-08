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
        repo: PropTypes.string.isRequired,
        theme: PropTypes.object.isRequired,
        onChange: PropTypes.func.isRequired,
        loadOptions: PropTypes.func.isRequired,
        selection: PropTypes.oneOfType([
            PropTypes.array,
            PropTypes.string,
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

    componentDidUpdate(prevProps) {
        if (this.props.repo && prevProps.repo !== this.props.repo) {
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
            });
        } catch (err) {
            this.setState({
                error: err,
                isLoading: false,
            });
        }
    };

    filterSelection = (options) => {
        const optionValues = options.map((option) => option.value);

        // filter out currently selected options that do not exist in the new repo
        let validSelection;
        if (this.props.isMulti) {
            validSelection = this.props.selection.filter((v) => optionValues.includes(v));
        } else {
            validSelection = optionValues.includes(this.props.selection) ? this.props.selection : {};
        }

        this.props.onChange(validSelection);
    }

    onChange = (selection) => {
        if (this.props.isMulti) {
            this.props.onChange(selection ? selection.map((s) => s.value) : []);
            return;
        }

        if (!selection || !selection.value) {
            this.props.onChange('');
            return;
        }

        this.props.onChange(selection.value);
    };

    render() {
        let selection;
        if (this.props.isMulti) {
            selection = this.props.selection.map((s) => ({label: s, value: s}));
        } else {
            selection = this.props.selection ? {
                label: this.props.selection,
                value: this.props.selection,
            } : '';
        }

        return (
            <Setting {...this.props}>
                <ReactSelect
                    {...this.props}
                    isClearable={true}
                    placeholder={'Select...'}
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
