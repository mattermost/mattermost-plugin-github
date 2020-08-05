// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

import IssueAttributeSelector from 'components/issue_attribute_selector';

export default class GithubMilestoneSelector extends PureComponent {
    static propTypes = {
        repo: PropTypes.string.isRequired,
        theme: PropTypes.object.isRequired,
        selectedMilestone: PropTypes.object.isRequired,
        onChange: PropTypes.func.isRequired,
        milestones: PropTypes.array.isRequired,
        actions: PropTypes.shape({
            getMilestones: PropTypes.func.isRequired,
        }).isRequired,
    };

    loadMilestones = async () => {
        if (this.props.repo === '') {
            return [];
        }

        const milestones = await this.props.actions.getMilestones(this.props.repo);

        if (!milestones || !milestones.data) {
            return [];
        }

        return milestones.data.map((milestone) => ({
            value: milestone.title,
            label: milestone.title,
        }));
    };

    onChange = (selection) => {
        if (!selection || Object.keys(selection).length === 0) {
            this.props.onChange({});
            return;
        }

        // we have to find the selected milestone from the options in order to insert its number
        const milestone = this.props.milestones.find((m) => m.title === selection.value);

        this.props.onChange({
            number: milestone.number,
            label: milestone.title,
            value: milestone.title,
        });
    }

    render() {
        const isSelectionBlank = Object.keys(this.props.selectedMilestone).length === 0;
        const selection = isSelectionBlank ? '' : {
            label: this.props.selectedMilestone.label,
            value: this.props.selectedMilestone.value,
        };

        return (
            <div className='form-group margin-bottom x3'>
                <label className='control-label margin-bottom x2'>
                    {'Milestone'}
                </label>
                <IssueAttributeSelector
                    {...this.props}
                    isMulti={false}
                    selection={selection}
                    loadOptions={this.loadMilestones}
                    onChange={this.onChange}
                />
            </div>
        );
    }
}
