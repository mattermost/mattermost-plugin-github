// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

import ReactSelectSetting from '@/components/react_select_setting';

export default class GithubRepoSelector extends PureComponent {
    static propTypes = {
        yourOrgs: PropTypes.array.isRequired,
        yourReposByOrg: PropTypes.object,
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

    static defaultProps = {
        yourReposByOrg: {repos: []},
    };

    constructor(props) {
        super(props);
        this.state = {org: ''};
    }

    componentDidMount() {
        this.props.actions.getOrgs();
    }

    getReposArray = () => {
        const {yourReposByOrg} = this.props;

        if (yourReposByOrg?.repos?.length > 0) {
            return yourReposByOrg.repos;
        }

        if (yourReposByOrg?.defaultRepo) {
            return [yourReposByOrg.defaultRepo];
        }

        return [];
    }

    componentDidUpdate(prevProps) {
        const repos = this.getReposArray();
        const defaultRepo = this.props.yourReposByOrg?.defaultRepo;
        const prevDefaultRepo = prevProps.yourReposByOrg?.defaultRepo;

        if ((!this.props.value || (defaultRepo && defaultRepo.full_name !== prevDefaultRepo?.full_name)) && defaultRepo) {
            this.onChangeForRepo(defaultRepo.name, defaultRepo.full_name);
        } else if (!defaultRepo && !this.props.value && repos.length > 0) {
            this.onChangeForRepo(repos[0].name, repos[0].full_name);
        }

        if (prevProps.yourOrgs !== this.props.yourOrgs && this.props.yourOrgs.length > 0) {
            const newOrg = this.props.yourOrgs[0].login;
            if (this.state.org !== newOrg) {
                this.onChangeForOrg(newOrg);
            }
        }
    }

    onChangeForOrg = (org) => {
        if (this.state.org !== org) {
            this.setState({org});
            this.props.actions.getReposByOrg(org, this.props.currentChannelId);
            this.props.onChange(null);
        }
    }

    onChangeForRepo = (_, name) => {
        const repos = this.getReposArray();

        const repo = repos.find((r) => r.full_name === name);
        if (repo) {
            this.props.onChange({name, permissions: repo.permissions});
        }
    }

    render() {
        const orgOptions = this.props.yourOrgs.map((org) => ({value: org.login, label: org.login}));

        const repos = this.getReposArray();
        const repoOptions = repos.map((repo) => ({value: repo.full_name, label: repo.name}));

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
