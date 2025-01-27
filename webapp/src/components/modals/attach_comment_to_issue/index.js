// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';
import {getPost} from 'mattermost-redux/selectors/entities/posts';

import manifest from '@/manifest';

import {closeAttachCommentToIssueModal, attachCommentToIssue} from '@/actions';

import AttachCommentToIssue from './attach_comment_to_issue';

const mapStateToProps = (state) => {
    const {id: pluginId} = manifest;
    const postId = state[`plugins-${pluginId}`].attachCommentToIssueModalForPostId;
    const post = getPost(state, postId);

    return {
        visible: state[`plugins-${pluginId}`].attachCommentToIssueModalVisible,
        post,
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    close: closeAttachCommentToIssueModal,
    create: attachCommentToIssue,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(AttachCommentToIssue);
