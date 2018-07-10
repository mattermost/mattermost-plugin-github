const {connect} = window['react-redux'];
const {bindActionCreators} = window.redux;

import {getBool} from 'mattermost-redux/selectors/entities/preferences';
import {displayUsernameForUser} from '../../utils/user_utils';

import {updateSettings} from '../../actions';

import PostTypeSettings from './post_type_settings.jsx';

function mapStateToProps(state, ownProps) {
    return {
        username: state['plugins-github'].username,
        settings: state['plugins-github'].settings,
        enterpriseURL: state['plugins-github'].enterpriseURL,
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
           updateSettings,
        }, dispatch)
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(PostTypeSettings);
