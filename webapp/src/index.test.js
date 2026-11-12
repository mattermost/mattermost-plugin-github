// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {getConnected} from '@/actions';

jest.mock('@/actions', () => ({
    ...jest.requireActual('@/actions'),
    getConnected: jest.fn(),
}));

jest.mock('@/components/post_menu_actions/attach_comment_to_issue', () => () => null);
jest.mock('@/components/modals/attach_comment_to_issue', () => () => null);
jest.mock('./components/modals/create_issue', () => () => null);
jest.mock('./components/post_menu_action/create_issue', () => () => null);
jest.mock('./components/sidebar_header', () => () => null);
jest.mock('./components/team_sidebar', () => () => null);
jest.mock('./components/user_attribute', () => () => null);
jest.mock('./components/sidebar_right', () => () => null);
jest.mock('./components/link_tooltip', () => () => null);

let plugin;
global.window.registerPlugin = (id, instance) => {
    plugin = instance;
};
require('./index');

function createRegistry() {
    let nextId = 0;
    const newId = () => `component-${++nextId}`;

    return {
        registerReducer: jest.fn(),
        registerLeftSidebarHeaderComponent: jest.fn(newId),
        registerBottomTeamSidebarComponent: jest.fn(newId),
        registerPopoverUserAttributesComponent: jest.fn(newId),
        registerRootComponent: jest.fn(newId),
        registerPostDropdownMenuAction: jest.fn(newId),
        registerLinkTooltipComponent: jest.fn(newId),
        registerRightHandSidebarComponent: jest.fn(() => ({id: newId(), showRHSPlugin: {type: 'SHOW_RHS'}})),
        registerWebSocketEventHandler: jest.fn(),
        registerReconnectHandler: jest.fn(),
        unregisterComponent: jest.fn(),
    };
}

const store = {
    getState: () => ({entities: {general: {config: {}}}}),
    dispatch: jest.fn(),
};

// Holds each getConnected request open until the test resolves it, like a slow network.
function deferGetConnected() {
    const pending = [];
    getConnected.mockImplementation(() => () => new Promise((resolve) => {
        pending.push(resolve);
    }));
    return () => pending.shift()({data: {}});
}

function resolveGetConnected() {
    getConnected.mockImplementation(() => () => Promise.resolve({data: {}}));
}

function registeredIds(registry) {
    return Object.entries(registry).
        filter(([name]) => name !== 'unregisterComponent').
        flatMap(([, fn]) => fn.mock.results.map(({value}) => value?.id ?? value)).
        filter(Boolean);
}

function unregisteredIds(registry) {
    return registry.unregisterComponent.mock.calls.map(([id]) => id);
}

describe('PluginClass', () => {
    beforeEach(resolveGetConnected);

    afterEach(() => {
        plugin.uninitialize();
    });

    it('unregisters every registered component on uninitialize', async () => {
        const registry = createRegistry();
        await plugin.initialize(registry, store);

        const ids = registeredIds(registry);
        expect(ids).toContain(registry.registerBottomTeamSidebarComponent.mock.results[0].value);

        plugin.uninitialize();

        expect(unregisteredIds(registry).sort()).toEqual([...ids].sort());
    });

    it('does not register components when uninitialized while connecting', async () => {
        const resolveConnected = deferGetConnected();
        const registry = createRegistry();

        const initializing = plugin.initialize(registry, store);
        plugin.uninitialize();
        resolveConnected();
        await initializing;

        expect(registeredIds(registry)).toEqual([]);
    });

    it('registers components only once when re-initialized while connecting', async () => {
        const resolveConnected = deferGetConnected();
        const first = createRegistry();
        const second = createRegistry();

        const firstInit = plugin.initialize(first, store);
        const secondInit = plugin.initialize(second, store);
        resolveConnected();
        resolveConnected();
        await Promise.all([firstInit, secondInit]);

        expect(registeredIds(first)).toEqual([]);
        expect(second.registerBottomTeamSidebarComponent).toHaveBeenCalledTimes(1);
    });

    it('unregisters the previous components when re-initialized', async () => {
        const first = createRegistry();
        await plugin.initialize(first, store);
        const firstIds = registeredIds(first);

        const second = createRegistry();
        await plugin.initialize(second, store);

        expect(unregisteredIds(first).sort()).toEqual([...firstIds].sort());
        expect(second.unregisterComponent).not.toHaveBeenCalled();
        expect(second.registerBottomTeamSidebarComponent).toHaveBeenCalledTimes(1);
    });
});
