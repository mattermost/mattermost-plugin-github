const {connect} = window['react-redux'];
const {bindActionCreators} = window.redux;

import {getReviews, getUnreads} from '../../actions';

import SidebarButtons from './sidebar_buttons.jsx';

function mapStateToProps(state, ownProps) {
    return {
        connected: state['plugins-github'].connected,
        username: state['plugins-github'].username,
        clientId: state['plugins-github'].clientId,
        reviews: state['plugins-github'].reviews,
        unreads: state['plugins-github'].unreads,
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            getReviews,
            getUnreads,
        }, dispatch)
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(SidebarButtons);
