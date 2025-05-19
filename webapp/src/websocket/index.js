// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import ActionTypes from '../action_types';
import Constants from '../constants';
import {
    getConnected,
    openCreateOrUpdateIssueModal,
    openCreateCommentOnIssueModal,
    openCloseOrReopenIssueModal,
    getSidebarContent,
} from '../actions';

import manifest from '../manifest';

let timeoutId;
const RECONNECT_JITTER_MAX_TIME_IN_SEC = 10;
export function handleConnect(store) {
    return (msg) => {
        if (!msg.data) {
            return;
        }

        store.dispatch({
            type: ActionTypes.RECEIVED_CONNECTED,
            data: {
                ...msg.data,
                user_settings: {
                    sidebar_buttons: Constants.SETTING_BUTTONS_TEAM,
                    daily_reminder: true,
                    ...msg.data.user_settings,
                },
            },
        });
    };
}

export function handleDisconnect(store) {
    return () => {
        store.dispatch({
            type: ActionTypes.RECEIVED_CONNECTED,
            data: {
                connected: false,
                github_username: '',
                github_client_id: '',
                user_settings: {},
                configuration: {},
            },
        });
    };
}

export function handleConfigurationUpdate(store) {
    return (msg) => {
        if (!msg.data) {
            return;
        }

        store.dispatch({
            type: ActionTypes.RECEIVED_CONFIGURATION,
            data: msg.data,
        });
    };
}

export function handleReconnect(store, reminder = false) {
    return async () => {
        const {data} = await getConnected(reminder)(store.dispatch, store.getState);
        if (data && data.connected) {
            if (typeof timeoutId === 'number') {
                clearTimeout(timeoutId);
            }

            const rand = Math.floor(Math.random() * RECONNECT_JITTER_MAX_TIME_IN_SEC) + 1;
            timeoutId = setTimeout(() => {
                getSidebarContent()(store.dispatch, store.getState);
                timeoutId = undefined; //eslint-disable-line no-undefined
            }, rand * 1000);
        }
    };
}

export function handleRefresh(store) {
    return (msg) => {
        if (store.getState()[`plugins-${manifest.id}`].connected) {
            const {data} = msg;

            store.dispatch({
                type: ActionTypes.RECEIVED_SIDEBAR_CONTENT,
                data,
            });
        }
    };
}

export function handleOpenCreateOrUpdateIssueModal(store) {
    return (msg) => {
        if (!msg.data) {
            return;
        }
        store.dispatch(openCreateOrUpdateIssueModal(msg.data));
    };
}

export function handleOpenEditIssueModal(store) {
    return (msg) => {
        if (!msg.data) {
            return;
        }
        const editIssueModalData = {
            repo_owner: msg.data.repo_owner,
            repo_name: msg.data.repo_name,
            issue_number: msg.data.issue_number,
            postId: msg.data.postId,
            status: msg.data.status,
            channel_id: msg.data.channel_id,
        };
        store.dispatch(openCreateOrUpdateIssueModal(editIssueModalData));
    };
}

export function handleOpenCreateCommentOnIssueModal(store) {
    return (msg) => {
        if (!msg.data) {
            return;
        }
        const commmentModalData = {
            repo_owner: msg.data.repo_owner,
            repo_name: msg.data.repo_name,
            issue_number: msg.data.issue_number,
            postId: msg.data.postId,
            status: msg.data.status,
            channel_id: msg.data.channel_id,
        };
        store.dispatch(openCreateCommentOnIssueModal(commmentModalData));
    };
}

export function handleOpenCloseOrReopenIssueModal(store) {
    return (msg) => {
        if (!msg.data) {
            return;
        }
        const statusModalData = {
            repo_owner: msg.data.repo_owner,
            repo_name: msg.data.repo_name,
            issue_number: msg.data.issue_number,
            postId: msg.data.postId,
            status: msg.data.status,
            channel_id: msg.data.channel_id,
        };
        store.dispatch(openCloseOrReopenIssueModal(statusModalData));
    };
}
