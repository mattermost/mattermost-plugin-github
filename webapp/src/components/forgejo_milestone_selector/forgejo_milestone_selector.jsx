// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

import IssueAttributeSelector from '@/components/issue_attribute_selector';

export default class ForgejoMilestoneSelector extends PureComponent {
    static propTypes = {
        repoName: PropTypes.string.isRequired,
        theme: PropTypes.object.isRequired,
        selectedMilestone: PropTypes.object,
        onChange: PropTypes.func.isRequired,
        actions: PropTypes.shape({
            getMilestoneOptions: PropTypes.func.isRequired,
        }).isRequired,
    };

    loadMilestones = async () => {
        if (this.props.repoName === '') {
            return [];
        }

        const options = await this.props.actions.getMilestoneOptions(this.props.repoName);

        if (options.error) {
            throw new Error('Failed to load milestones');
        }

        if (!options || !options.data) {
            return [];
        }

        return options.data.map((option) => ({
            value: option.number,
            label: option.title,
        }));
    };

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
                />
            </div>
        );
    }
}
