import React from 'react';
import {shallow} from 'enzyme';

import IssueAttributeSelector from 'components/issue_attribute_selector';

import GithubAssigneeSelector from './github_assignee_selector';

describe('components/GithubAssigneeSelector', () => {
    const baseActions = {
        getAssigneeOptions: jest.fn().mockResolvedValue({data: [{login: 'user1'}, {login: 'user2'}]}),
    };

    const baseProps = {
        repoName: 'test-repo',
        theme: {
            sidebarBg: '#ffffff',
            sidebarText: '#000000',
        },
        selectedAssignees: ['user1'],
        onChange: jest.fn(),
        actions: baseActions,
    };

    test('should match snapshot', () => {
        const wrapper = shallow(<GithubAssigneeSelector {...baseProps}/>);
        expect(wrapper).toMatchSnapshot();
    });

    test('should render IssueAttributeSelector with correct props', () => {
        const wrapper = shallow(<GithubAssigneeSelector {...baseProps}/>);
        const issueAttributeSelector = wrapper.find(IssueAttributeSelector);

        expect(issueAttributeSelector.exists()).toBe(true);
        expect(issueAttributeSelector.prop('isMulti')).toBe(true);
        expect(issueAttributeSelector.prop('selection')).toEqual(baseProps.selectedAssignees);
        expect(issueAttributeSelector.prop('loadOptions')).toEqual(wrapper.instance().loadAssignees);
    });

    test('should call loadAssignees and return correct options', async () => {
        const wrapper = shallow<GithubAssigneeSelector>(<GithubAssigneeSelector {...baseProps}/>);
        const options = await wrapper.instance().loadAssignees();

        expect(baseActions.getAssigneeOptions).toHaveBeenCalledWith(baseProps.repoName);
        expect(options).toEqual([
            {value: 'user1', label: 'user1'},
            {value: 'user2', label: 'user2'},
        ]);
    });

    test('should handle loadAssignees error gracefully', async () => {
        const errorActions = {
            getAssigneeOptions: jest.fn().mockResolvedValue({error: 'Failed to load'}),
        };
        const props = {...baseProps, actions: errorActions};
        const wrapper = shallow<GithubAssigneeSelector>(<GithubAssigneeSelector {...props}/>);

        await expect(wrapper.instance().loadAssignees()).rejects.toThrow('Failed to load assignees');
    });

    test('should handle empty repoName in loadAssignees', async () => {
        const props = {...baseProps, repoName: ''};
        const wrapper = shallow<GithubAssigneeSelector>(<GithubAssigneeSelector {...props}/>);
        const options = await wrapper.instance().loadAssignees();

        expect(options).toEqual([]);
        expect(baseActions.getAssigneeOptions).not.toHaveBeenCalled();
    });

    test('should call onChange with correct values', () => {
        const wrapper = shallow(<GithubAssigneeSelector {...baseProps}/>);
        const instance = wrapper.instance();
        const selection = [{value: 'user1'}, {value: 'user2'}];

        instance.onChange(selection);
        expect(baseProps.onChange).toHaveBeenCalledWith(['user1', 'user2']);
    });
});
