// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {DispatchFunc} from 'mattermost-redux/types/actions';

import {ClientError} from 'mattermost-redux/client/client4';

import {ApiError} from '../client/client';
import Client from '../client';

import {APIError, PrsDetailsData, ShowRhsPluginActionData} from '../types/github_types';

import {getPluginState} from '../selectors';

import {GetStateFunc} from '../types/store';

import ActionTypes from '../action_types';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const isAPIError = (data: any): data is APIError => {
    return 'status_code' in data && Boolean((data as APIError).status_code);
};

export function getConnected(reminder = false) {
    return async (dispatch: DispatchFunc) => {
        try {
            const data = await Client.getConnected(reminder);
            dispatch({
                type: ActionTypes.RECEIVED_CONNECTED,
                data,
            });

            return {data};
        } catch (e) {
            return {error: e as ClientError};
        }
    };
}

function checkAndHandleNotConnected(data: ApiError | Object) {
    return async (dispatch: DispatchFunc) => {
        if (data && 'id' in data && data.id === 'not_connected') {
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

export function getReviewsDetails(prList: PrsDetailsData[]) {
    return async (dispatch: DispatchFunc) => {
        try {
            const data = await Client.getPrsDetails(prList);

            if (isAPIError(data)) {
                await checkAndHandleNotConnected(data)(dispatch);
                return {error: data};
            }

            dispatch({
                type: ActionTypes.RECEIVED_REVIEWS_DETAILS,
                data,
            });

            return {data};
        } catch (e) {
            return {error: e as ClientError};
        }
    };
}

export function getOrgs() {
    return async (dispatch: DispatchFunc) => {
        let data;
        try {
            data = await Client.getOrganizations();
        } catch (error) {
            return {error: data};
        }

        const connected = await checkAndHandleNotConnected(data)(dispatch);
        if (!connected) {
            return {error: data};
        }

        dispatch({
            type: ActionTypes.RECEIVED_ORGANIZATIONS,
            data,
        });

        return {data};
    };
}

export function getReposByOrg(organization: string) {
    return async (dispatch: DispatchFunc) => {
        let data;
        try {
            data = await Client.getRepositoriesByOrganization(organization);
        } catch (error) {
            return {error: data};
        }

        const connected = await checkAndHandleNotConnected(data)(dispatch);
        if (!connected) {
            return {error: data};
        }

        dispatch({
            type: ActionTypes.RECEIVED_REPOSITORIES_BY_ORGANIZATION,
            data,
        });

        return {data};
    };
}

export function getRepos(channelId: string) {
    return async (dispatch: DispatchFunc) => {
        try {
            const data = await Client.getRepositories();

            if (isAPIError(data)) {
                await checkAndHandleNotConnected(data)(dispatch);
                return {error: data};
            }

            dispatch({
                type: ActionTypes.RECEIVED_REPOSITORIES,
                data,
            });

            return {data};
        } catch (e) {
            return {error: e as ClientError};
        }
    };
}

export function getSidebarContent() {
    return async (dispatch: DispatchFunc) => {
        try {
            const data = await Client.getSidebarContent();

            if (isAPIError(data)) {
                await checkAndHandleNotConnected(data)(dispatch);
                return {error: data};
            }

            dispatch({
                type: ActionTypes.RECEIVED_SIDEBAR_CONTENT,
                data,
            });

            return {data};
        } catch (e) {
            return {error: e as ClientError};
        }
    };
}

export function getYourPrsDetails(prList: {url: string, number: number}[]) {
    return async (dispatch: DispatchFunc) => {
        try {
            const data = await Client.getPrsDetails(prList);
            if (isAPIError(data)) {
                await checkAndHandleNotConnected(data)(dispatch);
                return {error: data};
            }

            dispatch({
                type: ActionTypes.RECEIVED_YOUR_PRS_DETAILS,
                data,
            });

            return {data};
        } catch (e) {
            return {error: e as ClientError};
        }
    };
}

export function getLabelOptions(repo: string) {
    return async (dispatch: DispatchFunc) => {
        try {
            const data = await Client.getLabels(repo);

            if (isAPIError(data)) {
                await checkAndHandleNotConnected(data)(dispatch);
                return {error: data};
            }

            return {data};
        } catch (e) {
            return {error: e as ClientError};
        }
    };
}

export function getAssigneeOptions(repo: string) {
    return async (dispatch: DispatchFunc) => {
        try {
            const data = await Client.getAssignees(repo);

            if (isAPIError(data)) {
                await checkAndHandleNotConnected(data)(dispatch);
                return {error: data};
            }

            return {data};
        } catch (e) {
            return {error: e as ClientError};
        }
    };
}

export function getMilestoneOptions(repo: string) {
    return async (dispatch: DispatchFunc) => {
        try {
            const data = await Client.getMilestones(repo);

            if (isAPIError(data)) {
                await checkAndHandleNotConnected(data)(dispatch);
                return {error: data};
            }

            return {data};
        } catch (e) {
            return {error: e as ClientError};
        }
    };
}

export function getMentions() {
    return async (dispatch: DispatchFunc) => {
        try {
            const data = await Client.getMentions();

            if (isAPIError(data)) {
                await checkAndHandleNotConnected(data)(dispatch);
                return {error: data};
            }

            dispatch({
                type: ActionTypes.RECEIVED_MENTIONS,
                data,
            });

            return {data};
        } catch (e) {
            return {error: e as ClientError};
        }
    };
}

const GITHUB_USER_GET_TIMEOUT_MILLISECONDS = 1000 * 60 * 60; // 1 hour

export function getGitHubUser(userID: string) {
    return async (dispatch: DispatchFunc, getState: GetStateFunc) => {
        if (!userID) {
            return {data: false};
        }

        const user = getPluginState(getState()).githubUsers[userID];
        if (user && user.last_try && Date.now() - user.last_try < GITHUB_USER_GET_TIMEOUT_MILLISECONDS) {
            return {data: false};
        }

        if (user && user.username) {
            return {data: user};
        }

        try {
            const data = await Client.getGitHubUser(userID);

            if (isAPIError(data)) {
                if (data.status_code === 404) {
                    dispatch({
                        type: ActionTypes.RECEIVED_GITHUB_USER,
                        userID,
                        data: {last_try: Date.now()},
                    });
                }

                return {error: data};
            }

            dispatch({
                type: ActionTypes.RECEIVED_GITHUB_USER,
                userID,
                data,
            });

            return {data};
        } catch (e: unknown) {
            if (isAPIError(e) && e.status_code === 404) {
                dispatch({
                    type: ActionTypes.RECEIVED_GITHUB_USER,
                    userID,
                    data: {last_try: Date.now()},
                });
            }
            return {error: e as ClientError};
        }
    };
}

/**
 * Stores`showRHSPlugin` action returned by
 * registerRightHandSidebarComponent in plugin initialization.
 */
export function setShowRHSAction(showRHSPluginAction: ShowRhsPluginActionData) {
    return {
        type: ActionTypes.RECEIVED_SHOW_RHS_ACTION,
        showRHSPluginAction,
    };
}

export function updateRhsState(rhsState: string) {
    return {
        type: ActionTypes.UPDATE_RHS_STATE,
        state: rhsState,
    };
}

export function openCreateIssueModal(postId: string) {
    return {
        type: ActionTypes.OPEN_CREATE_ISSUE_MODAL,
        data: {
            postId,
        },
    };
}

export function openCreateIssueModalWithoutPost(title: string, channelId: string) {
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

export function createIssue(payload: CreateIssuePayload) {
    return async (dispatch: DispatchFunc) => {
        try {
            const data = await Client.createIssue(payload);

            if (isAPIError(data)) {
                await checkAndHandleNotConnected(data)(dispatch);
                return {error: data};
            }

            return {data};
        } catch (e) {
            return {error: e as ClientError};
        }
    };
}

export function openAttachCommentToIssueModal(postId: string) {
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

export function attachCommentToIssue(payload: AttachCommentToIssuePayload) {
    return async (dispatch: DispatchFunc) => {
        try {
            const data = await Client.attachCommentToIssue(payload);

            if (isAPIError(data)) {
                await checkAndHandleNotConnected(data)(dispatch);
                return {error: data};
            }

            dispatch({
                type: ActionTypes.RECEIVED_ATTACH_COMMENT_RESULT,
                data,
            });
            return {data};
        } catch (e) {
            return {error: e as ClientError};
        }
    };
}
