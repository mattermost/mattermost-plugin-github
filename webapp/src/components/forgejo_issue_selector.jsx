// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

import debounce from 'debounce-promise';
import AsyncSelect from 'react-select/async';

import {getStyleForReactSelect} from '@/utils/styles';
import Client from '@/client';

const searchDebounceDelay = 400;

export default class ForgejoIssueSelector extends PureComponent {
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

    handleIssueSearchTermChange = (inputValue) => {
        return this.debouncedSearchIssues(inputValue);
    };

    searchIssues = (text) => {
        const textEncoded = encodeURIComponent(text.trim().replace(/"/g, '\\"'));

        return Client.searchIssues(textEncoded).then((data) => {
            if (!Array.isArray(data)) {
                return [];
            }
            return data.map((item) => {
                const repoParts = item.repository_url.split('/');
                let prefix = '';
                if (repoParts.length >= 2) {
                    prefix = repoParts[repoParts.length - 2] + '/' + repoParts[repoParts.length - 1] + ', ';
                }
                return ({value: item, label: prefix + '#' + item.number + ': ' + item.title, isDisabled: item.locked});
            });
        }).catch((e) => {
            this.setState({error: e});
        });
    };

    debouncedSearchIssues = debounce(this.searchIssues, searchDebounceDelay);

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
                    {'Forgejo Issue'}
                </label>
                {this.props.required && requiredStar}
                <AsyncSelect
                    name={'issue'}
                    placeholder={'Search for issues containing text...'}
                    onChange={this.onChange}
                    required={true}
                    disabled={false}
                    isMulti={false}
                    isClearable={true}
                    defaultOptions={true}
                    loadOptions={this.handleIssueSearchTermChange}
                    menuPortalTarget={document.body}
                    menuPlacement='auto'
                    styles={getStyleForReactSelect(this.props.theme)}
                />
                {validationError}
                {issueError}
                <div className={'help-text'}>
                    {'Returns issues sorted by most recently updated.'} <br/>
                </div>
            </div>
        );
    }
}
