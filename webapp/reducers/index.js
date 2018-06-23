const {combineReducers} = window.redux;

import ActionTypes from '../action_types'

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

function reviews(state = [], action) {
    switch(action.type) {
    case ActionTypes.RECEIVED_REVIEWS:
        return action.data;
    default:
        return state;
    }
}

export default combineReducers({
    connected,
    username,
    reviews,
});