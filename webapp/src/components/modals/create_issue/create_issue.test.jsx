import React from 'react';
import { shallow } from 'enzyme';
import CreateIssueModal from './create_issue';

jest.mock('utils/user_utils', () => ({
    getErrorMessage: jest.fn(() => 'Error occurred'),
}));

describe('CreateIssueModal', () => {
    const defaultProps = {
        close: jest.fn(),
        create: jest.fn(() => Promise.resolve({})),
        post: null,
        title: '',
        channelId: '',
        theme: {
            centerChannelColor: '#000',
            centerChannelBg: '#fff'
        },
        visible: true,
    };

    it('should render correctly with default props', () => {
        const wrapper = shallow(<CreateIssueModal {...defaultProps} />);
        expect(wrapper).toMatchSnapshot();
    });

    it('should call close prop when handleClose is called', () => {
        const wrapper = shallow(<CreateIssueModal {...defaultProps} />);
        wrapper.instance().handleClose();
        expect(defaultProps.close).toHaveBeenCalled();
    });

    it('should call create prop when form is submitted with valid data', async () => {
        const wrapper = shallow(<CreateIssueModal {...defaultProps} />);
        wrapper.setState({ issueTitle: 'Test Issue' });

        await wrapper.instance().handleCreate({ preventDefault: () => {} });
        expect(defaultProps.create).toHaveBeenCalled();
    });

    it('should display error message when create returns an error', async () => {
        const errorProps = {
            ...defaultProps,
            create: jest.fn(() => Promise.resolve({ error: { message: 'Some error' } })),
        };

        const wrapper = shallow(<CreateIssueModal {...errorProps} />);
        wrapper.setState({ issueTitle: 'Test Issue' });

        await wrapper.instance().handleCreate({ preventDefault: () => {} });
        wrapper.update();

        expect(wrapper.find('.help-text.error-text').text()).toEqual('Error occurred');
    });

    it('should show validation error when issueTitle is empty', async () => {
        const wrapper = shallow(<CreateIssueModal {...defaultProps} />);
        wrapper.setState({ issueTitle: '' });

        await wrapper.instance().handleCreate({ preventDefault: () => {} });
        expect(wrapper.state('issueTitleValid')).toBe(false);
        expect(wrapper.state('showErrors')).toBe(true);
    });

    it('should update repo state when handleRepoChange is called', () => {
        const wrapper = shallow(<CreateIssueModal {...defaultProps} />);
        const repo = { name: 'repo-name' };

        wrapper.instance().handleRepoChange(repo);
        expect(wrapper.state('repo')).toEqual(repo);
    });

    it('should update labels state when handleLabelsChange is called', () => {
        const wrapper = shallow(<CreateIssueModal {...defaultProps} />);
        const labels = ['label1', 'label2'];

        wrapper.instance().handleLabelsChange(labels);
        expect(wrapper.state('labels')).toEqual(labels);
    });

    it('should update assignees state when handleAssigneesChange is called', () => {
        const wrapper = shallow(<CreateIssueModal {...defaultProps} />);
        const assignees = ['user1', 'user2'];

        wrapper.instance().handleAssigneesChange(assignees);
        expect(wrapper.state('assignees')).toEqual(assignees);
    });

    it('should set issueDescription state when post prop is updated', () => {
        const wrapper = shallow(<CreateIssueModal {...defaultProps} />);
        const post = { message: 'test post' };
        wrapper.setProps({ post });

        expect(wrapper.state('issueDescription')).toEqual(post.message);
    });

    it('should not display attribute selectors when repo does not have push permissions', () => {
        const wrapper = shallow(<CreateIssueModal {...defaultProps} />);
        wrapper.setState({ repo: { name: 'repo-name', permissions: { push: false } } });

        expect(wrapper.instance().renderIssueAttributeSelectors()).toBeNull();
    });

    it('should display attribute selectors when repo has push permissions', () => {
        const wrapper = shallow(<CreateIssueModal {...defaultProps} />);
        wrapper.setState({ repo: { name: 'repo-name', permissions: { push: true } } });

        const selectors = wrapper.instance().renderIssueAttributeSelectors();
        expect(selectors).not.toBeNull();
    });
});


























// import React from 'react';
// import { shallow } from 'enzyme';
// import CreateIssueModal from './create_issue';

// jest.mock('utils/user_utils', () => ({
//     getErrorMessage: jest.fn(() => 'Error occurred'),
// }));

