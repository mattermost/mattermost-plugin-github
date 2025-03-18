import React from 'react';
import {shallow} from 'enzyme';

import CreateIssueModal from './create_issue';

jest.mock('utils/user_utils', () => ({
    getErrorMessage: jest.fn(() => 'Error occurred'),
}));

describe('CreateIssueModal', () => {
    const defaultProps = {
        close: jest.fn(),
        create: jest.fn(() => Promise.resolve({})),
        post: null,
        theme: {
            centerChannelColor: '#000',
            centerChannelBg: '#fff',
        },
        visible: true,
    };

    it('should render correctly with default props', () => {
        const wrapper = shallow(<CreateIssueModal {...defaultProps}/>);
        expect(wrapper).toMatchSnapshot();
    });

    it('should call close prop when handleClose is called', () => {
        const wrapper = shallow(<CreateIssueModal {...defaultProps}/>);
        wrapper.instance().handleClose();
        expect(defaultProps.close).toHaveBeenCalled();
    });

    it('should call create prop when form is submitted with valid data', async () => {
        const wrapper = shallow(<CreateIssueModal {...defaultProps}/>);
        wrapper.setState({issueTitle: 'Test Issue'});

        await wrapper.instance().handleCreate({preventDefault: jest.fn()});
        expect(defaultProps.create).toHaveBeenCalled();
    });

    it('should display error message when create returns an error', async () => {
        const mockCreateFunction = jest.fn().mockResolvedValue({error: {message: 'Some error'}});
        const errorProps = {
            ...defaultProps,
            create: mockCreateFunction,
        };

        const wrapper = shallow(<CreateIssueModal {...errorProps}/>);
        wrapper.setState({issueTitle: 'Test Issue'});

        await wrapper.instance().handleCreate({preventDefault: jest.fn()});
        wrapper.update();

        expect(wrapper.find('.help-text.error-text').text()).toEqual('Error occurred');
    });

    it('should show validation error when issueTitle is empty', async () => {
        const wrapper = shallow(<CreateIssueModal {...defaultProps}/>);
        wrapper.setState({issueTitle: ''});

        await wrapper.instance().handleCreate({preventDefault: jest.fn()});
        expect(wrapper.state('issueTitleValid')).toBe(false);
        expect(wrapper.state('showErrors')).toBe(true);
    });

    it('should update repo state when handleRepoChange is called', () => {
        const wrapper = shallow(<CreateIssueModal {...defaultProps}/>);
        const repo = {name: 'repo-name'};

        wrapper.instance().handleRepoChange(repo);
        expect(wrapper.state('repo')).toEqual(repo);
    });

    it('should update labels state when handleLabelsChange is called', () => {
        const wrapper = shallow(<CreateIssueModal {...defaultProps}/>);
        const labels = ['label1', 'label2'];

        wrapper.instance().handleLabelsChange(labels);
        expect(wrapper.state('labels')).toEqual(labels);
    });

    it('should update assignees state when handleAssigneesChange is called', () => {
        const wrapper = shallow(<CreateIssueModal {...defaultProps}/>);
        const assignees = ['user1', 'user2'];

        wrapper.instance().handleAssigneesChange(assignees);
        expect(wrapper.state('assignees')).toEqual(assignees);
    });

    it('should set issueDescription state when post prop is updated', () => {
        const wrapper = shallow(<CreateIssueModal {...defaultProps}/>);
        const post = {message: 'test post'};
        wrapper.setProps({post});

        expect(wrapper.state('issueDescription')).toEqual(post.message);
    });

    it('should not display attribute selectors when repo does not have push permissions', () => {
        const wrapper = shallow(<CreateIssueModal {...defaultProps}/>);
        wrapper.setState({repo: {name: 'repo-name', permissions: {push: false}}});

        expect(wrapper.instance().renderIssueAttributeSelectors()).toBeNull();
    });

    it('should display attribute selectors when repo has push permissions', () => {
        const wrapper = shallow(<CreateIssueModal {...defaultProps}/>);
        wrapper.setState({repo: {name: 'repo-name', permissions: {push: true}}});

        const selectors = wrapper.instance().renderIssueAttributeSelectors();
        expect(selectors).not.toBeNull();
    });
});
