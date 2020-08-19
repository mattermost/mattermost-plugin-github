// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getMilestoneOptions} from '../../actions';

import GithubMilestoneSelector from './github_milestone_selector.jsx';

const mapDispatchToProps = (dispatch) => ({
    actions: bindActionCreators({getMilestoneOptions}, dispatch),
});

export default connect(
    null,
    mapDispatchToProps
)(GithubMilestoneSelector);
