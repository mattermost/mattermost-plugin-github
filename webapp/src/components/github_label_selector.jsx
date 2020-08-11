// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

import IssueAttributeSelector from 'components/issue_attribute_selector';
import Client from 'client';

export default class GithubLabelSelector extends PureComponent {
    static propTypes = {
        repo: PropTypes.string.isRequired,
        theme: PropTypes.object.isRequired,
        selectedLabels: PropTypes.array.isRequired,
        onChange: PropTypes.func.isRequired,
    };

    loadLabels = async () => {
        if (this.props.repo === '') {
            return [];
        }

        try {
            const options = await Client.getLabels(this.props.repo) || [];
            return options.map((option) => ({
                value: option.name,
                label: option.name,
            }));
        } catch (err) {
            throw new Error('Failed to load labels');
        }
    };

    onChange = (selection) => {
        if (!selection) {
            this.onChange([]);
        }

        this.props.onChange(selection.map((s) => s.value));
    }

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
