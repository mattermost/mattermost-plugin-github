// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

import ReactSelectSetting from '@/components/react_select_setting';

const initialState = {
    invalid: false,
    error: null,
    org: '',
};

export default class GithubRepoSelector extends PureComponent {
    static propTypes = {
        yourOrgs: PropTypes.array.isRequired,
        yourReposByOrg: PropTypes.array,
        theme: PropTypes.object.isRequired,
        onChange: PropTypes.func.isRequired,
        value: PropTypes.string,
        currentChannelId: PropTypes.string,
        addValidate: PropTypes.func,
        removeValidate: PropTypes.func,
        actions: PropTypes.shape({
            getOrgs: PropTypes.func.isRequired,
            getReposByOrg: PropTypes.func.isRequired,
        }).isRequired,
    };

    constructor(props) {
        super(props);
        this.state = initialState;
    }

    componentDidMount() {
        this.props.actions.getOrgs();
    }

    componentDidUpdate(prevProps) {
        if (prevProps.yourOrgs !== this.props.yourOrgs) {
            if (this.props.yourOrgs.length) {
                this.onChangeForOrg(0, this.props.yourOrgs[0].login);
            }
        }
    }

    onChangeForOrg = (_, org) => {
        if (this.state.org !== org) {
            this.setState({org});
            this.props.actions.getReposByOrg(org);
            this.props.onChange(null);
        }
    }

    onChangeForRepo = (_, name) => {
        const repo = this.props.yourReposByOrg.find((r) => r.full_name === name);
        this.props.onChange({name, permissions: repo.permissions});
    }

    render() {
        const orgOptions = this.props.yourOrgs.map((item) => ({value: item.login, label: item.login}));
        const repoOptions = this.props.yourReposByOrg.map((item) => ({value: item.full_name, label: item.name}));

        let orgSelector = null;
        let helperTextForRepoSelector = 'Returns GitHub repositories connected to the user account';

        // If there are no organizations for authenticated user, then don't show organization selector
        if (orgOptions.length > 1) {
            orgSelector = (
                <>
                    <ReactSelectSetting
                        name={'org'}
                        label={'Organization'}
                        limitOptions={true}
                        required={true}
                        onChange={this.onChangeForOrg}
                        options={orgOptions}
                        isMulti={false}
                        key={'org'}
                        theme={this.props.theme}
                        addValidate={this.props.addValidate}
                        formatGroupLabel='user repositories'
                        removeValidate={this.props.removeValidate}
                        value={orgOptions.find((option) => option.value === this.state.org)}
                    />
                    <div
                        className='help-text'
                        style={{marginBottom: '15px'}}
                    >
                        {'Returns GitHub organizations connected to the user account'}
                    </div>
                </>
            );
            helperTextForRepoSelector = 'Returns GitHub repositories under selected organizations';
        }

        return (
            <div className={'form-group margin-bottom x3'}>
                {orgSelector}
                <ReactSelectSetting
                    name={'repo'}
                    label={'Repository'}
                    limitOptions={true}
                    required={true}
                    onChange={this.onChangeForRepo}
                    options={repoOptions}
                    isMulti={false}
                    key={'repo'}
                    theme={this.props.theme}
                    addValidate={this.props.addValidate}
                    removeValidate={this.props.removeValidate}
                    value={repoOptions.find((option) => option.value === this.props.value)}
                />
                <div className={'help-text'}>
                    {helperTextForRepoSelector}
                </div>
            </div>
        );
    }
}
