// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

import Setting from './setting.jsx';

export default class Input extends PureComponent {
    static propTypes = {
        id: PropTypes.string,
        label: PropTypes.node.isRequired,
        placeholder: PropTypes.string,
        helpText: PropTypes.node,
        value: PropTypes.oneOfType([
            PropTypes.string,
            PropTypes.number,
        ]),
        addValidate: PropTypes.func,
        removeValidate: PropTypes.func,
        maxLength: PropTypes.number,
        onChange: PropTypes.func,
        disabled: PropTypes.bool,
        required: PropTypes.bool,
        readOnly: PropTypes.bool,
        type: PropTypes.oneOf([
            'number',
            'input',
            'textarea',
        ]),
    };

    static defaultProps = {
        type: 'input',
        maxLength: null,
        required: false,
        readOnly: false,
    };

    constructor(props) {
        super(props);

        this.state = {invalid: false};
    }

    componentDidMount() {
        if (this.props.addValidate && this.props.id) {
            this.props.addValidate(this.props.id, this.isValid);
        }
    }

    componentWillUnmount() {
        if (this.props.removeValidate && this.props.id) {
            this.props.removeValidate(this.props.id);
        }
    }

    componentDidUpdate(prevProps, prevState) {
        if (prevState.invalid && this.props.value !== prevProps.value) {
            this.setState({invalid: false}); //eslint-disable-line react/no-did-update-set-state
        }
    }

    handleChange = (e) => {
        if (this.props.type === 'number') {
            this.props.onChange(parseInt(e.target.value, 10));
        } else {
            this.props.onChange(e.target.value);
        }
    };

    isValid = () => {
        if (!this.props.required) {
            return true;
        }
        const valid = this.props.value && this.props.value.toString().length !== 0;
        this.setState({invalid: !valid});
        return valid;
    };

    render() {
        const requiredMsg = 'This field is required.';
        const style = getStyle();
        const value = this.props.value || '';

        let validationError = null;
        if (this.props.required && this.state.invalid) {
            validationError = (
                <p className='help-text error-text'>
                    <span>{requiredMsg}</span>
                </p>
            );
        }

        let input = null;
        if (this.props.type === 'input') {
            input = (
                <input
                    id={this.props.id}
                    className='form-control'
                    type='text'
                    placeholder={this.props.placeholder}
                    value={value}
                    maxLength={this.props.maxLength}
                    onChange={this.handleChange}
                    disabled={this.props.disabled}
                    readOnly={this.props.readOnly}
                />
            );
        } else if (this.props.type === 'number') {
            input = (
                <input
                    id={this.props.id}
                    className='form-control'
                    type='number'
                    placeholder={this.props.placeholder}
                    value={value}
                    maxLength={this.props.maxLength}
                    onChange={this.handleChange}
                    disabled={this.props.disabled}
                    readOnly={this.props.readOnly}
                />
            );
        } else if (this.props.type === 'textarea') {
            input = (
                <textarea
                    style={style.textarea}
                    resize='none'
                    id={this.props.id}
                    className='form-control'
                    rows='5'
                    placeholder={this.props.placeholder}
                    value={value}
                    maxLength={this.props.maxLength}
                    onChange={this.handleChange}
                    disabled={this.props.disabled}
                    readOnly={this.props.readOnly}
                />
            );
        }

        return (
            <Setting
                label={this.props.label}
                helpText={this.props.helpText}
                inputId={this.props.id}
                required={this.props.required}
            >
                {input}
                {validationError}
            </Setting>
        );
    }
}

const getStyle = () => ({
    textarea: {
        resize: 'none',
    },
});
