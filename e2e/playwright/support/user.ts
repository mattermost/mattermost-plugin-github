// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// preferences any new plain user should have to avoid tours appearing
export const preferencesForUser = (userId: string) => {
    return [
        {
            user_id: userId,
            category: 'playbook_edit',
            name: userId,
            value: '999',
        },
        {
            user_id: userId,
            category: 'tutorial_pb_run_details',
            name: userId,
            value: '999',
        },
        {
            user_id: userId,
            category: 'crt_thread_pane_step',
            name: userId,
            value: '999',
        },
        {
            user_id: userId,
            category: 'playbook_preview',
            name: userId,
            value: '999',
        },
        {
            user_id: userId,
            category: 'tutorial_step',
            name: userId,
            value: '999',
        },
        {
            user_id: userId,
            category: 'crt_tutorial_triggered',
            name: userId,
            value: '999',
        },
        {
            user_id: userId,
            category: 'crt_thread_pane_step',
            name: userId,
            value: '999',
        },
        {
            user_id: userId,
            category: 'actions_menu',
            name: 'actions_menu_tutorial_state',
            value: '{"actions_menu_modal_viewed":true}',
        },
        {
            user_id: userId,
            category: 'insights',
            name: 'insights_tutorial_state',
            value: '{"insights_modal_viewed":true}',
        },
        {
            user_id: userId,
            category: 'drafts',
            name: 'drafts_tour_tip_showed',
            value: '{"drafts_tour_tip_showed":true}',
        },
    ];
};
