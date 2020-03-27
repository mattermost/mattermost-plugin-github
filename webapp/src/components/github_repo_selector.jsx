// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

import Client from 'client';
import ReactSelectSetting from 'components/react_select_setting';

const initialState = {
    invalid: false,
    error: null,
    repoOptions: [],
};

export default class GithubRepoSelector extends PureComponent {
    static propTypes = {
        required: PropTypes.bool,
        theme: PropTypes.object.isRequired,
        onChange: PropTypes.func.isRequired,
        value: PropTypes.string,
        addValidate: PropTypes.func,
        removeValidate: PropTypes.func,
    };

    constructor(props) {
        super(props);
        this.state = initialState;
    }

    componentDidMount() {
        this.getRepos();
    }

    getRepos = () => {
        return Client.getRepositories().then((data) => {
            if (!data) {
                return;
            }
            const options = data.map((item) => ({value: item.name, label: item.full_name}));
            this.setState({repoOptions: options});
        }).catch((e) => {
            this.setState({error: e});
        });
    };

    onChange = (name, newValue) => {
        this.props.onChange(newValue);
    };

    render() {
        return (
            <div className={'form-group margin-bottom x3'}>
                <ReactSelectSetting
                    name={'repo'}
                    label={'Repository'}
                    limitOptions={true}
                    required={true}
                    onChange={this.onChange}
                    options={this.state.repoOptions}
                    isMulti={false}
                    key={'LT'}
                    theme={this.props.theme}
                    addValidate={this.props.addValidate}
                    removeValidate={this.props.removeValidate}
                    value={this.state.repoOptions.find((option) => option.value === this.props.value)}
                />
                <div className={'help-text'}>
                    {'Returns GitHub repositories connected to the user account'} <br/>
                </div>
            </div>
        );
    }
}
