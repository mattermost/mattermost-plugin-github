// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

import debounce from 'debounce-promise';
import AsyncSelect from 'react-select/async';

import {getStyleForReactSelect} from 'utils/styles';

import Client from 'client';

const searchDebounceDelay = 400;

export default class GithubRepoSelector extends PureComponent {
    static propTypes = {
        required: PropTypes.bool,
        theme: PropTypes.object.isRequired,
        onChange: PropTypes.func.isRequired,
        error: PropTypes.string,
        value: PropTypes.object,
    };

    constructor(props) {
        super(props);

        this.state = {invalid: false};
    }

    componentDidUpdate(prevProps, prevState) {
        if (prevState.invalid && this.props.value !== prevProps.value) {
            this.setState({invalid: false}); //eslint-disable-line react/no-did-update-set-state
        }
    }

    handleGetRepoValue = () => {
        return this.debouncedSearchRepo();
    };

    searchRepo = () => {
        return Client.getRepositories().then((data) => {
            return Array.isArray(data) ? data.map((item) => ({value: item, label: item.full_name, name: item.name})) : [];
        }).catch((e) => {
            this.setState({error: e});
        });
    };

    debouncedSearchRepo = debounce(this.searchRepo, searchDebounceDelay);

    onChange = (e) => {
        const value = e ? e.value : '';
        this.props.onChange(value);
    }

    isValid = () => {
        if (!this.props.required) {
            return true;
        }

        const valid = this.props.value && this.props.value.toString().length !== 0;
        this.setState({invalid: !valid});
        return valid;
    };

    render() {
        const {error} = this.props;
        const requiredStar = (
            <span
                className={'error-text'}
                style={{marginLeft: '3px'}}
            >
                {'*'}
            </span>
        );

        let issueError = null;
        if (error) {
            issueError = (
                <p className='help-text error-text'>
                    <span>{error}</span>
                </p>
            );
        }

        const serverError = this.state.error;
        let errComponent;
        if (this.state.error) {
            errComponent = (
                <p className='alert alert-danger'>
                    <i
                        className='fa fa-warning'
                        title='Warning Icon'
                    />
                    <span>{serverError.toString()}</span>
                </p>
            );
        }

        const requiredMsg = 'This field is required.';
        let validationError = null;
        if (this.props.required && this.state.invalid) {
            validationError = (
                <p className='help-text error-text'>
                    <span>{requiredMsg}</span>
                </p>
            );
        }

        return (
            <div className={'form-group margin-bottom x3'}>
                {errComponent}
                <label
                    className={'control-label'}
                    htmlFor={'issue'}
                >
                    {'GitHub Repository'}
                </label>
                {this.props.required && requiredStar}
                <AsyncSelect
                    name={'repository'}
                    placeholder={'Select repository to post the issue in'}
                    onChange={this.onChange}
                    required={true}
                    disabled={false}
                    isMulti={false}
                    isClearable={true}
                    defaultOptions={true}
                    loadOptions={this.handleGetRepoValue}
                    menuPortalTarget={document.body}
                    menuPlacement='auto'
                    styles={getStyleForReactSelect(this.props.theme)}
                />
                {validationError}
                {issueError}
                <div className={'help-text'}>
                    {'Returns GitHub repositories connected to the user account'} <br/>
                </div>
            </div>
        );
    }
}
