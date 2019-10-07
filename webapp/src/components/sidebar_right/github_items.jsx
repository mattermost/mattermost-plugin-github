// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import PropTypes from 'prop-types';

import {Badge} from 'react-bootstrap';
import {makeStyleFromTheme, changeOpacity} from 'mattermost-redux/utils/theme_utils';

//retain previous item to show deleted items between 2 refresh call
let previousItems = [];
let previousStateRhs;

function GithubItems(props) {
    const style = getStyle(props.theme);
    let oldElements = [];
    let newElements = [];

    if (previousStateRhs === props.rhsState) {
        oldElements = previousItems.map((item) => {
            if (!props.items.find((i) => i.id === item.id)) {
                return {...item, missing: true};
            }
            return item;
        });
        newElements = props.items.filter((item) => !previousItems.find((i) => i.id === item.id));
    } else {
        previousStateRhs = props.rhsState;
        newElements = [...props.items];
    }

    const elements = [...oldElements, ...newElements];
    previousItems = [...elements];

    return elements.length > 0 ? elements.map((item) => {
        const repoName = item.repository_url ? item.repository_url.replace(/.+\/repos\//, '') : item.repository.full_name;

        let userName = null;

        if (item.user) {
            userName = item.user.login;
        } else if (item.owner) {
            userName = item.owner.login;
        }

        let title = item.title ? item.title : item.subject.title;

        if (item.html_url) {
            title = (
                <a
                    href={item.html_url}
                    target='_blank'
                    rel='noopener noreferrer'
                    style={style.itemTitle}
                >
                    {item.title ? item.title : item.subject.title}
                </a>);
        }

        return (
            <div
                key={item.id}
                style={item.missing ? style.containerDimmed : style.container}
            >
                <div>
                    <strong>
                        {title}
                    </strong>
                </div>
                <GithubLabels labels={item.labels}/>
                <div
                    className='light'
                    style={style.subtitle}
                >
                    {userName ? 'Created by ' + userName + ' ' : ''}
                    {'at ' + repoName + '.'}
                    {item.reason ?
                        (<React.Fragment>
                            <br/>
                            {notificationReasons[item.reason]}
                        </React.Fragment>) : null }
                </div>
            </div>
        );
    }) : <div style={style.container}>{'You have no active items'}</div>;
}

GithubItems.propTypes = {
    items: PropTypes.array.isRequired,
    theme: PropTypes.object.isRequired,
    rhsState: PropTypes.string.isRequired,
};

const getStyle = makeStyleFromTheme((theme) => {
    return {
        container: {
            padding: '15px',
            borderTop: `1px solid ${changeOpacity(theme.centerChannelColor, 0.2)}`,
        },
        containerDimmed: {
            padding: '15px',
            borderTop: `1px solid ${changeOpacity(theme.centerChannelColor, 0.2)}`,
            opacity: 0.5,
        },
        itemTitle: {
            color: theme.centerChannelColor,
            lineHeight: 1.7,
            fontWeight: 'bold',
        },
        subtitle: {
            margin: '5px 0 0 0',
            fontSize: '13px',
        },
    };
});

function GithubLabels(props) {
    return props.labels ? props.labels.map((label) => {
        return (
            <Badge
                key={label.id}
                style={{...itemStyle.label, ...{backgroundColor: `#${label.color}`}}}
            >{label.name}</Badge>
        );
    }) : null;
}

GithubLabels.propTypes = {
    labels: PropTypes.array,
};

const itemStyle = {
    label: {
        margin: '4px 5px 0 0',
        padding: '3px 8px',
        display: 'inline-flex',
        borderRadius: '3px',
        position: 'relative',
    },
};

const notificationReasons = {
    assign:	'You were assigned to the issue',
    author:	'You created the thread.',
    comment:	'You commented on the thread.',
    invitation:	'You accepted an invitation to contribute to the repository.',
    manual:	'You subscribed to the thread (via an issue or pull request).',
    mention:	'You were specifically @mentioned in the content.',
    review_requested:	'You were requested to review a pull request.',
    security_alert: 'GitHub discovered a security vulnerability in your repository.',
    state_change: 'You changed the thread state.',
    subscribed:	'You are watching the repository.',
    team_mention:	'You were on a team that was mentioned.',
};

export default GithubItems;
