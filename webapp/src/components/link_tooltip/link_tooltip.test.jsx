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

    test('should use html_url for opened by link if available', async () => {
        Client.getIssue.mockResolvedValueOnce({
            id: 1,
            title: 'Test Issue',
            body: 'Description',
            user: {
                login: 'testuser',
                html_url: 'https://github.com/testuser/profile',
            },
            state: 'open',
            labels: [],
            created_at: '2023-01-01T00:00:00Z',
        });

        wrapper = mount(<LinkTooltip {...baseProps}/>);

        await new Promise((resolve) => setTimeout(resolve, 0));
        wrapper.update();

        const link = wrapper.find('.opened-by a');
        expect(link.exists()).toBe(true);
        expect(link.prop('href')).toBe('https://github.com/testuser/profile');
    });

    test('should fallback to enterprise URL for opened by link if html_url missing', async () => {
        Client.getIssue.mockResolvedValueOnce({
            id: 1,
            title: 'Test Enterprise Issue',
            body: 'Description',
            user: {
                login: 'entuser',
            },
            state: 'open',
            labels: [],
            created_at: '2023-01-01T00:00:00Z',
        });

        const props = {
            ...baseProps,
            href: 'https://github.example.com/mattermost/mattermost-plugin-github/issues/3',
            enterpriseURL: 'https://github.example.com',
        };
        wrapper = mount(<LinkTooltip {...props}/>);

        await new Promise((resolve) => setTimeout(resolve, 0));
        wrapper.update();

        const link = wrapper.find('.opened-by a');
        expect(link.exists()).toBe(true);
        expect(link.prop('href')).toBe('https://github.example.com/entuser');
    });

    test('should handle enterprise URL with trailing slash for opened by link fallback', async () => {
        Client.getIssue.mockResolvedValueOnce({
            id: 1,
            title: 'Test Enterprise Issue',
            body: 'Description',
            user: {
                login: 'entuser',
            },
            state: 'open',
            labels: [],
            created_at: '2023-01-01T00:00:00Z',
        });

        const props = {
            ...baseProps,
            href: 'https://github.example.com/mattermost/mattermost-plugin-github/issues/3',
            enterpriseURL: 'https://github.example.com/',
        };
        wrapper = mount(<LinkTooltip {...props}/>);

        await new Promise((resolve) => setTimeout(resolve, 0));
        wrapper.update();

        const link = wrapper.find('.opened-by a');
        expect(link.prop('href')).toBe('https://github.example.com/entuser');
    });

    test('should default to github.com for opened by link if no enterpriseURL and no html_url', async () => {
        Client.getIssue.mockResolvedValueOnce({
            id: 1,
            title: 'Test Issue',
            body: 'Description',
            user: {
                login: 'clouduser',
            },
            state: 'open',
            labels: [],
            created_at: '2023-01-01T00:00:00Z',
        });

        wrapper = mount(<LinkTooltip {...baseProps}/>);

        await new Promise((resolve) => setTimeout(resolve, 0));
        wrapper.update();

        const link = wrapper.find('.opened-by a');
        expect(link.prop('href')).toBe('https://github.com/clouduser');
    });
});
