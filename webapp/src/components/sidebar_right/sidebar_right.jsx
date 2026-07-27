// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import PropTypes from 'prop-types';
import Scrollbars from 'react-custom-scrollbars-2';

import {makeStyleFromTheme} from 'mattermost-redux/utils/theme_utils';

import {RHSStates} from '../../constants';

import GithubItems from './github_items';
import SortDropdown from './sort_dropdown';

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
        orgs: PropTypes.array.isRequired,
        enterpriseURL: PropTypes.string,
        reviews: PropTypes.arrayOf(PropTypes.object),
        unreads: PropTypes.arrayOf(PropTypes.object),
        yourPrs: PropTypes.arrayOf(PropTypes.object),
        yourAssignments: PropTypes.arrayOf(PropTypes.object),
        rhsState: PropTypes.string,
        reviewTargetDays: PropTypes.number,
        theme: PropTypes.object.isRequired,
        actions: PropTypes.shape({
            getYourPrsDetails: PropTypes.func.isRequired,
            getReviewsDetails: PropTypes.func.isRequired,
        }).isRequired,
    };

    constructor(props) {
        super(props);
        this.state = {sortBy: 'created'};
    }

    handleSortChange = (selectedOption) => {
        this.setState({sortBy: selectedOption ? selectedOption.value : 'created'});
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
        let orgQuery = '';
        this.props.orgs.map((org) => {
            orgQuery += ('+org%3A' + org);
            return orgQuery;
        });
        const {yourPrs, reviews, unreads, yourAssignments, username, rhsState} = this.props;

        let title = '';
        let githubItems = [];
        let listUrl = '';

        switch (rhsState) {
        case RHSStates.PRS:

            githubItems = yourPrs;
            title = 'Your Open Pull Requests';
            listUrl = baseURL + '/pulls?q=is%3Aopen+is%3Apr+author%3A' + username + '+archived%3Afalse' + orgQuery;

            break;
        case RHSStates.REVIEWS:

            githubItems = reviews;
            listUrl = baseURL + '/pulls?q=is%3Aopen+is%3Apr+review-requested%3A' + username + '+archived%3Afalse' + orgQuery;
            title = 'Pull Requests Needing Review';

            break;
        case RHSStates.UNREADS:

            githubItems = unreads;
            title = 'Unread Messages';
            listUrl = baseURL + '/notifications';
            break;
        case RHSStates.ASSIGNMENTS:

            githubItems = yourAssignments;
            title = 'Your Assignments';
            listUrl = baseURL + '/pulls?q=is%3Aopen+archived%3Afalse+assignee%3A' + username + orgQuery;
            break;
        default:
            break;
        }

        // Sort items by selected criteria
        // Notification threads (UNREADS) don't have created_at, so fall back
        // to updated_at when created_at is undefined.
        const {sortBy} = this.state;
        const sortedItems = (githubItems || []).slice().sort((a, b) => {
            const dateA = sortBy === 'updated' ? a.updated_at : (a.created_at || a.updated_at);
            const dateB = sortBy === 'updated' ? b.updated_at : (b.created_at || b.updated_at);
            return new Date(dateB) - new Date(dateA);
        });

        // Build sort options
        const sortOptions = [
            {value: 'created', label: 'Recently Created'},
            {value: 'updated', label: 'Recently Updated'},
        ];
        const selectedSortOption = sortOptions.find((opt) => opt.value === sortBy) || sortOptions[0];

        const style = getStyle(this.props.theme);

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
                        <div style={style.sortDropdownContainer}>
                            <SortDropdown
                                value={selectedSortOption}
                                onChange={this.handleSortChange}
                                options={sortOptions}
                                theme={this.props.theme}
                                ariaLabel='Sort items'
                            />
                        </div>
                    </div>
                    <div>
                        <GithubItems
                            items={sortedItems}
                            theme={this.props.theme}
                            showReviewSLA={rhsState === RHSStates.REVIEWS}
                            reviewTargetDays={this.props.reviewTargetDays || 0}
                        />
                    </div>
                </Scrollbars>
            </React.Fragment>
        );
    }
}

const getStyle = makeStyleFromTheme(() => {
    return {
        sectionHeader: {
            padding: '15px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            flexWrap: 'wrap',
        },
        sortDropdownContainer: {
            marginLeft: '8px',
            width: 'fit-content',
            fontSize: '12px',
        },
    };
});