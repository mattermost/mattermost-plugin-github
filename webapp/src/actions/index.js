import Client from '../client';
import ActionTypes from '../action_types';

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
                    settings: {},
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

        const connected = await checkAndHandleNotConnected(data)(dispatch, getState);
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

export function getYourPrs() {
    return async (dispatch, getState) => {
        let data;
        try {
            data = await Client.getYourPrs();
        } catch (error) {
            return {error};
        }

        const connected = await checkAndHandleNotConnected(data)(dispatch, getState);
        if (!connected) {
            return {error: data};
        }

        dispatch({
            type: ActionTypes.RECEIVED_YOUR_PRS,
            data,
        });

        return {data};
    };
}

export function getYourAssignments() {
    return async (dispatch, getState) => {
        let data;
        try {
            data = await Client.getYourAssignments();
        } catch (error) {
            return {error};
        }

        const connected = await checkAndHandleNotConnected(data)(dispatch, getState);
        if (!connected) {
            return {error: data};
        }

        dispatch({
            type: ActionTypes.RECEIVED_YOUR_ASSIGNMENTS,
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

export function getUnreads() {
    return async (dispatch, getState) => {
        let data;
        try {
            data = await Client.getUnreads();
        } catch (error) {
            return {error};
        }

        const connected = await checkAndHandleNotConnected(data)(dispatch, getState);
        if (!connected) {
            return {error: data};
        }

        dispatch({
            type: ActionTypes.RECEIVED_UNREADS,
            data,
        });

        return {data};
    };
}
