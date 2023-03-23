import {createSelector} from 'reselect';

import {getConfig} from 'mattermost-redux/selectors/entities/general';

import {id as pluginId} from './manifest';

const getPluginState = (state) => state['plugins-' + pluginId] || {};

export const isEnabled = (state) => getPluginState(state).enabled;

export const getServerRoute = (state) => {
    const config = getConfig(state);
    let basePath = '';
    if (config && config.SiteURL) {
        basePath = new URL(config.SiteURL).pathname;
        if (basePath && basePath[basePath.length - 1] === '/') {
            basePath = basePath.substr(0, basePath.length - 1);
        }
    }

    return basePath;
};

export const getCloseOrReopenIssueModalData = createSelector(
    getPluginState,
    (pluginState) => {
        const {messageData} = pluginState.closeOrReopenIssueModal;
        return {
            visible: pluginState.isCloseOrReopenIssueModalVisible,
            messageData,
        };
    },
);

export const configuration = (state) => getPluginState(state).configuration;
