// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';
import {getPost} from 'mattermost-redux/selectors/entities/posts';

import manifest from '@/manifest';
import {closeCreateIssueModal, createIssue} from '@/actions';

import CreateIssueModal from './create_issue';

const mapStateToProps = (state) => {
    const {id: pluginId} = manifest;
    const {postId, title, channelId} = state[`plugins-${pluginId}`].createIssueModal;
    const post = (postId) ? getPost(state, postId) : null;

    return {
        visible: state[`plugins-${pluginId}`].isCreateIssueModalVisible,
        post,
        title,
        channelId,
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    close: closeCreateIssueModal,
    create: createIssue,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(CreateIssueModal);
