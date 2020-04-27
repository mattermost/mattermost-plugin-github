// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';
import {getPost} from 'mattermost-redux/selectors/entities/posts';
import {getCurrentTeam} from 'mattermost-redux/selectors/entities/teams';

import {id as pluginId} from 'manifest';
import {closeCreateIssueModal, createIssue} from 'actions';

import CreateIssueModal from './create_issue';

const mapStateToProps = (state) => {
    const postId = state[`plugins-${pluginId}`].createIssueModalForPostId;
    const post = getPost(state, postId);
    const currentTeam = getCurrentTeam(state);

    return {
        visible: state[`plugins-${pluginId}`].isCreateIssueModalVisible,
        post,
        currentTeam,
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    close: closeCreateIssueModal,
    create: createIssue,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(CreateIssueModal);
