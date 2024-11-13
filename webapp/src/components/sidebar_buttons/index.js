// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getConnected, updateRhsState, getSidebarContent} from '../../actions';

import manifest from '../../manifest';

import SidebarButtons from './sidebar_buttons.jsx';

function mapStateToProps(state) {
    const {id: pluginId} = manifest;
    return {
        connected: state[`plugins-${pluginId}`].connected,
        clientId: state[`plugins-${pluginId}`].clientId,
        reviews: state[`plugins-${pluginId}`].sidebarContent.reviews,
        yourPrs: state[`plugins-${pluginId}`].sidebarContent.prs,
        yourAssignments: state[`plugins-${pluginId}`].sidebarContent.assignments,
        unreads: state[`plugins-${pluginId}`].sidebarContent.unreads,
        baseURL: state[`plugins-${pluginId}`].baseURL,
        showRHSPlugin: state[`plugins-${pluginId}`].rhsPluginAction,
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            getConnected,
            updateRhsState,
            getSidebarContent,
        }, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(SidebarButtons);
