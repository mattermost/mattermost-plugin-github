import React from 'react';
import {act, render} from '@testing-library/react';

import GithubRepoSelector from './github_repo_selector';

const selectProps = {};

jest.mock('@/components/react_select_setting', () => ({
    __esModule: true,
    default: ({name, onChange}) => {
        selectProps[name] = {onChange};
        return <div/>;
    },
}));

const baseProps = {
    yourOrgs: [{login: 'acme'}, {login: 'globex'}],
    yourReposByOrg: {repos: []},
    theme: {},
    onChange: jest.fn(),
    value: 'acme/widgets',
    currentChannelId: 'channel-id',
    actions: {
        getOrgs: jest.fn(),
        getReposByOrg: jest.fn(),
    },
};

beforeEach(() => {
    jest.clearAllMocks();
});

test('picking an organization loads that organization\'s repositories', () => {
    render(<GithubRepoSelector {...baseProps}/>);

    act(() => selectProps.org.onChange('org', 'globex'));

    expect(baseProps.actions.getReposByOrg).toHaveBeenCalledWith('globex', 'channel-id');
    expect(baseProps.onChange).toHaveBeenCalledWith(null);
});

test('a new organization list selects its first organization', () => {
    const {rerender} = render(<GithubRepoSelector {...baseProps}/>);

    rerender(
        <GithubRepoSelector
            {...baseProps}
            yourOrgs={[{login: 'initech'}, {login: 'acme'}]}
        />,
    );

    expect(baseProps.actions.getReposByOrg).toHaveBeenCalledWith('initech', 'channel-id');
});
