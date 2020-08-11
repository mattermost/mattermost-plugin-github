// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {combineReducers} from 'redux';

import ActionTypes from '../action_types';
import Constants from '../constants';

function connected(state = false, action) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.connected;
    default:
        return state;
    }
}

function enterpriseURL(state = '', action) {
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

function organization(state = '', action) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        if (action.data && action.data.organization) {
            return action.data.organization;
        }
        return '';
    default:
        return state;
    }
}

function username(state = '', action) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.github_username;
    default:
        return state;
    }
}

function settings(state = {sidebar_buttons: Constants.SETTING_BUTTONS_TEAM, daily_reminder: true, notifications: true}, action) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.settings;
    default:
        return state;
    }
}

function clientId(state = '', action) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.github_client_id;
    default:
        return state;
    }
}

function reviews(state = [], action) {
    switch (action.type) {
    case ActionTypes.RECEIVED_REVIEWS:
        return action.data;
    default:
        return state;
    }
}

function reviewsDetails(state = [], action) {
    switch (action.type) {
    case ActionTypes.RECEIVED_REVIEWS_DETAILS:
        return action.data;
    default:
        return state;
    }
}

function yourPrs(state = [], action) {
    switch (action.type) {
    case ActionTypes.RECEIVED_YOUR_PRS:
        return action.data;
    default:
        return state;
    }
}

function yourRepos(state = [], action) {
    switch (action.type) {
    case ActionTypes.RECEIVED_REPOSITORIES:
        return action.data;
    default:
        return state;
    }
}

function yourPrsDetails(state = [], action) {
    switch (action.type) {
    case ActionTypes.RECEIVED_YOUR_PRS_DETAILS:
        return action.data;
    default:
        return state;
    }
}

function yourAssignments(state = [], action) {
    switch (action.type) {
    case ActionTypes.RECEIVED_YOUR_ASSIGNMENTS:
        return action.data;
    default:
        return state;
    }
}

function mentions(state = [], action) {
    switch (action.type) {
    case ActionTypes.RECEIVED_MENTIONS:
        return action.data;
    default:
        return state;
    }
}

function unreads(state = [], action) {
    switch (action.type) {
    case ActionTypes.RECEIVED_UNREADS:
        return action.data;
    default:
        return state;
    }
}

function githubUsers(state = {}, action) {
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

function rhsPluginAction(state = null, action) {
    switch (action.type) {
    case ActionTypes.RECEIVED_SHOW_RHS_ACTION:
        return action.showRHSPluginAction;
    default:
        return state;
    }
}

function rhsState(state = null, action) {
    switch (action.type) {
    case ActionTypes.UPDATE_RHS_STATE:
        return action.state;
    default:
        return state;
    }
}

const isCreateIssueModalVisible = (state = false, action) => {
    switch (action.type) {
    case ActionTypes.OPEN_CREATE_ISSUE_MODAL:
        return true;
    case ActionTypes.CLOSE_CREATE_ISSUE_MODAL:
        return false;
    default:
        return state;
    }
};

const attachCommentToIssueModalVisible = (state = false, action) => {
    switch (action.type) {
    case ActionTypes.OPEN_ATTACH_COMMENT_TO_ISSUE_MODAL:
        return true;
    case ActionTypes.CLOSE_ATTACH_COMMENT_TO_ISSUE_MODAL:
        return false;
    default:
        return state;
    }
};

const createIssueModalForPostId = (state = '', action) => {
    switch (action.type) {
    case ActionTypes.OPEN_CREATE_ISSUE_MODAL:
        return action.data.postId;
    case ActionTypes.CLOSE_CREATE_ISSUE_MODAL:
        return '';
    default:
        return state;
    }
};

const attachCommentToIssueModalForPostId = (state = '', action) => {
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
    enterpriseURL,
    organization,
    username,
    settings,
    clientId,
    reviews,
    reviewsDetails,
    yourPrs,
    yourRepos,
    yourPrsDetails,
    yourAssignments,
    mentions,
    unreads,
    githubUsers,
    rhsPluginAction,
    rhsState,
    isCreateIssueModalVisible,
    createIssueModalForPostId,
    attachCommentToIssueModalVisible,
    attachCommentToIssueModalForPostId,
});
