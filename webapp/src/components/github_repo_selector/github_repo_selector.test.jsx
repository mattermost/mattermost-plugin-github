import React from 'react';
import {shallow} from 'enzyme';

import ReactSelectSetting from 'components/react_select_setting';

import GithubRepoSelector from './github_repo_selector';

describe('components/GithubRepoSelector', () => {
    const baseActions = {
        getRepos: jest.fn(),
    };

    const baseProps = {
        yourRepos: [
            {full_name: 'user/repo1', permissions: {admin: true, push: true, pull: true}},
            {full_name: 'user/repo2', permissions: {admin: false, push: true, pull: true}},
        ],
        theme: {
            sidebarBg: '#ffffff',
            sidebarText: '#000000',
        },
        onChange: jest.fn(),
        value: 'user/repo1',
        addValidate: jest.fn(),
        removeValidate: jest.fn(),
        actions: baseActions,
    };

    beforeEach(() => {
        jest.clearAllMocks();
    });

    test('should match snapshot', () => {
        const wrapper = shallow(<GithubRepoSelector {...baseProps}/>);
        expect(wrapper).toMatchSnapshot();
    });

    test('should call getRepos on componentDidMount', () => {
        shallow(<GithubRepoSelector {...baseProps}/>);
        expect(baseActions.getRepos).toHaveBeenCalled();
    });

    test('should render ReactSelectSetting with correct props', () => {
        const wrapper = shallow(<GithubRepoSelector {...baseProps}/>);
        const selectSetting = wrapper.find(ReactSelectSetting);

        expect(selectSetting.exists()).toBe(true);
        expect(selectSetting.prop('name')).toBe('repo');
        expect(selectSetting.prop('label')).toBe('Repository');
        expect(selectSetting.prop('limitOptions')).toBe(true);
        expect(selectSetting.prop('required')).toBe(true);
        expect(selectSetting.prop('isMulti')).toBe(false);
        expect(selectSetting.prop('theme')).toEqual(baseProps.theme);
        expect(selectSetting.prop('value')).toEqual({
            value: baseProps.value,
            label: baseProps.value,
        });
    });

    test('should handle onChange correctly', () => {
        const wrapper = shallow(<GithubRepoSelector {...baseProps}/>);
        const selectSetting = wrapper.find(ReactSelectSetting);

        const selectedRepo = 'user/repo2';
        selectSetting.simulate('change', null, selectedRepo);

        expect(baseProps.onChange).toHaveBeenCalledWith({
            name: selectedRepo,
            permissions: {admin: false, push: true, pull: true},
        });
    });

    test('should handle empty repos gracefully', () => {
        const props = {...baseProps, yourRepos: []};
        const wrapper = shallow(<GithubRepoSelector {...props}/>);
        const selectSetting = wrapper.find(ReactSelectSetting);

        expect(selectSetting.prop('options')).toEqual([]);
        expect(selectSetting.prop('value')).toBeUndefined();
    });

    test('should handle missing value prop gracefully', () => {
        const props = {...baseProps, value: null};
        const wrapper = shallow(<GithubRepoSelector {...props}/>);
        const selectSetting = wrapper.find(ReactSelectSetting);

        expect(selectSetting.prop('value')).toBeUndefined();
    });

    test('should render help text correctly', () => {
        const wrapper = shallow(<GithubRepoSelector {...baseProps}/>);
        const helpText = wrapper.find('.help-text');

        expect(helpText.exists()).toBe(true);
        expect(helpText.text()).toBe('Returns GitHub repositories connected to the user account');
    });
});
