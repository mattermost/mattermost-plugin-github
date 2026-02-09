import React from 'react';
import {render, screen, waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import CreateIssueModal from './create_issue';

jest.mock('@/utils/user_utils', () => ({
    getErrorMessage: jest.fn(() => 'Error occurred'),
}));

jest.mock('@/components/github_repo_selector', () => ({
    __esModule: true,
    default: ({onChange, addValidate}) => {
        if (addValidate) {
            addValidate('repo', () => true);
        }

        return (
            <div data-testid='github-repo-selector'>
                <button
                    type='button'
                    onClick={() => onChange({name: 'test-repo', permissions: {push: true}})}
                >
                    {'Select Repo'}
                </button>
                <button
                    type='button'
                    onClick={() => onChange({name: 'no-push-repo', permissions: {push: false}})}
                >
                    {'Select No Push Repo'}
                </button>
            </div>
        );
    },
}));

jest.mock('@/components/github_label_selector', () => ({
    __esModule: true,
    default: () => <div data-testid='github-label-selector'>{'Label Selector'}</div>,
}));

jest.mock('@/components/github_assignee_selector', () => ({
    __esModule: true,
    default: () => <div data-testid='github-assignee-selector'>{'Assignee Selector'}</div>,
}));

jest.mock('@/components/github_milestone_selector', () => ({
    __esModule: true,
    default: () => <div data-testid='github-milestone-selector'>{'Milestone Selector'}</div>,
}));

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

beforeEach(() => {
    jest.clearAllMocks();
});

test('CreateIssueModal should render correctly with default props', () => {
    render(<CreateIssueModal {...defaultProps}/>);
    expect(screen.getByText('Create GitHub Issue')).toBeInTheDocument();
    expect(screen.getByText('Submit')).toBeInTheDocument();
    expect(screen.getByText('Cancel')).toBeInTheDocument();
});

test('CreateIssueModal should call close prop when handleClose is called', async () => {
    render(<CreateIssueModal {...defaultProps}/>);

    const cancelButton = screen.getByText('Cancel');
    await userEvent.click(cancelButton);

    expect(defaultProps.close).toHaveBeenCalled();
});

test('CreateIssueModal should call create prop when form is submitted with valid data', async () => {
    render(<CreateIssueModal {...defaultProps}/>);

    const titleInput = screen.getByRole('textbox', {name: /title for the github issue/i});
    await userEvent.clear(titleInput);
    await userEvent.type(titleInput, 'Test Issue');

    const selectRepoButton = screen.getByText('Select Repo');
    await userEvent.click(selectRepoButton);

    const submitButton = screen.getByRole('button', {name: 'Submit'});
    await userEvent.click(submitButton);

    await waitFor(() => expect(defaultProps.create).toHaveBeenCalledWith(expect.objectContaining({
        title: 'Test Issue',
        repo: 'test-repo',
    })));
});

test('CreateIssueModal should display error message when create returns an error', async () => {
    const mockCreateFunction = jest.fn().mockResolvedValue({error: {message: 'Some error'}});
    const errorProps = {
        ...defaultProps,
        create: mockCreateFunction,
    };

    render(<CreateIssueModal {...errorProps}/>);

    const titleInput = screen.getByRole('textbox', {name: /title for the github issue/i});
    await userEvent.clear(titleInput);
    await userEvent.type(titleInput, 'Test Issue');

    const selectRepoButton = screen.getByText('Select Repo');
    await userEvent.click(selectRepoButton);

    const submitButton = screen.getByRole('button', {name: 'Submit'});
    await userEvent.click(submitButton);

    await waitFor(() => expect(screen.getByText('Error occurred')).toBeInTheDocument());
});

test('CreateIssueModal should show validation error when issueTitle is empty', async () => {
    render(<CreateIssueModal {...defaultProps}/>);

    const selectRepoButton = screen.getByText('Select Repo');
    await userEvent.click(selectRepoButton);

    const submitButton = screen.getByRole('button', {name: 'Submit'});
    await userEvent.click(submitButton);

    expect(defaultProps.create).not.toHaveBeenCalled();
});

test('CreateIssueModal should update repo state when handleRepoChange is called', async () => {
    render(<CreateIssueModal {...defaultProps}/>);

    const selectRepoButton = screen.getByText('Select Repo');
    await userEvent.click(selectRepoButton);

    expect(screen.getByTestId('github-label-selector')).toBeInTheDocument();
    expect(screen.getByTestId('github-assignee-selector')).toBeInTheDocument();
    expect(screen.getByTestId('github-milestone-selector')).toBeInTheDocument();
});

test('CreateIssueModal should update labels state when handleLabelsChange is called', async () => {
    render(<CreateIssueModal {...defaultProps}/>);

    const selectRepoButton = screen.getByText('Select Repo');
    await userEvent.click(selectRepoButton);

    expect(screen.getByTestId('github-label-selector')).toBeInTheDocument();
});

test('CreateIssueModal should update assignees state when handleAssigneesChange is called', async () => {
    render(<CreateIssueModal {...defaultProps}/>);

    const selectRepoButton = screen.getByText('Select Repo');
    await userEvent.click(selectRepoButton);

    expect(screen.getByTestId('github-assignee-selector')).toBeInTheDocument();
});

test('CreateIssueModal should set issueDescription state when post prop is updated', () => {
    const {rerender} = render(<CreateIssueModal {...defaultProps}/>);

    const post = {message: 'test post'};
    rerender(
        <CreateIssueModal
            {...defaultProps}
            post={post}
        />,
    );

    const descriptionInput = screen.getByRole('textbox', {name: ''});
    expect(descriptionInput).toHaveValue('test post');
});

test('CreateIssueModal should not display attribute selectors when repo does not have push permissions', async () => {
    render(<CreateIssueModal {...defaultProps}/>);

    const selectNoPushRepoButton = screen.getByText('Select No Push Repo');
    await userEvent.click(selectNoPushRepoButton);

    expect(screen.queryByTestId('github-label-selector')).not.toBeInTheDocument();
    expect(screen.queryByTestId('github-assignee-selector')).not.toBeInTheDocument();
    expect(screen.queryByTestId('github-milestone-selector')).not.toBeInTheDocument();
});

test('CreateIssueModal should display attribute selectors when repo has push permissions', async () => {
    render(<CreateIssueModal {...defaultProps}/>);

    const selectRepoButton = screen.getByText('Select Repo');
    await userEvent.click(selectRepoButton);

    expect(screen.getByTestId('github-label-selector')).toBeInTheDocument();
    expect(screen.getByTestId('github-assignee-selector')).toBeInTheDocument();
    expect(screen.getByTestId('github-milestone-selector')).toBeInTheDocument();
});
