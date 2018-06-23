const {connect} = window['react-redux'];
const {bindActionCreators} = window.redux;

import {getReviews} from '../../actions';

import TeamSidebar from './team_sidebar.jsx';

function mapStateToProps(state, ownProps) {
    return {
        connected: state['plugins-github'].connected,
        pullRequests: state['plugins-github'].reviews,
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            getReviews,
        }, dispatch)
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(TeamSidebar);
