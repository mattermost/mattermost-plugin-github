import {GlobalState as ReduxGlobalState} from 'mattermost-redux/types/store';

import reducers from '../reducers';

export type GetStateFunc = () => GlobalState

export type GlobalState = ReduxGlobalState & {
    'plugins-github': PluginState
};

export type PluginState = ReturnType<typeof reducers>
