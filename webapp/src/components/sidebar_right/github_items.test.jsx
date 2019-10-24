// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {mount} from 'enzyme';

import GithubItems from './github_items';
import { RHSStates } from '../../constants';

describe('GithubItems', () => {
    const baseProps = {
        items: [],
        theme: {centerChannelColor: '#fff'},
        rhsState: RHSStates.PRS,
    };

    test('should match snapshot', () => {
        const wrapper = mount(<GithubItems {...baseProps}/>);
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot with items', () => {
        const items = [{
            id: 1, 
            repository: {full_name: 'test'}, 
            user:{login: 'manland'}, 
            title: 'make it work', 
            html_url: 'http://mattermost.com',
            labels: [],
            reason: 'mention'
        }]
        const wrapper = mount(<GithubItems {...baseProps} items={items}/>);
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot with items deleted', () => {
        const items = [{
            id: 1, 
            repository: {full_name: 'test'}, 
            user:{login: 'manland'}, 
            title: 'make it work', 
            html_url: 'http://mattermost.com',
            labels: [],
            reason: 'mention'
        },{
            id: 2, 
            repository: {full_name: 'test'}, 
            user:{login: 'manland'}, 
            title: 'make it work', 
            html_url: 'http://mattermost.com',
            labels: [],
            reason: 'mention'
        }]
        const wrapper = mount(<GithubItems {...baseProps} items={items}/>);
        expect(wrapper).toMatchSnapshot();
        expect(wrapper.find('div').first().instance().style.opacity).toBe('');
        expect(wrapper.find('div').last().instance().style.opacity).toBe('');
        wrapper.setProps({...baseProps, items: [items[1]]}); 
        expect(wrapper).toMatchSnapshot();
        expect(wrapper.find('div').first().instance().style.opacity).toBe('0.5');
        expect(wrapper.find('div').last().instance().style.opacity).toBe('');
    });

    test('should not delete items deleted if user refresh', () => {
        const items = [{
            id: 1, 
            repository: {full_name: 'test'}, 
            user:{login: 'manland'}, 
            title: 'make it work', 
            html_url: 'http://mattermost.com',
            labels: [],
            reason: 'mention'
        }]
        const wrapper = mount(<GithubItems {...baseProps} items={items}/>);
        expect(wrapper.find('div').first().instance().style.opacity).toBe('');
        wrapper.setProps({...baseProps});
        expect(wrapper.find('div').first().instance().style.opacity).toBe('0.5');
        wrapper.setProps({...baseProps, items: []});
        expect(wrapper.find('div').first().instance().style.opacity).toBe('0.5');
    });
});
