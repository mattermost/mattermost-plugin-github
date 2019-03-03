import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getGitHubUser} from '../../actions';

import UserAttribute from './user_attribute.jsx';

function mapStateToProps(state, ownProps) {
    const id = ownProps.user ? ownProps.user.id : '';
    const user = state['plugins-github'].githubUsers[id] || {};

    return {
        id,
        username: user.username,
        enterpriseURL: state['plugins-github'].enterpriseURL,
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            getGitHubUser,
        }, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(UserAttribute);
