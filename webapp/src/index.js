// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.
import AttachCommentToIssuePostMenuAction from 'components/post_menu_actions/attach_comment_to_issue';
import AttachCommentToIssueModal from 'components/modals/attach_comment_to_issue';

import CreateIssueModal from './components/modals/create_issue';
import CreateIssuePostMenuAction from './components/post_menu_action/create_issue';
import SidebarHeader from './components/sidebar_header';
import TeamSidebar from './components/team_sidebar';
import UserAttribute from './components/user_attribute';
import SidebarRight from './components/sidebar_right';
import LinkTooltip from './components/link_tooltip';
import Reducer from './reducers';
import {getConnected, setShowRHSAction, getSettings} from './actions';
import {handleConnect, handleDisconnect, handleReconnect, handleRefresh} from './websocket';

import {id as pluginId} from './manifest';

let activityFunc;
let lastActivityTime = Number.MAX_SAFE_INTEGER;
const activityTimeout = 60 * 60 * 1000; // 1 hour

class PluginClass {
    async initialize(registry, store) {
        registry.registerReducer(Reducer);

        const {data: settings} = await getSettings(store.getState);
        await getConnected(true)(store.dispatch, store.getState);

        if (settings && settings.left_sidebar_enabled) {
            registry.registerLeftSidebarHeaderComponent(SidebarHeader);
            registry.registerBottomTeamSidebarComponent(TeamSidebar);
        }
        registry.registerPopoverUserAttributesComponent(UserAttribute);
        registry.registerRootComponent(CreateIssueModal);
        registry.registerPostDropdownMenuComponent(CreateIssuePostMenuAction);
        registry.registerRootComponent(AttachCommentToIssueModal);
        registry.registerPostDropdownMenuComponent(AttachCommentToIssuePostMenuAction);
        registry.registerLinkTooltipComponent(LinkTooltip);

        const {showRHSPlugin} = registry.registerRightHandSidebarComponent(SidebarRight, 'GitHub');
        store.dispatch(setShowRHSAction(() => store.dispatch(showRHSPlugin)));

        registry.registerWebSocketEventHandler(`custom_${pluginId}_connect`, handleConnect(store));
        registry.registerWebSocketEventHandler(`custom_${pluginId}_disconnect`, handleDisconnect(store));
        registry.registerWebSocketEventHandler(`custom_${pluginId}_refresh`, handleRefresh(store));
        registry.registerReconnectHandler(handleReconnect(store));

        activityFunc = () => {
            const now = new Date().getTime();
            if (now - lastActivityTime > activityTimeout) {
                handleReconnect(store, true)();
            }
            lastActivityTime = now;
        };

        document.addEventListener('click', activityFunc);
    }

    deinitialize() {
        document.removeEventListener('click', activityFunc);
    }
}

global.window.registerPlugin(pluginId, new PluginClass());
