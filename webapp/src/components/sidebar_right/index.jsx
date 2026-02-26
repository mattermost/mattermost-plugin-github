// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getReviewsDetails, getYourPrsDetails, selectPR, clearSelectedPR, getPRReviewThreads, getAIAgents} from '../../actions';

import {getSidebarData, getSelectedPR} from 'src/selectors';

import SidebarRight from './sidebar_right.jsx';

function mapStateToProps(state) {
    const {username, reviews, yourPrs, yourAssignments, unreads, enterpriseURL, orgs, rhsState} = getSidebarData(state);
    return {
        username,
        reviews,
        yourPrs,
        yourAssignments,
        unreads,
        enterpriseURL,
        orgs,
        rhsState,
        selectedPR: getSelectedPR(state),
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            getYourPrsDetails,
            getReviewsDetails,
            selectPR,
            clearSelectedPR,
            getPRReviewThreads,
            getAIAgents,
        }, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(SidebarRight);
