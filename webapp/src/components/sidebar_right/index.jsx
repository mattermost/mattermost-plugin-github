import {connect} from 'react-redux';

import SidebarRight from './sidebar_right.jsx';

function mapStateToProps(state) {
    return {
        username: state['plugins-github'].username,
        reviews: state['plugins-github'].reviews,
        yourPrs: state['plugins-github'].yourPrs,
        yourAssignments: state['plugins-github'].yourAssignments,
        unreads: state['plugins-github'].unreads,
        enterpriseURL: state['plugins-github'].enterpriseURL,
        org: state['plugins-github'].organization,
        rhsState: state['plugins-github'].rhsState,
    };
}

export default connect(mapStateToProps)(SidebarRight);
