// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.
import AttachCommentToIssuePostMenuAction from 'components/post_menu_actions/attach_comment_to_issue';
import AttachCommentToIssueModal from 'components/modals/attach_comment_to_issue';

import CreateOrUpdateIssueModal from './components/modals/create_update_issue';
import CloseOrReopenIssueModal from './components/modals/close_reopen_issue';
import CreateIssuePostMenuAction from './components/post_menu_action/create_issue';
import SidebarHeader from './components/sidebar_header';
import TeamSidebar from './components/team_sidebar';
import UserAttribute from './components/user_attribute';
import SidebarRight from './components/sidebar_right';
import LinkTooltip from './components/link_tooltip';
import GithubIssue from './components/github_issue';
import Reducer from './reducers';
import Client from './client';
import {getConnected, setShowRHSAction} from './actions';
import {handleConnect, handleDisconnect, handleConfigurationUpdate, handleOpenCreateOrUpdateIssueModal, handleOpenCreateCommentOnIssueModal, handleOpenCloseOrReopenIssueModal, handleReconnect, handleRefresh} from './websocket';
import {getServerRoute} from './selectors';
import {id as pluginId} from './manifest';

let activityFunc;
let lastActivityTime = Number.MAX_SAFE_INTEGER;
const activityTimeout = 60 * 60 * 1000; // 1 hour

class PluginClass {
    async initialize(registry, store) {
        registry.registerReducer(Reducer);
        Client.setServerRoute(getServerRoute(store.getState()));

        await getConnected(true)(store.dispatch, store.getState);

        registry.registerLeftSidebarHeaderComponent(SidebarHeader);
        registry.registerBottomTeamSidebarComponent(TeamSidebar);
        registry.registerPopoverUserAttributesComponent(UserAttribute);
        registry.registerRootComponent(CreateOrUpdateIssueModal);
        registry.registerRootComponent(CloseOrReopenIssueModal);
        registry.registerPostDropdownMenuComponent(CreateIssuePostMenuAction);
        registry.registerRootComponent(AttachCommentToIssueModal);
        registry.registerPostDropdownMenuComponent(AttachCommentToIssuePostMenuAction);
        registry.registerLinkTooltipComponent(LinkTooltip);

        const {showRHSPlugin} = registry.registerRightHandSidebarComponent(SidebarRight, 'GitHub');
        store.dispatch(setShowRHSAction(() => store.dispatch(showRHSPlugin)));

        registry.registerWebSocketEventHandler(`custom_${pluginId}_connect`, handleConnect(store));
        registry.registerWebSocketEventHandler(`custom_${pluginId}_disconnect`, handleDisconnect(store));
        registry.registerWebSocketEventHandler(`custom_${pluginId}_config_update`, handleConfigurationUpdate(store));
        registry.registerWebSocketEventHandler(`custom_${pluginId}_refresh`, handleRefresh(store));
        registry.registerWebSocketEventHandler(`custom_${pluginId}_createOrUpdateIssue`, handleOpenCreateOrUpdateIssueModal(store));
        registry.registerWebSocketEventHandler(`custom_${pluginId}_attachCommentToIssue`, handleOpenCreateCommentOnIssueModal(store));
        registry.registerWebSocketEventHandler(`custom_${pluginId}_closeOrReopenIssue`, handleOpenCloseOrReopenIssueModal(store));
        registry.registerPostTypeComponent('custom_git_issue', GithubIssue);

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
