// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {combineReducers} from 'redux';

import {AttachCommentToIssueModalForPostIdData, ConfigurationData, ConnectedData, CreateIssueModalData, GithubUsersData, MentionsData, PrsDetailsData, ShowRhsPluginActionData, SidebarContentData, UserSettingsData, YourReposData, Organization, RepositoriesByOrg} from '../types/github_types';

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

function configuration(state = {
    left_sidebar_enabled: true,
}, action: {type: string, data: ConnectedData | ConfigurationData}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return {
            ...state,
            ...(action.data as ConnectedData).configuration,
        };
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
    mentions: [],
} as SidebarContentData, action: {type: string, data: SidebarContentData}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_SIDEBAR_CONTENT:
        return action.data;
    default:
        return state;
    }
}

function yourRepos(state: YourReposData = {
    repos: [],
}, action: {type: string, data: YourReposData}) {
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

function githubUsers(state: Record<string, GithubUsersData | undefined> = {}, action: {type: string, data: GithubUsersData, userID: string}) {
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

const yourOrgs = (state: Organization[] = [], action:{type:string, data: Organization[]}) => {
    switch (action.type) {
    case ActionTypes.RECEIVED_ORGANIZATIONS:
        return action.data;
    default:
        return state;
    }
};

const yourReposByOrg = (state: RepositoriesByOrg[] = [], action:{type: string, data: RepositoriesByOrg[]}) => {
    switch (action.type) {
    case ActionTypes.RECEIVED_REPOSITORIES_BY_ORGANIZATION:
        return action.data;
    default:
        return state;
    }
};

export default combineReducers({
    yourOrgs,
    yourReposByOrg,
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
    isCreateIssueModalVisible,
    createIssueModal,
    attachCommentToIssueModalVisible,
    attachCommentToIssueModalForPostId,
    sidebarContent,
});
