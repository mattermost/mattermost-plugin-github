// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useRef, useEffect} from 'react';
import PropTypes from 'prop-types';

import {Badge} from 'react-bootstrap';
import {makeStyleFromTheme, changeOpacity} from 'mattermost-redux/utils/theme_utils';

function GithubItems({items, theme, rhsState}) {
    const previous = usePrevious({rhsState, items}, {items: [], rhsState: ''});
    const style = getStyle(theme);

    let elements = items;

    if (previous.rhsState === rhsState) {
        const oldElements = previous.items.map((item) => {
            const newItem = items.find((i) => i.id === item.id);
            if (newItem) {
                return newItem;
            }
            return {...item, missing: true};
        });
        const newElements = items.filter((item) => !previous.items.find((i) => i.id === item.id));
        elements = [...oldElements, ...newElements];
    }

    if (elements.length === 0) {
        return (<div style={style.container}>{'You have no active items'}</div>);
    }

    return (elements.map((item) => {
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
    }));
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

function usePrevious(value, init) {
    const ref = useRef(init);
    useEffect(() => {
        ref.current = value;
    });
    return ref.current;
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

function areEquals(prevProps, nextProps) {
    return prevProps.theme === nextProps.theme &&
        prevProps.rhsState === nextProps.rhsState &&
        prevProps.items.reduce((acc, i) => `${acc}-${i.id}`, '') === nextProps.items.reduce((acc, i) => `${acc}-${i.id}`, '');
}

export default React.memo(GithubItems, areEquals);
