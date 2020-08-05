// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import ReactSelect from 'react-select';
import PropTypes from 'prop-types';

import {getStyleForReactSelect} from 'utils/styles';
import Setting from 'components/setting';

export default class IssueAttributeSelector extends PureComponent {
    static propTypes = {
        repo: PropTypes.string.isRequired,
        theme: PropTypes.object.isRequired,
        selectedValues: PropTypes.array.isRequired,
        onChange: PropTypes.func.isRequired,
        loadOptions: PropTypes.func.isRequired,
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

            // filter out currently selected options that do not exist in the new repo
            const optionValues = options.map((option) => option.value);
            const validValues = this.props.selectedValues.filter((v) => optionValues.includes(v.value));

            this.props.onChange(validValues);
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

    onChange = (selectedValues) => this.props.onChange(selectedValues || []);

    render() {
        return (
            <Setting {...this.props}>
                <ReactSelect
                    {...this.props}
                    isMulti={true}
                    closeMenuOnSelect={false}
                    hideSelectedOptions={true}
                    onChange={this.onChange}
                    options={this.state.options}
                    value={this.props.selectedValues}
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
