import ActionTypes from '../action_types';
import {getConnected, getReviews, getUnreads} from '../actions';

export function handleConnect(store) {
    return (msg) => {
        if (!msg.data) {
            return;
        }

        store.dispatch({
            type: ActionTypes.RECEIVED_CONNECTED,
            data: msg.data,
        });
    }
}

export function handleDisconnect(store) {
    return () => {
        store.dispatch({
            type: ActionTypes.RECEIVED_CONNECTED,
            data: {
                connected: false,
                github_username: '',
                github_client_id: '',
            }
        });
    }
}

export function handleReconnect(store) {
    return async () => {
        const {data} = await getConnected()(store.dispatch, store.getState);
        if (data && data.connected) {
            getReviews()(store.dispatch, store.getState);
            getUnreads()(store.dispatch, store.getState);
        }
    }
}