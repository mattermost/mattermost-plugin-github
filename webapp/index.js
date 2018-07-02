// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

import TeamSidebar from './components/team_sidebar';
import UserAttribute from './components/user_attribute';
import Reducer from './reducers';
import {getConnected} from './actions';
import {handleConnect, handleDisconnect} from './websocket';

let activityFunc;
let lastActivityTime = 0;
const activityTimeout = 60*60*1000; // 1 hour

class PluginClass {
    async initialize(registry, store) {
        registry.registerReducer(Reducer);

        await getConnected()(store.dispatch, store.getState);

        registry.registerBottomTeamSidebarComponent(TeamSidebar);
        registry.registerPopoverUserAttributesComponent(UserAttribute);

        registry.registerWebSocketEventHandler('custom_github_connect', handleConnect(store));
        registry.registerWebSocketEventHandler('custom_github_disconnect', handleDisconnect(store));

        activityFunc = (store) => {
            const now = new Date().getTime();
            if (now - lastActivityTime > activityTimeout) {
                getConnected()(store.dispatch, store.getState);
            }
            lastActivityTime = now;
        };

        document.addEventListener('click', activityFunc);
    }

    deinitialize() {
        console.log('deinit');
        document.removeEventListener('click', activityFunc);
    }
}

global.window.plugins['github'] = new PluginClass();
