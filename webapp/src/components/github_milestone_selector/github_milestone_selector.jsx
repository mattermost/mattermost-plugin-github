// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

import IssueAttributeSelector from 'components/issue_attribute_selector';

export default class GithubMilestoneSelector extends PureComponent {
    static propTypes = {
        repo: PropTypes.string.isRequired,
        theme: PropTypes.object.isRequired,
        selectedMilestone: PropTypes.number.isRequired,
        onChange: PropTypes.func.isRequired,
        milestones: PropTypes.array,
        actions: PropTypes.shape({
            getMilestones: PropTypes.func.isRequired,
        }).isRequired,
    };

    loadMilestones = async () => {
        if (this.props.repo === '') {
            return [];
        }

        const options = await this.props.actions.getMilestones(this.props.repo);

        if (options.error) {
            throw new Error('Failed to load milestones');
        }

        if (!options || !options.data) {
            return [];
        }

        return options.data.map((option) => ({
            value: option.title,
            label: option.title,
        }));
    };

    onChange = (selection) => {
        if (!this.props.milestones) {
            this.props.onChange(null);
            return;
        }

        // we have to find the selected milestone from the options in order to insert its number
        const milestone = this.props.milestones.find((m) => m.title === selection);
        this.props.onChange(milestone ? milestone.number : 0);
    }

    render() {
        let milestone = '';
        if (this.props.selectedMilestone > 0 && this.props.milestones) {
            milestone = this.props.milestones.find((m) => m.number === this.props.selectedMilestone).title;
        }

        return (
            <div className='form-group margin-bottom x3'>
                <label className='control-label margin-bottom x2'>
                    {'Milestone'}
                </label>
                <IssueAttributeSelector
                    {...this.props}
                    isMulti={false}
                    selection={milestone}
                    loadOptions={this.loadMilestones}
                    onChange={this.onChange}
                />
            </div>
        );
    }
}
