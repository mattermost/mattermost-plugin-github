// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

import IssueAttributeSelector from 'components/issue_attribute_selector';
import Client from 'client';

export default class GithubMilestoneSelector extends PureComponent {
    static propTypes = {
        repo: PropTypes.string.isRequired,
        theme: PropTypes.object.isRequired,
        selectedMilestone: PropTypes.object.isRequired,
        onChange: PropTypes.func.isRequired,
    };

    loadMilestones = async () => {
        if (this.props.repo === '') {
            return [];
        }

        try {
            const options = await Client.getMilestones(this.props.repo) || [];
            return options.map((option) => ({
                value: option.number,
                label: option.title,
            }));
        } catch (err) {
            throw new Error('Failed to load milestones');
        }
    };

    onChange = (selection) => {
        if (!selection || !selection.value) {
            this.props.onChange(null);
            return;
        }

        this.props.onChange(selection);
    }

    render() {
        return (
            <div className='form-group margin-bottom x3'>
                <label className='control-label margin-bottom x2'>
                    {'Milestone'}
                </label>
                <IssueAttributeSelector
                    {...this.props}
                    isMulti={false}
                    selection={this.props.selectedMilestone}
                    loadOptions={this.loadMilestones}
                    onChange={this.onChange}
                />
            </div>
        );
    }
}
