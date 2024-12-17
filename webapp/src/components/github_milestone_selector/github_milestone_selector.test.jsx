import React from 'react';
import {shallow} from 'enzyme';

import IssueAttributeSelector from 'components/issue_attribute_selector';

import GithubMilestoneSelector from './github_milestone_selector';

describe('components/GithubMilestoneSelector', () => {
    const baseActions = {
        getMilestoneOptions: jest.fn().mockResolvedValue({
            data: [
                {number: 1, title: 'Milestone 1'},
                {number: 2, title: 'Milestone 2'},
            ],
        }),
    };

    const baseProps = {
        repoName: 'test-repo',
        theme: {
            sidebarBg: '#ffffff',
            sidebarText: '#000000',
        },
        selectedMilestone: {value: 1, label: 'Milestone 1'},
        onChange: jest.fn(),
        actions: baseActions,
    };

    test('should match snapshot', () => {
        const wrapper = shallow(<GithubMilestoneSelector {...baseProps}/>);
        expect(wrapper).toMatchSnapshot();
    });

    test('should render IssueAttributeSelector with correct props', () => {
        const wrapper = shallow(<GithubMilestoneSelector {...baseProps}/>);
        const issueAttributeSelector = wrapper.find(IssueAttributeSelector);

        expect(issueAttributeSelector.exists()).toBe(true);
        expect(issueAttributeSelector.prop('isMulti')).toBe(false);
        expect(issueAttributeSelector.prop('selection')).toEqual(baseProps.selectedMilestone);
        expect(issueAttributeSelector.prop('loadOptions')).toEqual(wrapper.instance().loadMilestones);
    });

    test('should call loadMilestones and return correct options', async () => {
        const wrapper = shallow(<GithubMilestoneSelector {...baseProps}/>);
        const options = await wrapper.instance().loadMilestones();

        expect(baseActions.getMilestoneOptions).toHaveBeenCalledWith(baseProps.repoName);
        expect(options).toEqual([
            {value: 1, label: 'Milestone 1'},
            {value: 2, label: 'Milestone 2'},
        ]);
    });

    test('should handle loadMilestones error gracefully', async () => {
        const errorActions = {
            getMilestoneOptions: jest.fn().mockResolvedValue({error: 'Failed to load'}),
        };
        const props = {...baseProps, actions: errorActions};
        const wrapper = shallow(<GithubMilestoneSelector {...props}/>);

        await expect(wrapper.instance().loadMilestones()).rejects.toThrow('Failed to load milestones');
    });

    test('should handle empty repoName in loadMilestones', async () => {
        const props = {...baseProps, repoName: ''};
        const wrapper = shallow(<GithubMilestoneSelector {...props}/>);
        const options = await wrapper.instance().loadMilestones();

        expect(options).toEqual([]);
        expect(baseActions.getMilestoneOptions).not.toHaveBeenCalled();
    });

    test('should call onChange with correct values', () => {
        const wrapper = shallow(<GithubMilestoneSelector {...baseProps}/>);
        const issueAttributeSelector = wrapper.find(IssueAttributeSelector);

        const selection = [{value: 1, label: 'Milestone 1'}];
        issueAttributeSelector.simulate('change', selection);

        expect(baseProps.onChange).toHaveBeenCalledWith([{value: 1, label: 'Milestone 1'}]);
    });

    test('should handle no milestone data gracefully', async () => {
        const emptyActions = {
            getMilestoneOptions: jest.fn().mockResolvedValue({data: []}),
        };
        const props = {...baseProps, actions: emptyActions};
        const wrapper = shallow(<GithubMilestoneSelector {...props}/>);

        const options = await wrapper.instance().loadMilestones();
        expect(options).toEqual([]);
    });

    test('should handle invalid milestone data structure gracefully', async () => {
        const invalidActions = {
            getMilestoneOptions: jest.fn().mockResolvedValue({
                data: [{number: null, title: null}],
            }),
        };
        const props = {...baseProps, actions: invalidActions};
        const wrapper = shallow(<GithubMilestoneSelector {...props}/>);

        const options = await wrapper.instance().loadMilestones();
        expect(options).toEqual([
            {value: null, label: null},
        ]);
    });

    test('should handle no selected milestone', () => {
        const props = {
            ...baseProps, selectedMilestone: {value: '', label: 'No Milestone'}};
        const wrapper = shallow(<GithubMilestoneSelector {...props}/>);
        const issueAttributeSelector = wrapper.find(IssueAttributeSelector);

        expect(issueAttributeSelector.exists()).toBe(true);
        expect(issueAttributeSelector.prop('selection')).toEqual({value: '', label: 'No Milestone'});
    });

    test('should reload milestones when repoName changes', async () => {
        const initialProps = {...baseProps, repoName: 'repo1'};
        const wrapper = shallow(<GithubMilestoneSelector {...initialProps}/>);

        const initialActions = {
            getMilestoneOptions: jest.fn().mockResolvedValue({
                data: [{number: 1, title: 'Milestone 1'}],
            }),
        };
        wrapper.setProps({actions: initialActions});
        await wrapper.instance().loadMilestones();

        expect(initialActions.getMilestoneOptions).toHaveBeenCalledWith('repo1');

        const updatedActions = {
            getMilestoneOptions: jest.fn().mockResolvedValue({
                data: [{number: 2, title: 'Milestone 2'}],
            }),
        };
        wrapper.setProps({repoName: 'repo2', actions: updatedActions});

        await wrapper.instance().loadMilestones();

        expect(updatedActions.getMilestoneOptions).toHaveBeenCalledWith('repo2');
    });
});
