// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {getPost} from 'mattermost-redux/selectors/entities/posts';
import {isSystemMessage} from 'mattermost-redux/utils/post_utils';

import AttachCommentToIssuePostMenuAction from '@/components/post_menu_actions/attach_comment_to_issue';
import AttachCommentToIssueModal from '@/components/modals/attach_comment_to_issue';

import {getConnected, openAttachCommentToIssueModal, openCreateIssueModal, setShowRHSAction} from '@/actions';

import CreateIssueModal from './components/modals/create_issue';
import CreateIssuePostMenuAction from './components/post_menu_action/create_issue';
import SidebarHeader from './components/sidebar_header';
import TeamSidebar from './components/team_sidebar';
import UserAttribute from './components/user_attribute';
import SidebarRight from './components/sidebar_right';
import LinkTooltip from './components/link_tooltip';
import LinkEmbedPreview from './components/link_embed_preview';
import Reducer from './reducers';
import Client from './client';

import {handleConnect, handleDisconnect, handleConfigurationUpdate, handleOpenCreateIssueModal, handleReconnect, handleRefresh} from './websocket';
import {getServerRoute} from './selectors';
import manifest from './manifest';
import {isUrlCanPreview} from './utils/github_utils';

let activityFunc;
let lastActivityTime = Number.MAX_SAFE_INTEGER;
const activityTimeout = 60 * 60 * 1000; // 1 hour
const {id: pluginId} = manifest;

class PluginClass {
    async initialize(registry, store) {
        registry.registerReducer(Reducer);
        Client.setServerRoute(getServerRoute(store.getState()));

        await getConnected(true)(store.dispatch, store.getState);

        registry.registerLeftSidebarHeaderComponent(SidebarHeader);
        registry.registerBottomTeamSidebarComponent(TeamSidebar);
        registry.registerPopoverUserAttributesComponent(UserAttribute);
        registry.registerRootComponent(CreateIssueModal);
        registry.registerPostDropdownMenuAction({
            text: CreateIssuePostMenuAction,
            action: (postId) => {
                store.dispatch(openCreateIssueModal(postId));
            },
            filter: (postId) => {
                const state = store.getState();
                const post = getPost(state, postId);
                const systemMessage = post ? isSystemMessage(post) : true;

                return state[`plugins-${manifest.id}`].connected && !systemMessage;
            },
        });
        registry.registerRootComponent(AttachCommentToIssueModal);
        registry.registerPostDropdownMenuAction({
            text: AttachCommentToIssuePostMenuAction,
            action: (postId) => {
                store.dispatch(openAttachCommentToIssueModal(postId));
            },
            filter: (postId) => {
                const state = store.getState();
                const post = getPost(state, postId);
                const systemMessage = post ? isSystemMessage(post) : true;

                return state[`plugins-${manifest.id}`].connected && !systemMessage;
            },
        });
        registry.registerLinkTooltipComponent(LinkTooltip);
        registry.registerPostWillRenderEmbedComponent((embed) => embed.url && isUrlCanPreview(embed.url), LinkEmbedPreview, true);

        const {showRHSPlugin} = registry.registerRightHandSidebarComponent(SidebarRight, 'GitHub');
        store.dispatch(setShowRHSAction(() => store.dispatch(showRHSPlugin)));

        registry.registerWebSocketEventHandler(`custom_${pluginId}_connect`, handleConnect(store));
        registry.registerWebSocketEventHandler(`custom_${pluginId}_disconnect`, handleDisconnect(store));
        registry.registerWebSocketEventHandler(`custom_${pluginId}_config_update`, handleConfigurationUpdate(store));
        registry.registerWebSocketEventHandler(`custom_${pluginId}_refresh`, handleRefresh(store));
        registry.registerWebSocketEventHandler(`custom_${pluginId}_createIssue`, handleOpenCreateIssueModal(store));
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
