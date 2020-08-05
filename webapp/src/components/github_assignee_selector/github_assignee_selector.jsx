// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

import IssueAttributeSelector from 'components/issue_attribute_selector';

export default class GithubAssigneeSelector extends PureComponent {
    static propTypes = {
        repo: PropTypes.string.isRequired,
        theme: PropTypes.object.isRequired,
        selectedAssignees: PropTypes.array.isRequired,
        onChange: PropTypes.func.isRequired,
        actions: PropTypes.shape({
            getAssignees: PropTypes.func.isRequired,
        }).isRequired,
    };

    loadAssignees = async () => {
        if (this.props.repo === '') {
            return [];
        }

        const assignees = await this.props.actions.getAssignees(this.props.repo);

        if (!assignees || !assignees.data) {
            return [];
        }

        return assignees.data.map((assignee) => ({
            value: assignee.login,
            label: assignee.login,
        }));
    };

    render() {
        return (
            <div className='form-group margin-bottom x3'>
                <label className='control-label margin-bottom x2'>
                    {'Assignees'}
                </label>
                <IssueAttributeSelector
                    {...this.props}
                    isMulti={true}
                    selection={this.props.selectedAssignees}
                    loadOptions={this.loadAssignees}
                />
            </div>
        );
    }
}
