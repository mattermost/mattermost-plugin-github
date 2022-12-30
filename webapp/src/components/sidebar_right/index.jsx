// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getReviewsDetails, getYourPrsDetails} from '../../actions';
import {id as pluginId} from '../../manifest';

import SidebarRight from './sidebar_right.jsx';

function mapPrsToDetails(prs, details) {
    if (!prs) {
        return [];
    }

    return prs.map((pr) => {
        let foundDetails;
        if (details) {
            foundDetails = details.find((prDetails) => {
                return (pr.repository_url === prDetails.url) && (pr.number === prDetails.number);
            });
        }
        if (!foundDetails) {
            return pr;
        }

        return {
            ...pr,
            status: foundDetails.status,
            mergeable: foundDetails.mergeable,
            requestedReviewers: foundDetails.requestedReviewers,
            reviews: foundDetails.reviews,
        };
    });
}

function mapStateToProps(state) {
    return {
        username: state[`plugins-${pluginId}`].username,
        reviews: mapPrsToDetails(state[`plugins-${pluginId}`].sidebarContent.reviews, state[`plugins-${pluginId}`].reviewsDetails),
        yourPrs: mapPrsToDetails(state[`plugins-${pluginId}`].sidebarContent.prs, state[`plugins-${pluginId}`].yourPrsDetails),
        yourAssignments: state[`plugins-${pluginId}`].sidebarContent.assignments,
        unreads: state[`plugins-${pluginId}`].sidebarContent.unreads,
        enterpriseURL: state[`plugins-${pluginId}`].enterpriseURL,
        org: state[`plugins-${pluginId}`].organization,
        rhsState: state[`plugins-${pluginId}`].rhsState,
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
