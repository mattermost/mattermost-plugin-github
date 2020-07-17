// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getConnected, getReviews, getUnreads, getYourAssignments, getYourPrs, updateRhsState} from '../../actions';

import {id as pluginId} from '../../manifest';

import SidebarButtons from './sidebar_buttons.jsx';

function mapStateToProps(state) {
    return {
        connected: state[`plugins-${pluginId}`].connected,
        clientId: state[`plugins-${pluginId}`].clientId,
        reviews: state[`plugins-${pluginId}`].reviews,
        yourPrs: state[`plugins-${pluginId}`].yourPrs,
        yourAssignments: state[`plugins-${pluginId}`].yourAssignments,
        unreads: state[`plugins-${pluginId}`].unreads,
        enterpriseURL: state[`plugins-${pluginId}`].enterpriseURL,
        showRHSPlugin: state[`plugins-${pluginId}`].rhsPluginAction,
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            getConnected,
            getReviews,
            getUnreads,
            getYourPrs,
            getYourAssignments,
            updateRhsState,
        }, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(SidebarButtons);
