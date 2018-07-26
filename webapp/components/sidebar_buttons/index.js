import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getReviews, getUnreads, getYourPrs} from '../../actions';

import SidebarButtons from './sidebar_buttons.jsx';

function mapStateToProps(state, ownProps) {
    return {
        connected: state['plugins-github'].connected,
        username: state['plugins-github'].username,
        clientId: state['plugins-github'].clientId,
        reviews: state['plugins-github'].reviews,
        yourPrs: state['plugins-github'].yourPrs,
        unreads: state['plugins-github'].unreads,
        enterpriseURL: state['plugins-github'].enterpriseURL,
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            getReviews,
            getUnreads,
            getYourPrs,
        }, dispatch)
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(SidebarButtons);
