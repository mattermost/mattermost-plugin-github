// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getAssigneeOptions} from '../../actions';

import ForgejoAssigneeSelector from './forgejo_assignee_selector.jsx';

const mapDispatchToProps = (dispatch) => ({
    actions: bindActionCreators({getAssigneeOptions}, dispatch),
});

export default connect(
    null,
    mapDispatchToProps,
)(ForgejoAssigneeSelector);
