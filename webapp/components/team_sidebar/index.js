const {connect} = window['react-redux'];
const {bindActionCreators} = window.redux;

import {getReviews, getMentions} from '../../actions';

import TeamSidebar from './team_sidebar.jsx';

function mapStateToProps(state, ownProps) {
    return {
        connected: state['plugins-github'].connected,
        username: state['plugins-github'].username,
        clientId: state['plugins-github'].clientId,
        reviews: state['plugins-github'].reviews,
        mentions: state['plugins-github'].mentions,
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            getReviews,
            getMentions,
        }, dispatch)
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(TeamSidebar);
