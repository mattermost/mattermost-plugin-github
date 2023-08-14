// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import Client from '../client';
import ActionTypes from '../action_types';

import {id as pluginId} from '../manifest';

export function getConnected(reminder = false) {
    return async (dispatch) => {
        let data;
        try {
            data = await Client.getConnected(reminder);
        } catch (error) {
            return {error};
        }

        dispatch({
            type: ActionTypes.RECEIVED_CONNECTED,
            data,
        });

        return {data};
    };
}

function checkAndHandleNotConnected(data) {
    return async (dispatch) => {
        if (data && data.id === 'not_connected') {
            dispatch({
                type: ActionTypes.RECEIVED_CONNECTED,
                data: {
                    connected: false,
                    github_username: '',
                    github_client_id: '',
                    user_settings: {},
                },
            });
            return false;
        }
        return true;
    };
}

export function getReviewsDetails(prList) {
    return async (dispatch, getState) => {
        let data;
        try {
            data = await Client.getPrsDetails(prList);
        } catch (error) {
            return {error};
        }

        const connected = await checkAndHandleNotConnected(data)(dispatch, getState);
        if (!connected) {
            return {error: data};
        }

        dispatch({
            type: ActionTypes.RECEIVED_REVIEWS_DETAILS,
            data,
        });

        return {data};
    };
}

export function getRepos() {
    return async (dispatch, getState) => {
        let data;
        try {
            data = await Client.getRepositories();
        } catch (error) {
            return {error: data};
        }

        const connected = await checkAndHandleNotConnected(data)(dispatch, getState);
        if (!connected) {
            return {error: data};
        }

        dispatch({
            type: ActionTypes.RECEIVED_REPOSITORIES,
            data,
        });

        return {data};
    };
}

export function getSidebarContent() {
    return async (dispatch, getState) => {
        let data;
        try {
            data = await Client.getSidebarContent();
        } catch (error) {
            return {error};
        }

        const connected = await checkAndHandleNotConnected(data)(dispatch, getState);
        if (!connected) {
            return {error: data};
        }

        dispatch({
            type: ActionTypes.RECEIVED_SIDEBAR_CONTENT,
            data,
        });

        return {data};
    };
}

export function getYourPrsDetails(prList) {
    return async (dispatch, getState) => {
        let data;
        try {
            data = await Client.getPrsDetails(prList);
        } catch (error) {
            return {error};
        }

        const connected = await checkAndHandleNotConnected(data)(dispatch, getState);
        if (!connected) {
            return {error: data};
        }

        dispatch({
            type: ActionTypes.RECEIVED_YOUR_PRS_DETAILS,
            data,
        });

        return {data};
    };
}

export function getLabelOptions(repo) {
    return async (dispatch, getState) => {
        let data;
        try {
            data = await Client.getLabels(repo);
        } catch (error) {
            return {error};
        }

        const connected = await checkAndHandleNotConnected(data)(dispatch, getState);
        if (!connected) {
            return {error: data};
        }

        return {data};
    };
}

export function getAssigneeOptions(repo) {
    return async (dispatch, getState) => {
        let data;
        try {
            data = await Client.getAssignees(repo);
        } catch (error) {
            return {error};
        }

        const connected = await checkAndHandleNotConnected(data)(dispatch, getState);
        if (!connected) {
            return {error: data};
        }

        return {data};
    };
}

export function getMilestoneOptions(repo) {
    return async (dispatch, getState) => {
        let data;
        try {
            data = await Client.getMilestones(repo);
        } catch (error) {
            return {error};
        }

        const connected = await checkAndHandleNotConnected(data)(dispatch, getState);
        if (!connected) {
            return {error: data};
        }

        return {data};
    };
}

export function getMentions() {
    return async (dispatch, getState) => {
        let data;
        try {
            data = await Client.getMentions();
        } catch (error) {
            return {error};
        }

        const connected = await checkAndHandleNotConnected(data)(dispatch, getState);
        if (!connected) {
            return {error: data};
        }

        dispatch({
            type: ActionTypes.RECEIVED_MENTIONS,
            data,
        });

        return {data};
    };
}

const GITHUB_USER_GET_TIMEOUT_MILLISECONDS = 1000 * 60 * 60; // 1 hour

export function getGitHubUser(userID) {
    return async (dispatch, getState) => {
        if (!userID) {
            return {};
        }

        const user = getState()[`plugins-${pluginId}`].githubUsers[userID];
        if (user && user.last_try && Date.now() - user.last_try < GITHUB_USER_GET_TIMEOUT_MILLISECONDS) {
            return {};
        }

        if (user && user.username) {
            return {data: user};
        }

        let data;
        try {
            data = await Client.getGitHubUser(userID);
        } catch (error) {
            if (error.status === 404) {
                dispatch({
                    type: ActionTypes.RECEIVED_GITHUB_USER,
                    userID,
                    data: {last_try: Date.now()},
                });
            }
            return {error};
        }

        dispatch({
            type: ActionTypes.RECEIVED_GITHUB_USER,
            userID,
            data,
        });

        return {data};
    };
}

/**
 * Stores`showRHSPlugin` action returned by
 * registerRightHandSidebarComponent in plugin initialization.
 */
export function setShowRHSAction(showRHSPluginAction) {
    return {
        type: ActionTypes.RECEIVED_SHOW_RHS_ACTION,
        showRHSPluginAction,
    };
}

export function updateRhsState(rhsState) {
    return {
        type: ActionTypes.UPDATE_RHS_STATE,
        state: rhsState,
    };
}

export function openCreateIssueModal(postId) {
    return {
        type: ActionTypes.OPEN_CREATE_ISSUE_MODAL,
        data: {
            postId,
        },
    };
}

export function openCreateIssueModalWithoutPost(title, channelId) {
    return {
        type: ActionTypes.OPEN_CREATE_ISSUE_MODAL_WITHOUT_POST,
        data: {
            title,
            channelId,
        },
    };
}

export function closeCreateIssueModal() {
    return {
        type: ActionTypes.CLOSE_CREATE_ISSUE_MODAL,
    };
}

export function createIssue(payload) {
    return async (dispatch) => {
        let data;
        try {
            data = await Client.createIssue(payload);
        } catch (error) {
            return {error};
        }

        const connected = await dispatch(checkAndHandleNotConnected(data));
        if (!connected) {
            return {error: data};
        }

        return {data};
    };
}

export function openAttachCommentToIssueModal(postId) {
    return {
        type: ActionTypes.OPEN_ATTACH_COMMENT_TO_ISSUE_MODAL,
        data: {
            postId,
        },
    };
}

export function closeAttachCommentToIssueModal() {
    return {
        type: ActionTypes.CLOSE_ATTACH_COMMENT_TO_ISSUE_MODAL,
    };
}

export function attachCommentToIssue(payload) {
    return async (dispatch) => {
        let data;
        try {
            data = await Client.attachCommentToIssue(payload);
        } catch (error) {
            return {error};
        }

        const connected = await dispatch(checkAndHandleNotConnected(data));
        if (!connected) {
            return {error: data};
        }

        dispatch({
            type: ActionTypes.RECEIVED_ATTACH_COMMENT_RESULT,
            data,
        });
        return {data};
    };
}
