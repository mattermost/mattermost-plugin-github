import React from 'react';
import {mount} from 'enzyme';

import Client from '@/client';

import {LinkTooltip} from './link_tooltip';

jest.mock('@/client', () => ({
    getIssue: jest.fn(),
    getPullRequest: jest.fn(),
}));

jest.mock('react-markdown', () => () => <div/>);

describe('LinkTooltip', () => {
    const baseProps = {
        href: 'https://github.com/mattermost/mattermost-plugin-github/issues/1',
        connected: true,
        show: true,
        theme: {
            centerChannelBg: '#ffffff',
            centerChannelColor: '#333333',
        },
        enterpriseURL: '',
    };

    let wrapper;

    beforeEach(() => {
        jest.clearAllMocks();
    });

    afterEach(() => {
        if (wrapper && wrapper.length) {
            wrapper.unmount();
        }
    });

    test('should fetch issue for github.com link', () => {
        // We need to use mount or wait for useEffect?
        // shallow renders the component, useEffect is a hook.
        // Enzyme shallow supports hooks in newer versions, but let's check if we need to manually trigger logic.
        // The component uses useEffect to call initData.
        wrapper = mount(<LinkTooltip {...baseProps}/>);
        expect(Client.getIssue).toHaveBeenCalledWith('mattermost', 'mattermost-plugin-github', '1');
    });

    test('should fetch pull request for github.com link', () => {
        const props = {
            ...baseProps,
            href: 'https://github.com/mattermost/mattermost-plugin-github/pull/2',
        };
        wrapper = mount(<LinkTooltip {...props}/>);
        expect(Client.getPullRequest).toHaveBeenCalledWith('mattermost', 'mattermost-plugin-github', '2');
    });

    test('should fetch issue for enterprise link', () => {
        const props = {
            ...baseProps,
            href: 'https://github.example.com/mattermost/mattermost-plugin-github/issues/3',
            enterpriseURL: 'https://github.example.com',
        };
        wrapper = mount(<LinkTooltip {...props}/>);
        expect(Client.getIssue).toHaveBeenCalledWith('mattermost', 'mattermost-plugin-github', '3');
    });

    test('should fetch pull request for enterprise link', () => {
        const props = {
            ...baseProps,
            href: 'https://github.example.com/mattermost/mattermost-plugin-github/pull/4',
            enterpriseURL: 'https://github.example.com',
        };
        wrapper = mount(<LinkTooltip {...props}/>);
        expect(Client.getPullRequest).toHaveBeenCalledWith('mattermost', 'mattermost-plugin-github', '4');
    });

    test('should handle enterprise URL with trailing slash', () => {
        const props = {
            ...baseProps,
            href: 'https://github.example.com/mattermost/mattermost-plugin-github/issues/5',
            enterpriseURL: 'https://github.example.com/',
        };
        wrapper = mount(<LinkTooltip {...props}/>);
        expect(Client.getIssue).toHaveBeenCalledWith('mattermost', 'mattermost-plugin-github', '5');
    });

    test('should not fetch if enterprise URL does not match', () => {
        const props = {
            ...baseProps,
            href: 'https://other-github.com/mattermost/mattermost-plugin-github/issues/6',
            enterpriseURL: 'https://github.example.com',
        };
        wrapper = mount(<LinkTooltip {...props}/>);
        expect(Client.getIssue).not.toHaveBeenCalled();
    });
});
