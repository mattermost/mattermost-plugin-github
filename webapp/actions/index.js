import Client from '../client';
import ActionTypes from '../action_types';

export function getConnected() {
    return async (dispatch, getState) => {
        let data;
        try {
            data = await Client.getConnected();
        } catch (error) {
            return {error};
        }

        store.dispatch({
            type: ActionTypes.RECEIVED_CONNECTED,
            data: data,
        });

        return {data};
    };
}

function checkAndHandleNotConnected(data) {
    return async (dispatch, getState) => {
        if (data && data.id === 'not_connected') {
            store.dispatch({
                type: ActionTypes.RECEIVED_CONNECTED,
                data: {
                    connected: false,
                    github_username: '',
                    github_client_id: '',
                },
            });
            return false;
        }
        return true;
    };
}

export function getReviews() {
    return async (dispatch, getState) => {
        let data;
        try {
            data = await Client.getReviews();
        } catch (error) {
            return {error};
        }

        let actions = [];

        let connected = await checkAndHandleNotConnected(data)(dispatch, getState);
        if (!connected) {
            return {error: data};
        }

        dispatch({
            type: ActionTypes.RECEIVED_REVIEWS,
            data,
        });

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

        let actions = [];

        let connected = await checkAndHandleNotConnected(data)(dispatch, getState);
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
