// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import ReactSelect from 'react-select';
import PropTypes from 'prop-types';

import {getStyleForReactSelect} from 'utils/styles';
import Setting from 'components/setting';

export default class IssueAttributeSelector extends PureComponent {
    static propTypes = {
        name: PropTypes.string,
        repo: PropTypes.string,
        required: PropTypes.bool,
        isMulti: PropTypes.bool,
        resetInvalidOnChange: PropTypes.bool,
        addValidate: PropTypes.func,
        removeValidate: PropTypes.func,
        onChange: PropTypes.func,
        load: PropTypes.func,
        theme: PropTypes.object.isRequired,
        value: PropTypes.oneOfType([
            PropTypes.object,
            PropTypes.array,
            PropTypes.string,
        ]),
    };

    constructor(props) {
        super(props);

        this.state = {
            options: [],
            invalid: false,
            error: '',
            loading: false,
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

        if (prevProps.repo !== this.props.repo) {
            this.loadOptions();
        }
    }

    onChange = (options) => {
        if (!options) {
            this.props.onChange(this.props.isMulti ? [] : '');
            return;
        }

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

    loadOptions = async () => {
        this.setState({isLoading: true});
        const options = await this.props.load();
        this.setState({isLoading: false});

        // prevent re-render if the options remain unchanged
        if (JSON.stringify(options) === JSON.stringify(this.state.options)) {
            return;
        }

        this.setState({options});
    }

    render = () => {
        return (
            <Setting {...this.props}>
                <ReactSelect
                    {...this.props}
                    onChange={this.onChange}
                    options={this.state.options}
                    isLoading={this.state.isLoading}
                    styles={getStyleForReactSelect(this.props.theme)}
                />
                {this.errorComponent()}
                {this.validationError()}
            </Setting>
        );
    };
}
