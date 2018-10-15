import {connect} from 'react-redux';

import SidebarHeader from './sidebar_header.jsx';

function mapStateToProps(state) {
    const members = state.entities.teams.myMembers || {};
    return {
        show: Object.keys(members).length <= 1,
    };
}

export default connect(mapStateToProps)(SidebarHeader);
