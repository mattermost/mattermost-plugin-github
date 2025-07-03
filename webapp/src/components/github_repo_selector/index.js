// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import manifest from '@/manifest';

import {getReposByOrg, getOrgs} from '../../actions';

import GithubRepoSelector from './github_repo_selector.jsx';

function mapStateToProps(state) {
    return {
        yourOrgs: state[`plugins-${manifest.id}`].yourOrgs,
        yourReposByOrg: state[`plugins-${manifest.id}`].yourReposByOrg,
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators(
            {
                getOrgs,
                getReposByOrg,
            },
            dispatch,
        ),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(GithubRepoSelector);
