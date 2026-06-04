// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

import IssueAttributeSelector from '@/components/issue_attribute_selector';

export default class ForgejoAssigneeSelector extends PureComponent {
    static propTypes = {
        repoName: PropTypes.string.isRequired,
        theme: PropTypes.object.isRequired,
        selectedAssignees: PropTypes.array.isRequired,
        onChange: PropTypes.func.isRequired,
        actions: PropTypes.shape({
            getAssigneeOptions: PropTypes.func.isRequired,
        }).isRequired,
    };

    loadAssignees = async () => {
        if (this.props.repoName === '') {
            return [];
        }

        const options = await this.props.actions.getAssigneeOptions(this.props.repoName);

        if (options.error) {
            throw new Error('Failed to load assignees');
        }

        if (!options || !options.data) {
            return [];
        }

        return options.data.map((option) => ({
            value: option.login,
            label: option.login,
        }));
    };

    onChange = (selection) => this.props.onChange(selection.map((s) => s.value));

    render() {
        return (
            <div className='form-group margin-bottom x3'>
                <label className='control-label margin-bottom x2'>
                    {'Assignees'}
                </label>
                <IssueAttributeSelector
                    {...this.props}
                    isMulti={true}
                    onChange={this.onChange}
                    selection={this.props.selectedAssignees}
                    loadOptions={this.loadAssignees}
                />
            </div>
        );
    }
}
