// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getLabels} from '../../actions';

import GithubLabelSelector from './github_label_selector.jsx';

const mapDispatchToProps = (dispatch) => ({
    actions: bindActionCreators({getLabels}, dispatch),
});

export default connect(
    null,
    mapDispatchToProps
)(GithubLabelSelector);
