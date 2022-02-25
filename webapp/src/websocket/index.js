// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import ActionTypes from '../action_types';
import Constants from '../constants';
import {
    getConnected,
    getReviews,
    getUnreads,
    getYourAssignments,
    getYourPrs,
    openCreateIssueModalWithoutPost,
} from '../actions';

import {id as pluginId} from '../manifest';

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
            getReviews()(store.dispatch, store.getState);
            getUnreads()(store.dispatch, store.getState);
            getYourPrs()(store.dispatch, store.getState);
            getYourAssignments()(store.dispatch, store.getState);
        }
    };
}

export function handleRefresh(store) {
    return () => {
        if (store.getState()[`plugins-${pluginId}`].connected) {
            getReviews()(store.dispatch, store.getState);
            getUnreads()(store.dispatch, store.getState);
            getYourPrs()(store.dispatch, store.getState);
            getYourAssignments()(store.dispatch, store.getState);
        }
    };
}

export function handleOpenCreateIssueModal(store) {
    return (msg) => {
        if (!msg.data) {
            return;
        }
        store.dispatch(openCreateIssueModalWithoutPost(msg.data.title, msg.data.channel_id));
    };
}
