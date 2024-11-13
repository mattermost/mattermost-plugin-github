// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getForgejoUser} from '../../actions';

import manifest from '../../manifest';

import UserAttribute from './user_attribute.jsx';

function mapStateToProps(state, ownProps) {
    const {id: pluginId} = manifest;
    const id = ownProps.user ? ownProps.user.id : '';
    const user = state[`plugins-${pluginId}`].forgejoUsers[id] || {};

    return {
        id,
        username: user.username,
        baseURL: state[`plugins-${pluginId}`].baseURL,
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            getForgejoUser,
        }, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(UserAttribute);
