// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getReviewsDetails, getYourPrsDetails, getSidebarContent} from '../../actions';

import {getSidebarData} from 'src/selectors';

import SidebarRight from './sidebar_right.jsx';

function mapStateToProps(state) {
    const {username, reviews, yourPrs, yourAssignments, unreads, enterpriseURL, orgs, rhsState, reviewTargetDays} = getSidebarData(state);
    return {
        username,
        reviews,
        yourPrs,
        yourAssignments,
        unreads,
        enterpriseURL,
        orgs,
        rhsState,
        reviewTargetDays,
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            getYourPrsDetails,
            getReviewsDetails,
            getSidebarContent,
        }, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(SidebarRight);
