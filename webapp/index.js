// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

import TeamSidebar from './components/team_sidebar';
import UserAttribute from './components/user_attribute';
import Reducer from './reducers';
import {getConnected} from './actions';
import {handleConnect, handleDisconnect} from './websocket';

class PluginClass {
    async initialize(registry, store) {
        registry.registerReducer(Reducer);

        await getConnected()(store.dispatch, store.getState);

        registry.registerBottomTeamSidebarComponent(TeamSidebar);
        registry.registerPopoverUserAttributesComponent(UserAttribute);

        registry.registerWebSocketEventHandler('custom_github_connect', handleConnect(store));
        registry.registerWebSocketEventHandler('custom_github_disconnect', handleDisconnect(store));
    }
}

global.window.plugins['github'] = new PluginClass();
