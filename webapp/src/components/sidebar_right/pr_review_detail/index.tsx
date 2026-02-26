// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators, Dispatch} from 'redux';

import {getSelectedPR, getPRReviewThreads as getPRReviewThreadsSelector, getPRReviewThreadsLoading, getAIAgents as getAIAgentsSelector, getThreadsGroupedByFile} from '../../../selectors';
import {clearSelectedPR, getPRReviewThreads, replyToReviewComment, toggleReaction, resolveThread, postAIAssignment, getAIAgents} from '../../../actions';

import {GlobalState} from '../../../types/store';

import PRReviewDetail from './pr_review_detail';

function mapStateToProps(state: GlobalState) {
    return {
        selectedPR: getSelectedPR(state),
        threads: getPRReviewThreadsSelector(state),
        threadsGroupedByFile: getThreadsGroupedByFile(state),
        loading: getPRReviewThreadsLoading(state),
        aiAgents: getAIAgentsSelector(state),
    };
}

function mapDispatchToProps(dispatch: Dispatch) {
    return {
        actions: bindActionCreators({
            clearSelectedPR,
            getPRReviewThreads,
            replyToReviewComment,
            toggleReaction,
            resolveThread,
            postAIAssignment,
            getAIAgents,
        }, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(PRReviewDetail);
