import React from 'react';
import {shallow} from 'enzyme';

import IssueAttributeSelector from 'components/issue_attribute_selector';

import GithubLabelSelector from './github_label_selector';

describe('components/GithubLabelSelector', () => {
    const baseActions = {
        getLabelOptions: jest.fn().mockResolvedValue({data: [{name: 'bug'}, {name: 'enhancement'}]}),
    };

    const baseProps = {
        repoName: 'test-repo',
        theme: {
            sidebarBg: '#ffffff',
            sidebarText: '#000000',
        },
        selectedLabels: ['bug'],
        onChange: jest.fn(),
        actions: baseActions,
    };

    test('should match snapshot', () => {
        const wrapper = shallow(<GithubLabelSelector {...baseProps}/>);
        expect(wrapper).toMatchSnapshot();
    });

    test('should render IssueAttributeSelector with correct props', () => {
        const wrapper = shallow(<GithubLabelSelector {...baseProps}/>);
        const issueAttributeSelector = wrapper.find(IssueAttributeSelector);

        expect(issueAttributeSelector.exists()).toBe(true);
        expect(issueAttributeSelector.prop('isMulti')).toBe(true);
        expect(issueAttributeSelector.prop('selection')).toEqual(baseProps.selectedLabels);
        expect(issueAttributeSelector.prop('loadOptions')).toEqual(wrapper.instance().loadLabels);
    });

    test('should call loadLabels and return correct options', async () => {
        const wrapper = shallow(<GithubLabelSelector {...baseProps}/>);
        const options = await wrapper.instance().loadLabels();

        expect(baseActions.getLabelOptions).toHaveBeenCalledWith(baseProps.repoName);
        expect(options).toEqual([
            {value: 'bug', label: 'bug'},
            {value: 'enhancement', label: 'enhancement'},
        ]);
    });

    test('should handle loadLabels error gracefully', async () => {
        const errorActions = {
            getLabelOptions: jest.fn().mockResolvedValue({error: 'Failed to load'}),
        };
        const props = {...baseProps, actions: errorActions};
        const wrapper = shallow(<GithubLabelSelector {...props}/>);

        await expect(wrapper.instance().loadLabels()).rejects.toThrow('Failed to load labels');
    });

    test('should handle empty repoName in loadLabels', async () => {
        const props = {...baseProps, repoName: ''};
        const wrapper = shallow(<GithubLabelSelector {...props}/>);
        const options = await wrapper.instance().loadLabels();

        expect(options).toEqual([]);
        expect(baseActions.getLabelOptions).not.toHaveBeenCalled();
    });

    test('should call onChange with correct values', () => {
        const wrapper = shallow(<GithubLabelSelector {...baseProps}/>);
        const instance = wrapper.instance();
        const selection = [{value: 'bug'}, {value: 'enhancement'}];

        instance.onChange(selection);
        expect(baseProps.onChange).toHaveBeenCalledWith(['bug', 'enhancement']);
    });
});
