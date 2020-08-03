// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import AsyncSelect from 'react-select/async';
import PropTypes from 'prop-types';
import debounce from 'debounce-promise';

import {getStyleForReactSelect} from 'utils/styles';
import Setting from 'components/setting';

const SEARCH_DEBOUNCE_DELAY = 400;

export default class BackendSelector extends PureComponent {
    static propTypes = {
        name: PropTypes.string,
        required: PropTypes.bool,
        isMulti: PropTypes.bool,
        resetInvalidOnChange: PropTypes.bool,
        search: PropTypes.func,
        addValidate: PropTypes.func,
        removeValidate: PropTypes.func,
        onChange: PropTypes.func,
        theme: PropTypes.object.isRequired,
        fetchInitialSelectedValues: PropTypes.func,
        value: PropTypes.oneOfType([
            PropTypes.object,
            PropTypes.array,
            PropTypes.string,
        ]),
    };

    constructor(props) {
        super(props);

        this.state = {
            invalid: false,
            error: '',
            cachedSelectedOptions: [],
        };
    }

    componentDidMount() {
        if (this.props.addValidate) {
            this.props.addValidate(this.isValid);
        }

        this.props.fetchInitialSelectedValues().then((options) => {
            this.setState({
                cachedSelectedOptions: this.state.cachedSelectedOptions.concat(options),
            });
        }).catch((e) => this.setState({error: e}));
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
    }

    handleIssueSearchTermChange = (inputValue) => {
        return this.debouncedSearch(inputValue);
    };

    search = (userInput) => {
        return this.props.
            search(userInput).
            then((options) => options || []).
            catch((e) => {
                this.setState({error: e});
                return [];
            });
    };

    debouncedSearch = debounce(this.search, SEARCH_DEBOUNCE_DELAY);

    onChange = (options) => {
        if (!options) {
            this.props.onChange(this.props.isMulti ? [] : '');
            return;
        }

        this.setState({
            cachedSelectedOptions: this.state.cachedSelectedOptions.concat(options),
        });

        this.props.onChange(this.props.isMulti ? options.map((v) => v.value) : options.value);

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
                <span> {this.state.error.toString()}</span>
            </p>
        );
    }

    validationError = () => {
        if (!this.props.required || !this.state.invalid) {
            return null;
        }

        return (
            <p className='help-text error-text'>
                <span>{'This field is required.'}</span>
            </p>
        );
    }

    render = () => {
        const valueToOption = (v) => {
            if (
                this.state.cachedSelectedOptions &&
                this.state.cachedSelectedOptions.length
            ) {
                const selected = this.state.cachedSelectedOptions.find((option) => option.value === v);
                if (selected) {
                    return selected;
                }
            }

            // option's label hasn't been fetched yet
            return {
                label: v,
                value: v,
            };
        };

        const value = this.props.isMulti ? this.props.value.map(valueToOption) : valueToOption(this.props.value);

        return (
            <Setting {...this.props}>
                <AsyncSelect
                    {...this.props}
                    name={this.props.name}
                    value={value}
                    onChange={this.onChange}
                    required={this.props.required}
                    isMulti={this.props.isMulti}
                    defaultOptions={true}
                    loadOptions={this.handleIssueSearchTermChange}
                    menuPortalTarget={document.body}
                    menuPlacement='auto'
                    styles={getStyleForReactSelect(this.props.theme)}
                />
                {this.errorComponent}
                {this.validationError}
            </Setting>
        );
    };
}
