// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {id as pluginId} from '../manifest';

const getPluginState = (state) => state['plugins-' + pluginId] || {};

export const isAttachCommentToIssueModalVisible = (state) => getPluginState(state).attachCommentToIssueModalVisible;

export const getAttachCommentToIssueModalForPostId = (state) => getPluginState(state).attachCommentToIssueModalForPostId;

export const isUserConnected = (state) => getPluginState(state).userConnected;

export const isInstanceInstalled = (state) => getPluginState(state).instanceInstalled;
