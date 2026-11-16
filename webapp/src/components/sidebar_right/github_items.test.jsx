import React from 'react';
import {render, screen} from '@testing-library/react';

import GithubItems from './github_items';

const theme = {
    centerChannelBg: '#ffffff',
    centerChannelColor: '#333333',
    buttonBg: '#1c58d9',
    buttonColor: '#ffffff',
    linkColor: '#1c58d9',
    mentionHighlightBg: '#fff2bb',
    mentionHighlightLink: '#2f81b7',
    onlineIndicator: '#3db887',
    awayIndicator: '#ffbc42',
    dndIndicator: '#d24b4e',
};

test('renders an item without html_url instead of throwing', () => {
    const item = {
        id: 1,
        number: 970,
        title: 'Panic when loading lhs-content',
        html_url: '',
        repository_url: 'https://github.com/mattermost/mattermost-plugin-github',
        state: 'open',
    };

    render(
        <GithubItems
            items={[item]}
            theme={theme}
        />,
    );

    expect(screen.getByText('#970')).toBeInTheDocument();
    expect(screen.getByText('Panic when loading lhs-content')).toBeInTheDocument();
});
