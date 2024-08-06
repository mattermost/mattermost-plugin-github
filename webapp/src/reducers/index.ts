// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {combineReducers} from 'redux';

import {AttachCommentToIssueModalForPostIdData, CloseOrReopenIssueModalData, ConfigurationData, ConnectedData, CreateIssueModalData, GithubUsersData, MentionsData, MessageData, PrsDetailsData, ShowRhsPluginActionData, SidebarContentData, UserSettingsData, YourReposData} from '../types/github_types';

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

function enterpriseURL(state = '', action: {type: string, data: ConnectedData}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        if (action.data && action.data.enterprise_base_url) {
            return action.data.enterprise_base_url;
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
        return action.data.github_username;
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
        return action.data.github_client_id;
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

function githubUsers(state: Record<string, GithubUsersData> = {}, action: {type: string, data: GithubUsersData, userID: string}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_GITHUB_USER: {
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

const isCreateOrUpdateIssueModalVisible = (state = false, action: {type: string}) => {
    switch (action.type) {
    case ActionTypes.OPEN_CREATE_ISSUE_MODAL_WITH_POST:
    case ActionTypes.OPEN_CREATE_OR_UPDATE_ISSUE_MODAL:
        return true;
    case ActionTypes.CLOSE_CREATE_OR_UPDATE_ISSUE_MODAL:
        return false;
    default:
        return state;
    }
};

const isCloseOrReopenIssueModalVisible = (state = false, action: {type: string}) => {
    switch (action.type) {
    case ActionTypes.OPEN_CLOSE_OR_REOPEN_ISSUE_MODAL:
        return true;
    case ActionTypes.CLOSE_CLOSE_OR_REOPEN_ISSUE_MODAL:
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

const createOrUpdateIssueModal = (state = {} as CreateIssueModalData, action: {type: string, data: CreateIssueModalData}) => {
    switch (action.type) {
    case ActionTypes.OPEN_CREATE_ISSUE_MODAL_WITH_POST:
    case ActionTypes.OPEN_CREATE_OR_UPDATE_ISSUE_MODAL:
        return {
            ...state,
            postId: action.data.postId,
            messageData: action.data.messageData,
        };
    case ActionTypes.CLOSE_CREATE_OR_UPDATE_ISSUE_MODAL:
        return {};
    default:
        return state;
    }
};

const closeOrReopenIssueModal = (state = {}, action: {type: string, data: CloseOrReopenIssueModalData}) => {
    switch (action.type) {
    case ActionTypes.OPEN_CLOSE_OR_REOPEN_ISSUE_MODAL:
        return {
            ...state,
            messageData: action.data.messageData,
        };
    case ActionTypes.CLOSE_CLOSE_OR_REOPEN_ISSUE_MODAL:
        return {};
    default:
        return state;
    }
};

const attachCommentToIssueModalForPostId = (state = {}, action: {type: string, data: AttachCommentToIssueModalForPostIdData}) => {
    switch (action.type) {
    case ActionTypes.OPEN_ATTACH_COMMENT_TO_ISSUE_MODAL:
        return {
            ...state,
            postId: action.data.postId,
            messageData: action.data.messageData,
        };
    case ActionTypes.CLOSE_ATTACH_COMMENT_TO_ISSUE_MODAL:
        return '';
    default:
        return state;
    }
};

export default combineReducers({
    connected,
    enterpriseURL,
    organizations,
    username,
    userSettings,
    configuration,
    clientId,
    reviewDetails,
    yourRepos,
    yourPrDetails,
    mentions,
    githubUsers,
    rhsPluginAction,
    rhsState,
    isCreateOrUpdateIssueModalVisible,
    isCloseOrReopenIssueModalVisible,
    createOrUpdateIssueModal,
    closeOrReopenIssueModal,
    attachCommentToIssueModalVisible,
    attachCommentToIssueModalForPostId,
    sidebarContent,
});
