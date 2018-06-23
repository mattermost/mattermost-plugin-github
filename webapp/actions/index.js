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

export function getReviews() {
    return async (dispatch, getState) => {
        let data;
        try {
            data = await Client.getReviews();
        } catch (error) {
            return {error};
        }

        let actions = [];

        let connected = true;
        if (data.id === 'not_connected') {
            store.dispatch({
                type: ActionTypes.RECEIVED_CONNECTED,
                data: {
                    connected: false,
                    github_username: '',
                },
            });
            return {data};
        }

        dispatch({
            type: ActionTypes.RECEIVED_REVIEWS,
            data,
        });

        return {data};
    };
}
