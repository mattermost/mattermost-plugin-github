// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {id as pluginId} from '../manifest';

export default {
    RECEIVED_REPOSITORIES: pluginId + '_received_repositories',
    RECEIVED_ORGANIZATIONs: pluginId + '_received_organizations',
    RECEIVED_REPOSITORIES_BY_ORGANIZATION: pluginId + '_received_repositories_by_organization',
    RECEIVED_REVIEWS: pluginId + '_received_reviews',
    RECEIVED_REVIEWS_DETAILS: pluginId + '_received_reviews_details',
    RECEIVED_YOUR_PRS: pluginId + '_received_your_prs',
    RECEIVED_YOUR_PRS_DETAILS: pluginId + '_received_your_prs_details',
    RECEIVED_YOUR_ASSIGNMENTS: pluginId + '_received_your_assignments',
    RECEIVED_MENTIONS: pluginId + '_received_mentions',
    RECEIVED_UNREADS: pluginId + '_received_unreads',
    RECEIVED_CONNECTED: pluginId + '_received_connected',
    RECEIVED_CONFIGURATION: pluginId + '_received_configuration',
    RECEIVED_GITHUB_USER: pluginId + '_received_github_user',
    RECEIVED_SHOW_RHS_ACTION: pluginId + '_received_rhs_action',
    UPDATE_RHS_STATE: pluginId + '_update_rhs_state',
    CLOSE_CREATE_ISSUE_MODAL: pluginId + '_close_create_modal',
    OPEN_CREATE_ISSUE_MODAL: pluginId + '_open_create_modal',
    OPEN_CREATE_ISSUE_MODAL_WITHOUT_POST: pluginId + '_open_create_modal_without_post',
    CLOSE_ATTACH_COMMENT_TO_ISSUE_MODAL: pluginId + '_close_attach_modal',
    OPEN_ATTACH_COMMENT_TO_ISSUE_MODAL: pluginId + '_open_attach_modal',
    RECEIVED_ATTACH_COMMENT_RESULT: pluginId + '_received_attach_comment',
};
