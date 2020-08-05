// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getAssignees} from '../../actions';

import GithubAssigneeSelector from './github_assignee_selector.jsx';

const mapDispatchToProps = (dispatch) => ({
    actions: bindActionCreators({getAssignees}, dispatch),
});

export default connect(
    null,
    mapDispatchToProps
)(GithubAssigneeSelector);
