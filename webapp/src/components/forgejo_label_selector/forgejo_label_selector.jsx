// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

import IssueAttributeSelector from 'components/issue_attribute_selector';

export default class ForgejoLabelSelector extends PureComponent {
    static propTypes = {
        repoName: PropTypes.string.isRequired,
        theme: PropTypes.object.isRequired,
        selectedLabels: PropTypes.array.isRequired,
        onChange: PropTypes.func.isRequired,
        actions: PropTypes.shape({
            getLabelOptions: PropTypes.func.isRequired,
        }).isRequired,
    };

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

        return options.data.map((option) => ({
            value: option.name,
            label: option.name,
        }));
    };

    onChange = (selection) => this.props.onChange(selection.map((s) => s.value));

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
