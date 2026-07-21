import React from 'react';
import {act, fireEvent, render} from '@testing-library/react';

import {RHSStates} from '../../constants';

import SidebarButtons from './sidebar_buttons';

// The component hides its controls during E2E testing.
// eslint-disable-next-line no-underscore-dangle
global.__E2E_TESTING__ = false;

jest.mock('react-bootstrap', () => ({
    OverlayTrigger: ({children}) => children,
    Tooltip: ({children}) => children,
}));

jest.mock('mattermost-redux/utils/theme_utils', () => ({
    changeOpacity: (color) => color,
    makeStyleFromTheme: (createStyle) => (theme) => createStyle(theme),
}));

const baseProps = {
    connected: true,
    clientId: 'client-id',
    enterpriseURL: '',
    isTeamSidebar: false,
    reviewTargetDays: 0,
    reviews: [],
    theme: {
        centerChannelBg: '#ffffff',
        centerChannelColor: '#333333',
    },
    unreads: [],
    yourAssignments: [],
    yourPrs: [],
};

function renderSidebarButtons(overrides = {}) {
    const actions = overrides.actions || {
        getConnected: jest.fn(),
        getSidebarContent: jest.fn().mockResolvedValue({}),
        updateRhsState: jest.fn(),
    };
    const showRHSPlugin = jest.fn();
    const result = render(
        <SidebarButtons
            {...baseProps}
            {...overrides}
            actions={actions}
            showRHSPlugin={showRHSPlugin}
        />,
    );

    return {actions, showRHSPlugin, ...result};
}

beforeEach(() => {
    jest.clearAllMocks();
});

test('refreshes sidebar content when opening or switching the RHS view', async () => {
    const {actions, showRHSPlugin, container} = renderSidebarButtons();
    await act(async () => Promise.resolve());
    actions.getSidebarContent.mockClear();

    const links = container.querySelectorAll('a');
    await act(async () => {
        links[1].click();
    });
    await act(async () => {
        links[2].click();
    });

    expect(actions.updateRhsState).toHaveBeenNthCalledWith(1, RHSStates.PRS);
    expect(actions.updateRhsState).toHaveBeenNthCalledWith(2, RHSStates.REVIEWS);
    expect(showRHSPlugin).toHaveBeenCalledTimes(2);
    expect(actions.getSidebarContent).toHaveBeenCalledTimes(2);
});

test('does not start duplicate sidebar requests while an open refresh is pending', async () => {
    let resolveRequest;
    const pendingRequest = new Promise((resolve) => {
        resolveRequest = resolve;
    });
    const getSidebarContent = jest.fn().mockResolvedValueOnce({}).mockReturnValue(pendingRequest);
    const {actions, container} = renderSidebarButtons({
        actions: {
            getConnected: jest.fn(),
            getSidebarContent,
            updateRhsState: jest.fn(),
        },
    });
    await act(async () => Promise.resolve());
    getSidebarContent.mockClear();

    const links = container.querySelectorAll('a');
    act(() => {
        links[1].click();
        links[2].click();
    });

    expect(getSidebarContent).toHaveBeenCalledTimes(1);
    expect(actions.updateRhsState).toHaveBeenCalledTimes(2);
    await act(async () => {
        resolveRequest({});
        await pendingRequest;
    });
});

test('prevents the refresh anchor default action while a refresh is pending', async () => {
    let resolveRequest;
    const pendingRequest = new Promise((resolve) => {
        resolveRequest = resolve;
    });
    const getSidebarContent = jest.fn().mockResolvedValueOnce({}).mockReturnValue(pendingRequest);
    const {container} = renderSidebarButtons({
        actions: {
            getConnected: jest.fn(),
            getSidebarContent,
            updateRhsState: jest.fn(),
        },
    });
    await act(async () => Promise.resolve());

    const refreshLink = container.querySelector('a[href="#"]');
    expect(fireEvent.click(refreshLink)).toBe(false);
    expect(fireEvent.click(refreshLink)).toBe(false);

    await act(async () => {
        resolveRequest({});
        await pendingRequest;
    });
});
