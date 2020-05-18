// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {id as pluginId} from 'manifest';
import {getRepos} from '../../actions';

import GithubRepoSelector from './github_repo_selector.jsx';

function mapStateToProps(state) {
    return {
        yourRepos: state[`plugins-${pluginId}`].yourRepos,
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            getRepos,
        }, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(GithubRepoSelector);
