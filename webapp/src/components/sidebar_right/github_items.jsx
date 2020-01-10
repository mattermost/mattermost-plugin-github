// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import PropTypes from 'prop-types';

import {Badge} from 'react-bootstrap';
import {makeStyleFromTheme, changeOpacity} from 'mattermost-redux/utils/theme_utils';

import {formatTimeSince} from 'utils/date_utils';

import CrossIcon from 'images/icons/cross.jsx';
import DotIcon from 'images/icons/dot.jsx';
import TickIcon from 'images/icons/tick.jsx';

function GithubItems(props) {
    const style = getStyle(props.theme);

    return props.items.length > 0 ? props.items.map((item) => {
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

        let milestone = '';
        if (item.milestone) {
            milestone = (<span><i className='fas fa-bullseye'/>{item.milestone.title}</span>);
        }

        let reviewers = '';
        if (item.reviewers && item.reviewers > 0) {
            reviewers = (<span>{item.reviewers} pending reviews.</span>);
        }

        let reviews = '';
        if (item.reviews) {
            const reviewerUsers = [];
            const filteredReviews = item.reviews.filter((v) => {
                if (!reviewerUsers.includes(v.user.login)) {
                    reviewerUsers.push(v.user.login);
                    return true;
                }
                return false;
            });

            const approved = filteredReviews.reduce((accum, cur) => {
                if (cur.state === 'APPROVED') {
                    return accum + 1;
                }
                return accum;
            }, 0);
            const changesRequested = filteredReviews.reduce((accum, cur) => {
                if (cur.state === 'CHANGES_REQUESTED') {
                    return accum + 1;
                }
                return accum;
            }, 0);

            if (changesRequested > 0) {
                reviews = (<span>Changes requested.</span>);
            } else if (approved > 0) {
                if (approved === 1) {
                    reviews = (<span>{approved} approved review.</span>);
                } else {
                    reviews = (<span>{approved} approved reviews.</span>);
                }
            }
        }

        let status = '';

        // Status images pasted directly from GitHub. Change to our own version when styles are decided.
        if (item.status) {
            switch (item.status) {
            case 'success':
                status = (<TickIcon />);
                break;
            case 'pending':
                status = (<DotIcon />);
                break;
            default:
                status = (<CrossIcon />);
            }
        }

        return (
            <div
                key={item.id}
                style={style.container}
            >
                <div>
                    <strong>
                        {title}{status}
                    </strong>
                </div>
                <div>
                    <strong><i className='fa fa-code-fork'/> #{item.number}</strong> <span className='light'>{repoName}</span>
                </div>
                <GithubLabels labels={item.labels}/>
                <div
                    className='light'
                    style={style.subtitle}
                >
                    {'Opened ' + formatTimeSince(item.created_at) + ' ago'}
                    {userName ? ' by ' + userName : '.'}
                    {item.reason ?
                        (<React.Fragment>
                            <br/>
                            {notificationReasons[item.reason]}
                        </React.Fragment>) : null }
                    {milestone}
                </div>
                <div>{reviews} {reviewers}</div>
            </div>
        );
    }) : <div style={style.container}>{'You have no active items'}</div>;
}

GithubItems.propTypes = {
    items: PropTypes.array.isRequired,
    theme: PropTypes.object.isRequired,
};

const getStyle = makeStyleFromTheme((theme) => {
    return {
        container: {
            padding: '15px',
            borderTop: `1px solid ${changeOpacity(theme.centerChannelColor, 0.2)}`,
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
    labels: PropTypes.array.isRequired,
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
