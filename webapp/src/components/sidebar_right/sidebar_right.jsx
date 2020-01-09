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

export default class SidebarRight extends React.PureComponent {
    static propTypes = {
        username: PropTypes.string,
        org: PropTypes.string,
        enterpriseURL: PropTypes.string,
        reviews: PropTypes.arrayOf(PropTypes.object),
        unreads: PropTypes.arrayOf(PropTypes.object),
        yourPrs: PropTypes.arrayOf(PropTypes.object),
        yourPrsExtraInfo: PropTypes.arrayOf(PropTypes.object),
        yourAssignments: PropTypes.arrayOf(PropTypes.object),
        rhsState: PropTypes.string,
        theme: PropTypes.object.isRequired,
        actions: PropTypes.shape({
            getYourPrsExtraInfo: PropTypes.func.isRequired,
        }).isRequired,
    };

    componentDidMount() {
        if (this.props.yourPrs) {
            this.props.actions.getYourPrsExtraInfo(this.props.yourPrs.map((pr) => {
                return {url: pr.repository_url, number: pr.number};
            }));
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

            githubItems = this.props.yourPrs.map((pr) => {
                let extra;
                if (this.props.yourPrsExtraInfo) {
                    extra = this.props.yourPrsExtraInfo.find((extraInfo) => {
                        return (pr.repository_url === extraInfo.url) && (pr.number === extraInfo.number);
                    });
                }
                if (!extra) {
                    return pr;
                }

                return {
                    ...pr,
                    status: extra.status,
                    reviewers: extra.reviewers,
                    reviews: extra.reviews,
                };
            });

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
