// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import PropTypes from 'prop-types';
import Scrollbars from 'react-custom-scrollbars-2';
import ReactSelect from 'react-select';

import {makeStyleFromTheme} from 'mattermost-redux/utils/theme_utils';

import {RHSStates} from '../../constants';
import {getStyleForReactSelect} from '@/utils/styles';

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

function getRepoName(item) {
    if (item.repository_url) {
        return item.repository_url.replace(/.+\/repos\//, '');
    }
    return item.repository?.full_name ?? null;
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
        this.state = {selectedRepo: ''};
    }

    handleRepoFilterChange = (selectedOption) => {
        this.setState({selectedRepo: selectedOption ? selectedOption.value : ''});
    }

    componentDidMount() {
        if (this.props.yourPrs && this.props.rhsState === RHSStates.PRS) {
            this.props.actions.getYourPrsDetails(mapGithubItemListToPrList(this.props.yourPrs));
        }

        if (this.props.reviews && this.props.rhsState === RHSStates.REVIEWS) {
            this.props.actions.getReviewsDetails(mapGithubItemListToPrList(this.props.reviews));
        }
    }

    componentDidUpdate(prevProps) {
        // Reset repo filter when switching RHS tabs
        if (prevProps.rhsState !== this.props.rhsState) {
            this.setState({selectedRepo: ''}); // eslint-disable-line react/no-did-update-set-state
        }

        // Validate selectedRepo against the current list — reset if the repo
        // is no longer present (stale state when the underlying data changes)
        if (this.state.selectedRepo) {
            const currentRepos = [...new Set(
                this.getCurrentItems().map(getRepoName).filter(Boolean),
            )];
            if (!currentRepos.includes(this.state.selectedRepo)) {
                this.setState({selectedRepo: ''}); // eslint-disable-line react/no-did-update-set-state
            }
        }

        if (shouldUpdateDetails(this.props.yourPrs, prevProps.yourPrs, RHSStates.PRS, this.props.rhsState, prevProps.rhsState)) {
            this.props.actions.getYourPrsDetails(mapGithubItemListToPrList(this.props.yourPrs));
        }

        if (shouldUpdateDetails(this.props.reviews, prevProps.reviews, RHSStates.REVIEWS, this.props.rhsState, prevProps.rhsState)) {
            this.props.actions.getReviewsDetails(mapGithubItemListToPrList(this.props.reviews));
        }
    }

    getCurrentItems = () => {
        const {yourPrs, reviews, unreads, yourAssignments, rhsState} = this.props;
        switch (rhsState) {
        case RHSStates.PRS:
            return yourPrs || [];
        case RHSStates.REVIEWS:
            return reviews || [];
        case RHSStates.UNREADS:
            return unreads || [];
        case RHSStates.ASSIGNMENTS:
            return yourAssignments || [];
        default:
            return [];
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

        // Extract unique repo names for filter dropdown
        const repoNames = [...new Set(
            githubItems.map(getRepoName).filter(Boolean),
        )].sort();

        // Build react-select options
        const repoOptions = [
            {value: '', label: 'All repositories'},
            ...repoNames.map((repo) => ({value: repo, label: repo})),
        ];

        // Filter items by selected repo
        const {selectedRepo} = this.state;
        const selectedOption = repoOptions.find((opt) => opt.value === selectedRepo) || repoOptions[0];
        const filteredItems = selectedRepo ?
            githubItems.filter((item) => getRepoName(item) === selectedRepo) :
            githubItems;

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
                        {repoNames.length > 1 && (
                            <div style={style.repoFilterContainer}>
                                <ReactSelect
                                    value={selectedOption}
                                    onChange={this.handleRepoFilterChange}
                                    options={repoOptions}
                                    styles={getStyleForReactSelect(this.props.theme)}
                                    aria-label='Filter by repository'
                                    isSearchable={false}
                                    menuPortalTarget={document.body}
                                    menuPlacement='auto'
                                />
                            </div>
                        )}
                    </div>
                    <div>
                        <GithubItems
                            items={filteredItems}
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

const getStyle = makeStyleFromTheme((theme) => {
    return {
        sectionHeader: {
            padding: '15px',
        },
        repoFilterContainer: {
            marginLeft: '8px',
            minWidth: '160px',
            maxWidth: '220px',
            fontSize: '12px',
        },
    };
});
