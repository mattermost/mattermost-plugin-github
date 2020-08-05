// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

import IssueAttributeSelector from 'components/issue_attribute_selector';

export default class GithubLabelSelector extends PureComponent {
    static propTypes = {
        repo: PropTypes.string.isRequired,
        theme: PropTypes.object.isRequired,
        onChange: PropTypes.func.isRequired,
        actions: PropTypes.shape({
            getLabels: PropTypes.func.isRequired,
        }).isRequired,
    };

    loadLabels = async () => {
        if (this.props.repo === '') {
            return [];
        }

        const labels = await this.props.actions.getLabels(this.props.repo);

        return labels.data.map((item) => ({
            value: item.name,
            label: item.name,
        }));
    };

    render() {
        return (
            <div className='form-group margin-bottom x3'>
                <label className='control-label margin-bottom x2'>
                    {'Labels'}
                </label>
                <IssueAttributeSelector
                    repo={this.props.repo}
                    required={false}
                    isMulti={true}
                    theme={this.props.theme}
                    onChange={this.props.onChange}
                    loadOptions={this.loadLabels}
                />
            </div>
        );
    }
}
