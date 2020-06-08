import {id as pluginId} from './manifest';

const getPluginState = (state) => state['plugins-' + pluginId] || {};

export const isEnabled = (state) => getPluginState(state).enabled;

