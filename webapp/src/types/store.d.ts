// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {GlobalState as ReduxGlobalState} from 'mattermost-redux/types/store';

import reducers from '../reducers';

export type GetStateFunc = () => GlobalState

export type GlobalState = ReduxGlobalState & {
    'plugins-github': PluginState
};

export type PluginState = ReturnType<typeof reducers>
