// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getGitHubUser} from '../../actions';

import {id as pluginId} from '../../manifest';

import UserAttribute from './user_attribute.jsx';

function mapStateToProps(state, ownProps) {
    const id = ownProps.user ? ownProps.user.id : '';
    const user = state[`plugins-${pluginId}`].githubUsers[id] || {};

    return {
        id,
        username: user.username,
        enterpriseURL: state[`plugins-${pluginId}`].enterpriseURL,
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            getGitHubUser,
        }, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(UserAttribute);
