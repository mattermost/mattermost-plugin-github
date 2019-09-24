// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getReviews, getUnreads, getYourPrs, getYourAssignments, updateRhsState} from '../../actions';

import SidebarButtons from './sidebar_buttons.jsx';

function mapStateToProps(state) {
    return {
        connected: state['plugins-github'].connected,
        clientId: state['plugins-github'].clientId,
        reviews: state['plugins-github'].reviews,
        yourPrs: state['plugins-github'].yourPrs,
        yourAssignments: state['plugins-github'].yourAssignments,
        unreads: state['plugins-github'].unreads,
        enterpriseURL: state['plugins-github'].enterpriseURL,
        showRHSPlugin: state['plugins-github'].rhsPluginAction,
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            getReviews,
            getUnreads,
            getYourPrs,
            getYourAssignments,
            updateRhsState,
        }, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(SidebarButtons);
