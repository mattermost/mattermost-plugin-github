// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

import IssueAttributeSelector from 'components/issue_attribute_selector';

export default class GithubLabelSelector extends PureComponent {
    static propTypes = {
        repo: PropTypes.string.isRequired,
        theme: PropTypes.object.isRequired,
        selectedLabels: PropTypes.array.isRequired,
        onChange: PropTypes.func.isRequired,
        actions: PropTypes.shape({
            getLabels: PropTypes.func.isRequired,
        }).isRequired,
    };

    loadLabels = async () => {
        if (this.props.repo === '') {
            return [];
        }

        const options = await this.props.actions.getLabels(this.props.repo);

        if (options.error) {
            throw new Error('Failed to load labels');
        }

        if (!options || !options.data) {
            return [];
        }

        return options.data.map((option) => ({
            value: option.name,
            label: option.name,
        }));
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
                    selection={this.props.selectedLabels}
                    loadOptions={this.loadLabels}
                />
            </div>
        );
    }
}
