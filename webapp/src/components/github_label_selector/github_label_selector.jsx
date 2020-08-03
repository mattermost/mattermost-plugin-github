// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {Component} from 'react';
import PropTypes from 'prop-types';

import BackendSelector from 'components/backend_selector';

export default class GithubLabelSelector extends Component {
    static propTypes = {
        repo: PropTypes.string,
        theme: PropTypes.object.isRequired,
        labels: PropTypes.array.isRequired,
        onChange: PropTypes.func,
        actions: PropTypes.shape({
            getLabels: PropTypes.func.isRequired,
        }).isRequired,
    };

    // prevent re-render if the selected repository remains unchanged
    shouldComponentUpdate(nextProps) {
        return this.props.repo !== nextProps.repo;
    }

    handleChange = (items) => {
        if (!items || items.length === 0) {
            return;
        }

        this.props.onChange(items);
    };

    render() {
        const loadLabels = async () => {
            if (this.props.repo === '') {
                return [];
            }

            await this.props.actions.getLabels(this.props.repo);

            return this.props.labels.map((item) => ({
                value: item.name,
                label: item.name,
            }));
        };

        return (
            <BackendSelector
                name={'labels'}
                required={false}
                isMulti={true}
                theme={this.props.theme}
                onChange={this.handleChange}
                load={loadLabels}
            />
        );
    }
}
