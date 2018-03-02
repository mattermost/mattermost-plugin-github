const {connect} = window['react-redux'];
const {bindActionCreators} = window.redux;

import {getBool} from 'mattermost-redux/selectors/entities/preferences';
import {displayUsernameForUser} from '../../utils/user_utils';

import {requestReviewers} from '../../actions';

import PostTypePullRequest from './post_type_pull_request.jsx';

function mapStateToProps(state, ownProps) {
    const post = ownProps.post || {};
    const user = state.entities.users.profiles[post.user_id] || {};

    return {
        ...ownProps,
        creatorName: displayUsernameForUser(user, state.entities.general.config),
        useMilitaryTime: getBool(state, 'display_settings', 'use_military_time', false)
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            requestReviewers
        }, dispatch)
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(PostTypePullRequest);
