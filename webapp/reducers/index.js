import {combineReducers} from 'redux';

import ActionTypes from '../action_types';
import Constants from '../constants';

function connected(state = false, action) {
    switch(action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.connected;
    default:
        return state;
    }
}

function username(state = '', action) {
    switch(action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.github_username;
    default:
        return state;
    }
}

function settings(state = {sidebar_buttons: Constants.SETTING_BUTTONS_TEAM, daily_reminder: true, notifications: true}, action) {
    switch(action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.settings;
    default:
        return state;
    }
}

function clientId(state = '', action) {
    switch(action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.github_client_id;
    default:
        return state;
    }
}

function reviews(state = [], action) {
    switch(action.type) {
    case ActionTypes.RECEIVED_REVIEWS:
        return action.data;
    default:
        return state;
    }
}

function yourPrs(state = [], action) {
    switch(action.type) {
    case ActionTypes.RECEIVED_YOUR_PRS:
        return action.data;
    default:
        return state;
    }
}

function mentions(state = [], action) {
    switch(action.type) {
    case ActionTypes.RECEIVED_MENTIONS:
        return action.data;
    default:
        return state;
    }
}

function unreads(state = [], action) {
    switch(action.type) {
    case ActionTypes.RECEIVED_UNREADS:
        return action.data;
    default:
        return state;
    }
}

export default combineReducers({
    connected,
    username,
    settings,
    clientId,
    reviews,
    yourPrs,
    mentions,
    unreads,
});
