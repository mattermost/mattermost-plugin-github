// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getReviewsDetails, getYourPrsDetails} from '../../actions';

import {getSidebarData} from 'src/selectors';

import SidebarRight from './sidebar_right.jsx';

function mapStateToProps(state) {
    const {username, mentions, reviews, yourPrs, yourAssignments, unreads, enterpriseURL, orgs, rhsState} = getSidebarData(state);
    return {
        username,
        reviews,
        yourPrs,
        mentions,
        yourAssignments,
        unreads,
        enterpriseURL,
        orgs,
        rhsState,
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            getYourPrsDetails,
            getReviewsDetails,
        }, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(SidebarRight);
