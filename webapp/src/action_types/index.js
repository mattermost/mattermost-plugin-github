// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {id as pluginId} from '../manifest';

export default {
    RECEIVED_REVIEWS: pluginId + '_received_reviews',
    RECEIVED_YOUR_PRS: pluginId + '_received_your_prs',
    RECEIVED_YOUR_PRS_EXTRA_INFO: pluginId + '_received_your_prs_extra_info',
    RECEIVED_YOUR_ASSIGNMENTS: pluginId + '_received_your_assignments',
    RECEIVED_MENTIONS: pluginId + '_received_mentions',
    RECEIVED_UNREADS: pluginId + '_received_unreads',
    RECEIVED_CONNECTED: pluginId + '_received_connected',
    RECEIVED_GITHUB_USER: pluginId + '_received_github_user',
    RECEIVED_SHOW_RHS_ACTION: pluginId + '_received_rhs_action',
    UPDATE_RHS_STATE: pluginId + '_update_rhs_state',
    CLOSE_ATTACH_COMMENT_TO_ISSUE_MODAL: pluginId + '_close_attach_modal',
    OPEN_ATTACH_COMMENT_TO_ISSUE_MODAL: pluginId + '_open_attach_modal',
    RECEIVED_ATTACH_COMMENT_RESULT: pluginId + '_received_attach_comment',
};
