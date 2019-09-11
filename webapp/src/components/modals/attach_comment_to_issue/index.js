import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getPost} from 'mattermost-redux/selectors/entities/posts';

import {closeAttachCommentToIssueModal, attachCommentToIssue} from 'actions';

import AttachCommentToIssue from './attach_comment_to_issue';

const mapStateToProps = (state) => {
    const postId = state['plugins-github'].attachCommentToIssueModalForPostId;
    const post = getPost(state, postId);

    return {
        visible: state['plugins-github'].attachCommentToIssueModalVisible,
        post,
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    close: closeAttachCommentToIssueModal,
    create: attachCommentToIssue,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(AttachCommentToIssue);
