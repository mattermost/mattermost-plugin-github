// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {combineReducers} from 'redux';

import {AttachCommentToIssueModalForPostIdData, ConfigurationData, ConnectedData, CreateIssueModalData, ForgejoUsersData, MentionsData, PrsDetailsData, ShowRhsPluginActionData, SidebarContentData, UserSettingsData, YourReposData} from '../types/forgejo_types';

import ActionTypes from '../action_types';
import Constants from '../constants';

function connected(state = false, action: {type: string, data: ConnectedData}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.connected;
    default:
        return state;
    }
}

function baseURL(state = '', action: {type: string, data: ConnectedData}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        if (action.data && action.data.base_url) {
            return action.data.base_url;
        }
        return '';
    default:
        return state;
    }
}

function organizations(state: string[] = [], action: {type: string, data: ConnectedData}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        if (action.data && action.data.organizations) {
            return action.data.organizations;
        }
        return [];
    default:
        return state;
    }
}

function username(state = '', action: {type: string, data: ConnectedData}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.forgejo_username;
    default:
        return state;
    }
}

function userSettings(state = {
    sidebar_buttons: Constants.SETTING_BUTTONS_TEAM,
    daily_reminder: true,
    notifications: true,
} as UserSettingsData, action: {type: string, data: ConnectedData}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.user_settings;
    default:
        return state;
    }
}

function configuration(state = true, action: {type: string, data: ConnectedData | ConfigurationData}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return (action.data as ConnectedData).configuration;
    case ActionTypes.RECEIVED_CONFIGURATION:
        return action.data as ConfigurationData;
    default:
        return state;
    }
}

function clientId(state = '', action: {type: string, data: ConnectedData}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.forgejo_client_id;
    default:
        return state;
    }
}

function reviewDetails(state: PrsDetailsData[] = [], action: {type: string, data: PrsDetailsData[]}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_REVIEWS_DETAILS:
        return action.data;
    default:
        return state;
    }
}

function sidebarContent(state = {
    reviews: [],
    assignments: [],
    prs: [],
    unreads: [],
} as SidebarContentData, action: {type: string, data: SidebarContentData}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_SIDEBAR_CONTENT:
        return action.data;
    default:
        return state;
    }
}

function yourRepos(state: YourReposData[] = [], action: {type: string, data: YourReposData[]}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_REPOSITORIES:
        return action.data;
    default:
        return state;
    }
}

function yourPrDetails(state: PrsDetailsData[] = [], action: {type: string, data: PrsDetailsData[]}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_YOUR_PRS_DETAILS:
        return action.data;
    default:
        return state;
    }
}

function mentions(state: MentionsData[] = [], action: {type: string, data: MentionsData[]}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_MENTIONS:
        return action.data;
    default:
        return state;
    }
}

function forgejoUsers(state: Record<string, ForgejoUsersData> = {}, action: {type: string, data: ForgejoUsersData, userID: string}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_FORGEJO_USER: {
        const nextState = {...state};
        nextState[action.userID] = action.data;
        return nextState;
    }
    default:
        return state;
    }
}

function rhsPluginAction(state = null, action: {type: string, showRHSPluginAction: ShowRhsPluginActionData}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_SHOW_RHS_ACTION:
        return action.showRHSPluginAction;
    default:
        return state;
    }
}

function rhsState(state = null, action: {type: string, state: string}) {
    switch (action.type) {
    case ActionTypes.UPDATE_RHS_STATE:
        return action.state;
    default:
        return state;
    }
}

const isCreateIssueModalVisible = (state = false, action: {type: string}) => {
    switch (action.type) {
    case ActionTypes.OPEN_CREATE_ISSUE_MODAL:
    case ActionTypes.OPEN_CREATE_ISSUE_MODAL_WITHOUT_POST:
        return true;
    case ActionTypes.CLOSE_CREATE_ISSUE_MODAL:
        return false;
    default:
        return state;
    }
};

const attachCommentToIssueModalVisible = (state = false, action: {type: string}) => {
    switch (action.type) {
    case ActionTypes.OPEN_ATTACH_COMMENT_TO_ISSUE_MODAL:
        return true;
    case ActionTypes.CLOSE_ATTACH_COMMENT_TO_ISSUE_MODAL:
        return false;
    default:
        return state;
    }
};

const createIssueModal = (state = {} as CreateIssueModalData, action: {type: string, data: CreateIssueModalData}) => {
    switch (action.type) {
    case ActionTypes.OPEN_CREATE_ISSUE_MODAL:
    case ActionTypes.OPEN_CREATE_ISSUE_MODAL_WITHOUT_POST:
        return {
            ...state,
            postId: action.data.postId,
            title: action.data.title,
            channelId: action.data.channelId,
        };
    case ActionTypes.CLOSE_CREATE_ISSUE_MODAL:
        return {};
    default:
        return state;
    }
};

const attachCommentToIssueModalForPostId = (state = '', action: {type: string, data: AttachCommentToIssueModalForPostIdData}) => {
    switch (action.type) {
    case ActionTypes.OPEN_ATTACH_COMMENT_TO_ISSUE_MODAL:
        return action.data.postId;
    case ActionTypes.CLOSE_ATTACH_COMMENT_TO_ISSUE_MODAL:
        return '';
    default:
        return state;
    }
};

export default combineReducers({
    connected,
    baseURL,
    organizations,
    username,
    userSettings,
    configuration,
    clientId,
    reviewDetails,
    yourRepos,
    yourPrDetails,
    mentions,
    forgejoUsers,
    rhsPluginAction,
    rhsState,
    isCreateIssueModalVisible,
    createIssueModal,
    attachCommentToIssueModalVisible,
    attachCommentToIssueModalForPostId,
    sidebarContent,
});
