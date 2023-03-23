// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';
import {getPost} from 'mattermost-redux/selectors/entities/posts';
import {isSystemMessage} from 'mattermost-redux/utils/post_utils';

import {id as pluginId} from 'manifest';
import {openCreateIssueModalWithPost} from 'actions';

import CreateIssuePostMenuAction from './create_issue';

const mapStateToProps = (state, ownProps) => {
    const post = getPost(state, ownProps.postId);
    const systemMessage = post ? isSystemMessage(post) : true;

    return {
        show: state[`plugins-${pluginId}`].connected && !systemMessage,
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    open: openCreateIssueModalWithPost,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(CreateIssuePostMenuAction);
