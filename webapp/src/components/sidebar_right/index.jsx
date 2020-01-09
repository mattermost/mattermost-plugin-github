// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getYourPrsExtraInfo} from '../../actions';

import SidebarRight from './sidebar_right.jsx';

function mapStateToProps(state) {
    return {
        username: state['plugins-github'].username,
        reviews: state['plugins-github'].reviews,
        yourPrs: state['plugins-github'].yourPrs,
        yourPrsExtraInfo: state['plugins-github'].yourPrsExtraInfo,
        yourAssignments: state['plugins-github'].yourAssignments,
        unreads: state['plugins-github'].unreads,
        enterpriseURL: state['plugins-github'].enterpriseURL,
        org: state['plugins-github'].organization,
        rhsState: state['plugins-github'].rhsState,
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            getYourPrsExtraInfo,
        }, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(SidebarRight);
