// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {id as pluginId} from 'manifest';
import {getMilestones} from '../../actions';

import GithubMilestoneSelector from './github_milestone_selector.jsx';

const mapDispatchToProps = (dispatch) => ({
    actions: bindActionCreators({getMilestones}, dispatch),
});

const mapStateToProps = (state) => ({
    milestones: state[`plugins-${pluginId}`].milestones,
});

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(GithubMilestoneSelector);