// describe('CreateIssueModal', () => {
//     const defaultProps = {
//         close: jest.fn(),
//         create: jest.fn(() => Promise.resolve({})),
//         post: null,
//         title: '',
//         channelId: '',
//         theme: {
//             centerChannelColor: '#000',
//             centerChannelBg: '#fff'
//         },
//         visible: true,
//     };

//     it('should render correctly with default props', () => {
//         const wrapper = shallow(<CreateIssueModal {...defaultProps} />);
//         expect(wrapper).toMatchSnapshot();
//     });

//     it('should call close prop when handleClose is called', () => {
//         const wrapper = shallow(<CreateIssueModal {...defaultProps} />);
//         wrapper.instance().handleClose();
//         expect(defaultProps.close).toHaveBeenCalled();
//     });

//     it('should call create prop when form is submitted with valid data', async () => {
//         const wrapper = shallow(<CreateIssueModal {...defaultProps} />);
//         wrapper.setState({ issueTitle: 'Test Issue' });

//         await wrapper.instance().handleCreate({ preventDefault: () => {} });
//         expect(defaultProps.create).toHaveBeenCalled();
//     });

//     it('should display error message when create returns an error', async () => {
//         const errorProps = {
//             ...defaultProps,
//             create: jest.fn(() => Promise.resolve({ error: { message: 'Some error' } })),
//         };

//         const wrapper = shallow(<CreateIssueModal {...errorProps} />);
//         wrapper.setState({ issueTitle: 'Test Issue' });

//         await wrapper.instance().handleCreate({ preventDefault: () => {} });
//         wrapper.update();

//         expect(wrapper.find('.help-text.error-text').text()).toEqual('Error occurred');
//     });
// });



///// IN ONE ///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
















// import React from 'react';
// import { shallow } from 'enzyme';
// import CreateIssueModal from './create_issue';

// describe('CreateIssueModal', () => {
//     const defaultProps = {
    //     close: jest.fn(),
    //     create: jest.fn(),
    //     post: { message: 'Post message' },
    //     title: 'Initial Title',
    //     channelId: 'channel-123',
    //     theme: {
    //         centerChannelColor: '#000',
    //         centerChannelBg: '#fff'
    //     },
    //     visible: true,
    // };

//     // it('should initialize state correctly from props', () => {
//     //     const wrapper = shallow(<CreateIssueModal {...defaultProps} />);
//     //     expect(wrapper.state('issueDescription')).toBe('Post message');
//     //     expect(wrapper.state('issueTitle')).toBe('Initial Title');
//     // });

//     // it('should update state when post prop changes', () => {
//     //     const wrapper = shallow(<CreateIssueModal {...defaultProps} />);
//     //     expect(wrapper.state('issueDescription')).toBe('Post message');

//     //     // Update props
//     //     wrapper.setProps({ post: { message: 'New post message' } });
//     //     wrapper.update(); // Force re-render to make sure `componentDidUpdate` runs
//     //     expect(wrapper.state('issueDescription')).toBe('New post message');
//     // });

//     it('should update state when channelId or title props change', () => {
//         const wrapper = shallow(<CreateIssueModal {...defaultProps} />);
//         expect(wrapper.state('issueTitle')).toBe('Initial Title');

//         // Update props
//         wrapper.setProps({ title: 'New Title', channelId: 'channel-456' });
//         wrapper.update();
//         expect(wrapper.state('issueTitle')).toBe('New Title');
//     });
// });


//c -------------------------------------------------------



// import React from 'react';
// import { shallow } from 'enzyme';
// import CreateIssueModal from './create_issue';

// describe('CreateIssueModal', () => {
//     let wrapper;
//     const defaultProps = {
//         close: jest.fn(),
//         create: jest.fn(),
//         post: null,
//         title: '',
//         channelId: '',
//         theme: {
//             centerChannelColor: '#000',
//             centerChannelBg: '#fff'
//         },
//         visible: true,
//     };

//     beforeEach(() => {
//         wrapper = shallow(<CreateIssueModal {...defaultProps} />);
//         wrapper.setState({ repo: { name: 'test-repo', permissions: { push: true } } });
//         wrapper.update();
//     });

//     it('should render issue attribute selectors when repo is selected', () => {
//         expect(wrapper.find('GithubLabelSelector').exists()).toBe(true);
//         // expect(wrapper.find('GithubAssigneeSelector').exists()).toBe(true);
//         // expect(wrapper.find('GithubMilestoneSelector').exists()).toBe(true);
//     });
// });
