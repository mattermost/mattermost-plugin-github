// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

import TeamSidebar from './components/team_sidebar';
import Reducer from './reducers';
import {getConnected} from './actions';

class PluginClass {
    async initialize(registry, store) {
        registry.registerReducer(Reducer);

        await getConnected()(store.dispatch, store.getState);

        registry.registerBottomTeamSidebarComponent(TeamSidebar);
    }
}

global.window.plugins['github'] = new PluginClass();
