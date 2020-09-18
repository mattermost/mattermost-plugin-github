// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import PropTypes from 'prop-types';

import {Badge, Tooltip, OverlayTrigger} from 'react-bootstrap';
import {makeStyleFromTheme, changeOpacity} from 'mattermost-redux/utils/theme_utils';

import {formatTimeSince} from 'utils/date_utils';

import CrossIcon from 'images/icons/cross.jsx';
import DotIcon from 'images/icons/dot.jsx';
import TickIcon from 'images/icons/tick.jsx';
import SignIcon from 'images/icons/sign.jsx';
import ChangesRequestedIcon from 'images/icons/changes_requested.jsx';
import {getLabelFontColor} from '../../utils/styles';

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
        let number = null;

        if (item.number) {
            number = (
                <strong>
                    <i className='fa fa-code-fork'/>{' #' + item.number}
                </strong>);
        }

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
            if (item.number) {
                number = (
                    <strong>
                        <a
                            href={item.html_url}
                            target='_blank'
                            rel='noopener noreferrer'
                        >
                            <i className='fa fa-code-fork'/>{' #' + item.number}
                        </a>
                    </strong>);
            }
        }

        let milestone = '';
        if (item.milestone) {
            milestone = (
                <span>
                    <div
                        style={
                            {
                                ...style.milestoneIcon,
                                ...((item.created_at || userName) && {paddingLeft: 10}),
                            }
                        }
                    ><SignIcon/></div>
                    {' '}
                    {item.milestone.title}
                </span>);
        }

        let reviews = '';

        if (item.reviews) {
            reviews = getReviewText(item, style, (item.created_at || userName || milestone));
        }

        let status = '';

        // Status images pasted directly from GitHub. Change to our own version when styles are decided.
        if (item.status) {
            switch (item.status) {
            case 'success':
                status = (<div style={{...style.icon, ...style.iconSucess}}><TickIcon/></div>);
                break;
            case 'pending':
                status = (<div style={{...style.icon, ...style.iconPending}}><DotIcon/></div>);
                break;
            default:
                status = (<div style={{...style.icon, ...style.iconFailed}}><CrossIcon/></div>);
            }
        }

        let hasConflict = '';
        if (item.mergeable != null && !item.mergeable) {
            hasConflict = (
                <OverlayTrigger
                    key='githubRHSPRMergeableIndicator'
                    placement='top'
                    overlay={
                        <Tooltip id='githubRHSPRMergeableTooltip'>
                            {'This pull request has conflicts that must be resolved'}
                        </Tooltip>
                    }
                >
                    <i
                        style={style.conflictIcon}
                        className='icon icon-alert-outline'
                    />
                </OverlayTrigger>
            );
        }

        return (
            <div
                key={item.id}
                style={style.container}
            >
                <div>
                    <strong>
                        {title}{hasConflict}{status}
                    </strong>
                </div>
                <div>
                    {number} <span className='light'>{'(' + repoName + ')'}</span>
                </div>
                <GithubLabels labels={item.labels}/>
                <div
                    className='light'
                    style={style.subtitle}
                >
                    {item.created_at && ('Opened ' + formatTimeSince(item.created_at) + ' ago')}
                    {userName && ' by ' + userName}
                    {(item.created_at || userName) && '.'}
                    {milestone}
                    {item.reason ? (<React.Fragment>
                        {(item.created_at || userName || milestone) && (<br/>)}
                        {notificationReasons[item.reason]}
                    </React.Fragment>) : null }
                </div>
                {reviews}
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
        subtitleSecondLine: {
            fontSize: '13px',
        },
        icon: {
            top: 3,
            position: 'relative',
            left: 3,
            height: 18,
            display: 'inline-flex',
            alignItems: 'center',
        },
        iconSucess: {
            fill: theme.onlineIndicator,
        },
        iconPending: {
            fill: theme.awayIndicator,
        },
        iconFailed: {
            fill: theme.dndIndicator,
        },
        iconChangesRequested: {
            fill: theme.dndIndicator,
        },
        conflictIcon: {
            color: theme.dndIndicator,
        },
        milestoneIcon: {
            top: 3,
            position: 'relative',
            height: 18,
            display: 'inline-flex',
            alignItems: 'center',
            fill: theme.centerChannelColor,
        },
    };
});

function GithubLabels(props) {
    return props.labels ? props.labels.map((label) => {
        return (
            <Badge
                key={label.id}
                style={{...itemStyle.label, ...{backgroundColor: `#${label.color}`, color: getLabelFontColor(label.color)}}}
            >{label.name}</Badge>
        );
    }) : null;
}

function getReviewText(item, style, secondLine) {
    let reviews = '';
    let changes = '';

    const finishedReviewers = [];

    const reverse = (accum, cur) => {
        accum.unshift(cur);
        return accum;
    };

    const lastReviews = item.reviews.reduce(reverse, []).filter((v) => {
        if (v.user.login === item.user.login) {
            return false;
        }

        if (item.requestedReviewers.includes(v.user.login)) {
            return false;
        }

        if (v.state === 'COMMENTED' || v.state === 'DISMISSED') {
            return false;
        }

        if (finishedReviewers.includes(v.user.login)) {
            return false;
        }

        finishedReviewers.push(v.user.login);
        return true;
    });

    const approved = lastReviews.reduce((accum, cur) => {
        if (cur.state === 'APPROVED') {
            return accum + 1;
        }
        return accum;
    }, 0);

    const changesRequested = lastReviews.reduce((accum, cur) => {
        if (cur.state === 'CHANGES_REQUESTED') {
            return accum + 1;
        }
        return accum;
    }, 0);

    const totalReviewers = finishedReviewers.length + item.requestedReviewers.length;
    if (totalReviewers > 0) {
        let reviewName;
        if (totalReviewers === 1) {
            reviewName = 'review';
        } else {
            reviewName = 'reviews';
        }
        reviews = (<span className='light'>{approved + ' out of ' + totalReviewers + ' ' + reviewName + ' complete.'}</span>);
    }

    if (changesRequested > 0) {
        changes = (
            <OverlayTrigger
                key='changesRequestedDot'
                placement='bottom'
                overlay={<Tooltip id='changesRequestedTooltip'>{'Changes Requested'}</Tooltip>}
            >
                <div style={{...style.icon, ...style.iconChangesRequested}}><ChangesRequestedIcon/></div>
            </OverlayTrigger>
        );
    }

    return (
        <div
            style={secondLine ? style.subtitleSecondLine : style.subtitle}
        >
            {reviews} {changes}
        </div>);
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
