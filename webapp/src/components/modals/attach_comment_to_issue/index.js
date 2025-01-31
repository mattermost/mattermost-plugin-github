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
    const {postId, messageData} = state[`plugins-${pluginId}`].attachCommentToIssueModalForPostId;
    const currentPostId = postId || messageData?.postId;
    const post = currentPostId ? getPost(state, currentPostId) : null;

    return {
        visible: state[`plugins-${pluginId}`].attachCommentToIssueModalVisible,
        post,
        messageData,
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    close: closeAttachCommentToIssueModal,
    create: attachCommentToIssue,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(AttachCommentToIssue);
