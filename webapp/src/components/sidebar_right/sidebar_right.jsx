// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {Tooltip, OverlayTrigger} from 'react-bootstrap';
import PropTypes from 'prop-types';
import Scrollbars from 'react-custom-scrollbars';

import {RHSStates} from '../../constants';

import GithubItems from './github_items';

export function renderView(props) {
    return (
        <div
            {...props}
            className='scrollbar--view'
        />);
}

export function renderThumbHorizontal(props) {
    return (
        <div
            {...props}
            className='scrollbar--horizontal'
        />);
}

export function renderThumbVertical(props) {
    return (
        <div
            {...props}
            className='scrollbar--vertical'
        />);
}

export default class SidebarRight extends React.PureComponent {
    static propTypes = {
        username: PropTypes.string,
        org: PropTypes.string,
        enterpriseURL: PropTypes.string,
        reviews: PropTypes.arrayOf(PropTypes.object),
        unreads: PropTypes.arrayOf(PropTypes.object),
        yourPrs: PropTypes.arrayOf(PropTypes.object),
        yourAssignments: PropTypes.arrayOf(PropTypes.object),
        rhsState: PropTypes.string,
        theme: PropTypes.object.isRequired,
        actions: PropTypes.shape({
            getReviews: PropTypes.func.isRequired,
            getUnreads: PropTypes.func.isRequired,
            getYourPrs: PropTypes.func.isRequired,
            getYourAssignments: PropTypes.func.isRequired,
        }).isRequired,
    };

    constructor(props) {
        super(props);

        this.state = {
            refreshing: false,
        };
    }

    refresh = async (e) => {
        if (this.state.refreshing) {
            return;
        }

        if (e) {
            e.preventDefault();
        }

        this.setState({refreshing: true});
        let refreshAction;
        switch (this.props.rhsState) {
        case RHSStates.PRS:
            refreshAction = this.props.actions.getYourPrs;
            break;
        case RHSStates.REVIEWS:
            refreshAction = this.props.actions.getReviews;
            break;
        case RHSStates.UNREADS:
            refreshAction = this.props.actions.getUnreads;
            break;
        case RHSStates.ASSIGNMENTS:
            refreshAction = this.props.actions.getYourAssignments;
            break;
        default:
            return;
        }
        await refreshAction();
        this.setState({refreshing: false});
    }

    render() {
        const baseURL = this.props.enterpriseURL ? this.props.enterpriseURL : 'https://github.com';
        const orgQuery = this.props.org ? '+org%3A' + this.props.org : '';
        const refreshClass = this.state.refreshing ? ' fa-spin' : '';

        let title = '';
        let githubItems = [];
        let listUrl = '';

        switch (this.props.rhsState) {
        case RHSStates.PRS:

            githubItems = this.props.yourPrs;
            title = 'Your Open Pull Requests';
            listUrl = baseURL + '/pulls?q=is%3Aopen+is%3Apr+author%3A' + this.props.username + '+archived%3Afalse' + orgQuery;

            break;
        case RHSStates.REVIEWS:

            githubItems = this.props.reviews;
            listUrl = baseURL + '/pulls?q=is%3Aopen+is%3Apr+review-requested%3A' + this.props.username + '+archived%3Afalse' + orgQuery;
            title = 'Pull Requests Needing Review';

            break;
        case RHSStates.UNREADS:

            githubItems = this.props.unreads;
            title = 'Unread Messages';
            listUrl = baseURL + '/notifications';

            break;
        case RHSStates.ASSIGNMENTS:

            githubItems = this.props.yourAssignments;
            title = 'Your Assignments';
            listUrl = baseURL + '/pulls?q=is%3Aopen+archived%3Afalse+assignee%3A' + this.props.username + orgQuery;

            break;
        default:
            break;
        }

        return (
            <React.Fragment>
                <Scrollbars
                    autoHide={true}
                    autoHideTimeout={500}
                    autoHideDuration={500}
                    renderThumbHorizontal={renderThumbHorizontal}
                    renderThumbVertical={renderThumbVertical}
                    renderView={renderView}
                >
                    <div style={style.sectionHeader}>
                        <strong>
                            <a
                                href={listUrl}
                                target='_blank'
                                rel='noopener noreferrer'
                            >{title}</a>
                        </strong>
                        <OverlayTrigger
                            key='githubRefreshButton'
                            placement='bottom'
                            overlay={<Tooltip id='refreshTooltip'>Refresh</Tooltip>}
                        >
                            <button
                                style={style.refresh}
                                onClick={this.refresh}
                            >
                                <i className={'fa fa-refresh' + refreshClass}/>
                            </button>
                        </OverlayTrigger>
                    </div>
                    <div>
                        <GithubItems
                            items={githubItems}
                            theme={this.props.theme}
                            rhsState={this.props.rhsState}
                        />
                    </div>
                </Scrollbars>
            </React.Fragment>
        );
    }
}

const style = {
    sectionHeader: {
        padding: '15px',
    },
    refresh: {
        float: 'right',
        border: 'none',
        background: 'none',
    },
};
