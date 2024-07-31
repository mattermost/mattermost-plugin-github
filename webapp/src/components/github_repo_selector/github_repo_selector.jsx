// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

import ReactSelectSetting from 'components/react_select_setting';

const initialState = {
    invalid: false,
    error: null,
};

export default class GithubRepoSelector extends PureComponent {
    static propTypes = {
        yourRepos: PropTypes.array.isRequired,
        theme: PropTypes.object.isRequired,
        onChange: PropTypes.func.isRequired,
        value: PropTypes.string,
        addValidate: PropTypes.func,
        removeValidate: PropTypes.func,
        actions: PropTypes.shape({
            getRepos: PropTypes.func.isRequired,
        }).isRequired,
    };

    constructor(props) {
        super(props);
        this.state = initialState;
    }

    componentDidMount() {
        this.props.actions.getRepos();
    }

    onChange = (_, name) => {
        const repo = this.props.yourRepos.find((r) => r.full_name === name);
        this.props.onChange({name, permissions: repo.permissions});
    }

    render() {
        const repoOptions = this.props.yourRepos.map((item) => ({value: item.full_name, label: item.full_name}));

        return (
            <div className={'form-group x3'}>
                <ReactSelectSetting
                    name={'repo'}
                    label={'Repository'}
                    limitOptions={true}
                    required={true}
                    onChange={this.onChange}
                    options={repoOptions}
                    isMulti={false}
                    key={'repo'}
                    theme={this.props.theme}
                    addValidate={this.props.addValidate}
                    removeValidate={this.props.removeValidate}
                    value={repoOptions.find((option) => option.value === this.props.value)}
                />
                <div
                    className={'help-text'}
                    style={{marginTop: '8px', marginBottom: '24px'}}
                >
                    {'Returns GitHub repositories connected to the user account'}
                </div>
            </div>
        );
    }
}
