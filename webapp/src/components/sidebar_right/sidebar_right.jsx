// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
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

function mapGithubItemListToPrList(gilist) {
    if (!gilist) {
        return [];
    }

    return gilist.map((pr) => {
        return {url: pr.repository_url, number: pr.number};
    });
}

function shouldUpdateDetails(prs, prevPrs, targetState, currentState, prevState) {
    if (currentState === targetState) {
        if (currentState !== prevState) {
            return true;
        }

        if (prs.length !== prevPrs.length) {
            return true;
        }

        for (let i = 0; i < prs.length; i++) {
            if (prs[i].id !== prevPrs[i].id) {
                return true;
            }
        }
    }

    return false;
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
            getYourPrsDetails: PropTypes.func.isRequired,
            getReviewsDetails: PropTypes.func.isRequired,
        }).isRequired,
    };

    componentDidMount() {
        if (this.props.yourPrs && this.props.rhsState === RHSStates.PRS) {
            this.props.actions.getYourPrsDetails(mapGithubItemListToPrList(this.props.yourPrs));
        }

        if (this.props.reviews && this.props.rhsState === RHSStates.REVIEWS) {
            this.props.actions.getReviewsDetails(mapGithubItemListToPrList(this.props.reviews));
        }
    }

    componentDidUpdate(prevProps) {
        if (shouldUpdateDetails(this.props.yourPrs, prevProps.yourPrs, RHSStates.PRS, this.props.rhsState, prevProps.rhsState)) {
            this.props.actions.getYourPrsDetails(mapGithubItemListToPrList(this.props.yourPrs));
        }

        if (shouldUpdateDetails(this.props.reviews, prevProps.reviews, RHSStates.REVIEWS, this.props.rhsState, prevProps.rhsState)) {
            this.props.actions.getReviewsDetails(mapGithubItemListToPrList(this.props.reviews));
        }
    }

    render() {
        const baseURL = this.props.enterpriseURL ? this.props.enterpriseURL : 'https://github.com';
        const orgQuery = this.props.org ? '+org%3A' + this.props.org : '';

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
                    </div>
                    <div>
                        <GithubItems
                            items={githubItems}
                            theme={this.props.theme}
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
};
