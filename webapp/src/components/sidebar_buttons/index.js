import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getReviews, getUnreads, getYourPrs, getYourAssignments} from '../../actions';

import SidebarButtons from './sidebar_buttons.jsx';

function mapStateToProps(state) {
    return {
        connected: state['plugins-github'].connected,
        username: state['plugins-github'].username,
        clientId: state['plugins-github'].clientId,
        reviews: state['plugins-github'].reviews,
        yourPrs: state['plugins-github'].yourPrs,
        yourAssignments: state['plugins-github'].yourAssignments,
        unreads: state['plugins-github'].unreads,
        enterpriseURL: state['plugins-github'].enterpriseURL,
        org: state['plugins-github'].organization,
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            getReviews,
            getUnreads,
            getYourPrs,
            getYourAssignments,
        }, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(SidebarButtons);
