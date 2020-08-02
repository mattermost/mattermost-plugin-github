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

    onChange = (name, newValue) => {
        const newID = this.props.yourRepos.reduce((id, repo) => {
            return repo.full_name === newValue ? repo.id : id;
        }, 0);

        // return if there's no match (ideally this should never happen)
        if (newID === 0) {
            return;
        }

        this.props.onChange(newID, newValue);
    };

    render() {
        const repoOptions = this.props.yourRepos.map((item) => ({value: item.name, label: item.full_name}));

        return (
            <div className={'form-group margin-bottom x3'}>
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
                    value={repoOptions.find((option) => option.label === this.props.value)}
                />
                <div className={'help-text'}>
                    {'Returns GitHub repositories connected to the user account'} <br/>
                </div>
            </div>
        );
    }
}
