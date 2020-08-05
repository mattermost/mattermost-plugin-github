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
        onChange: PropTypes.func.isRequired,
        loadOptions: PropTypes.func.isRequired,
        theme: PropTypes.object.isRequired,
        value: PropTypes.oneOfType([
            PropTypes.object,
            PropTypes.array,
            PropTypes.string,
        ]),
        required: PropTypes.bool,
        isMulti: PropTypes.bool,
        addValidate: PropTypes.func,
        removeValidate: PropTypes.func,
        resetInvalidOnChange: PropTypes.bool,
    };

    constructor(props) {
        super(props);

        this.state = {
            options: [],
            values: [],
            isLoading: false,
            invalid: false,
            error: null,
        };
    }

    componentDidMount() {
        if (this.props.addValidate) {
            this.props.addValidate(this.isValid);
        }
    }

    componentWillUnmount() {
        if (this.props.removeValidate) {
            this.props.removeValidate(this.isValid);
        }
    }

    componentDidUpdate(prevProps, prevState) {
        if (prevState.invalid && this.props.value !== prevProps.value) {
            this.setState({invalid: false}); //eslint-disable-line react/no-did-update-set-state
        }

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
            const validValues = this.state.values.filter((value) => optionValues.includes(value.value));

            this.setState({
                options,
                values: validValues,
                isLoading: false,
            });
        } catch (err) {
            this.setState({
                error: err,
                isLoading: false,
            });
        }
    }

    onChange = (values) => {
        this.setState({values});

        if (!values) {
            this.props.onChange(this.props.isMulti ? [] : '');
            return;
        }

        this.props.onChange(this.props.isMulti ? values.map((v) => v.value) : values.value);

        if (this.props.resetInvalidOnChange) {
            this.setState({invalid: false});
        }
    };

    isValid = () => {
        if (!this.props.required) {
            return true;
        }

        const valid = Boolean(this.props.value && this.props.value.toString().length !== 0);
        this.setState({invalid: !valid});
        return valid;
    };

    errorComponent = () => {
        if (!this.state.error) {
            return null;
        }

        return (
            <p className='alert alert-danger'>
                <i
                    className='fa fa-warning'
                    title='Warning Icon'
                />
                <span> {this.state.error.message}</span>
            </p>
        );
    };

    validationError = () => {
        if (!this.props.required || !this.state.invalid) {
            return null;
        }

        return (
            <p className='help-text error-text'>
                <span>{'This field is required.'}</span>
            </p>
        );
    };

    render = () => {
        return (
            <Setting {...this.props}>
                <ReactSelect
                    {...this.props}
                    onChange={this.onChange}
                    options={this.state.options}
                    value={this.state.values}
                    isLoading={this.state.isLoading}
                    closeMenuOnSelect={false}
                    hideSelectedOptions={true}
                    styles={getStyleForReactSelect(this.props.theme)}
                />
                {this.errorComponent()}
                {this.validationError()}
            </Setting>
        );
    };
}
