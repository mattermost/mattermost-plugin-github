// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';

import {Theme} from 'mattermost-redux/types/preferences';

import IssueAttributeSelector, {IssueAttributeSelectorSelection} from '../issue_attribute_selector';

import {GitHubLabelSelectorDispatchProps} from '.';

type Props = GitHubLabelSelectorDispatchProps & {
    repoName: string;
    theme: Theme;
    selectedLabels: string[];
    onChange: (selection: string[]) => void;
};

type Option = {
    name: string;
};

export default class GithubLabelSelector extends PureComponent<Props> {
    loadLabels = async () => {
        if (this.props.repoName === '') {
            return [];
        }

        const options = await this.props.actions.getLabelOptions(this.props.repoName);

        if (options.error) {
            throw new Error('Failed to load labels');
        }

        if (!options || !options.data) {
            return [];
        }

        return options.data.map((option: Option) => ({
            value: option.name,
            label: option.name,
        }));
    };

    onChange = (selection: IssueAttributeSelectorSelection) => {
        if (!selection || !Array.isArray(selection)) {
            return;
        }

        this.props.onChange(selection.map((s) => s.value));
    };

    render() {
        return (
            <div className='form-group margin-bottom x3'>
                <label className='control-label margin-bottom x2'>
                    {'Labels'}
                </label>
                <IssueAttributeSelector
                    {...this.props}
                    isMulti={true}
                    onChange={this.onChange}
                    selection={this.props.selectedLabels}
                    loadOptions={this.loadLabels}
                />
            </div>
        );
    }
}
