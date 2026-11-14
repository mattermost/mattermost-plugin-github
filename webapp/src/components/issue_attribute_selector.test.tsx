// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {render, screen, waitFor} from '@testing-library/react';

import IssueAttributeSelector, {Props} from './issue_attribute_selector';

const defaultProps: Props = {
    isMulti: false,
    repoName: 'mattermost/mattermost-plugin-github',
    theme: {
        centerChannelColor: '#000',
        centerChannelBg: '#fff',
    } as any,
    onChange: jest.fn(),
    loadOptions: jest.fn(() => Promise.resolve([
        {label: 'Bug', value: 'bug'},
        {label: 'Enhancement', value: 'enhancement'},
    ])),
    selection: null,
};

describe('IssueAttributeSelector', () => {
    beforeEach(() => {
        jest.clearAllMocks();
    });

    test('should disable underlying ReactSelect control when isDisabled is true', () => {
        const {container} = render(
            <IssueAttributeSelector
                {...defaultProps}
                isDisabled={true}
            />,
        );

        const input = container.querySelector('input');
        expect(input).toBeInTheDocument();
        expect(input).toBeDisabled();

        const control = container.querySelector('[aria-disabled="true"]');
        expect(control).toBeInTheDocument();
    });

    test('should enable underlying ReactSelect control when isDisabled is false', () => {
        const {container} = render(
            <IssueAttributeSelector
                {...defaultProps}
                isDisabled={false}
            />,
        );

        const input = container.querySelector('input');
        expect(input).toBeInTheDocument();
        expect(input).not.toBeDisabled();

        const disabledControl = container.querySelector('[aria-disabled="true"]');
        expect(disabledControl).not.toBeInTheDocument();
    });

    test('should discard stale options when repoName changes before loadOptions resolves', async () => {
        let resolveFirstRepo: (value: any) => void;
        const firstLoadPromise = new Promise((resolve) => {
            resolveFirstRepo = resolve;
        });

        const loadOptions = jest.fn().
            mockImplementationOnce(() => firstLoadPromise).
            mockImplementationOnce(() => Promise.resolve([
                {label: 'RepoB-Label', value: 'repob-label'},
            ]));

        const {rerender} = render(
            <IssueAttributeSelector
                {...defaultProps}
                repoName='repoA'
                loadOptions={loadOptions}
            />,
        );

        rerender(
            <IssueAttributeSelector
                {...defaultProps}
                repoName='repoB'
                loadOptions={loadOptions}
            />,
        );

        // Resolve the first loadOptions after repo has already changed to repoB
        resolveFirstRepo!([
            {label: 'RepoA-Label', value: 'repoa-label'},
        ]);

        await waitFor(() => {
            expect(loadOptions).toHaveBeenCalledTimes(2);
        });

        // The stale options from RepoA should have been discarded, only RepoB options remain
        expect(screen.queryByText('RepoA-Label')).not.toBeInTheDocument();
    });

    test('should discard earlier options on rapid A -> B -> A repo transitions', async () => {
        let resolveFirstA: (value: any) => void;
        const firstAPromise = new Promise((resolve) => {
            resolveFirstA = resolve;
        });

        const loadOptions = jest.fn().
            mockImplementationOnce(() => firstAPromise).
            mockImplementationOnce(() => Promise.resolve([{label: 'RepoB-Label', value: 'b'}])).
            mockImplementationOnce(() => Promise.resolve([{label: 'RepoA-Latest-Label', value: 'a-latest'}]));

        const onChange = jest.fn();

        const {rerender} = render(
            <IssueAttributeSelector
                {...defaultProps}
                isMulti={true}
                selection={['a-latest']}
                onChange={onChange}
                repoName='repoA'
                loadOptions={loadOptions}
            />,
        );

        rerender(
            <IssueAttributeSelector
                {...defaultProps}
                isMulti={true}
                selection={['a-latest']}
                onChange={onChange}
                repoName='repoB'
                loadOptions={loadOptions}
            />,
        );

        rerender(
            <IssueAttributeSelector
                {...defaultProps}
                isMulti={true}
                selection={['a-latest']}
                onChange={onChange}
                repoName='repoA'
                loadOptions={loadOptions}
            />,
        );

        // Resolve the first repoA request after switching back to repoA
        resolveFirstA!([{label: 'RepoA-Stale-Label', value: 'a-latest'}]);

        await waitFor(() => {
            expect(loadOptions).toHaveBeenCalledTimes(3);
        });

        // The latest load should apply matching options, never the stale one
        await waitFor(() => {
            expect(onChange).toHaveBeenLastCalledWith([
                {label: 'RepoA-Latest-Label', value: 'a-latest'},
            ]);
        });
    });

    test('should clear options while loading after a repo change', async () => {
        let resolveSecondRepo: (value: any) => void;
        const secondLoadPromise = new Promise((resolve) => {
            resolveSecondRepo = resolve;
        });

        const loadOptions = jest.fn().
            mockImplementationOnce(() => Promise.resolve([
                {label: 'RepoA-Label', value: 'repoa-label'},
            ])).
            mockImplementationOnce(() => secondLoadPromise);

        const {rerender} = render(
            <IssueAttributeSelector
                {...defaultProps}
                repoName='repoA'
                loadOptions={loadOptions}
            />,
        );

        // Wait for first repo options to load and be present
        await waitFor(() => {
            expect(loadOptions).toHaveBeenCalledTimes(1);
        });

        // Change repository to repoB, triggering a pending loadOptions
        rerender(
            <IssueAttributeSelector
                {...defaultProps}
                repoName='repoB'
                loadOptions={loadOptions}
            />,
        );

        // While repoB options are still loading, repoA options should be cleared immediately
        expect(screen.queryByText('RepoA-Label')).not.toBeInTheDocument();

        // Resolve second repo options
        resolveSecondRepo!([
            {label: 'RepoB-Label', value: 'repob-label'},
        ]);

        await waitFor(() => {
            expect(loadOptions).toHaveBeenCalledTimes(2);
        });
    });
});
