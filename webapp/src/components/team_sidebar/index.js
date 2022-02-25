// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';

import {configuration} from '../../selectors';

import TeamSidebar from './team_sidebar.jsx';

function mapStateToProps(state) {
    const members = state.entities.teams.myMembers || {};

    const sidebarEnabled = configuration(state).left_sidebar_enabled;
    const show = sidebarEnabled && Object.keys(members).length > 1;

    return {
        show,
    };
}

export default connect(mapStateToProps)(TeamSidebar);
