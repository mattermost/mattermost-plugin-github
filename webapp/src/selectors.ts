// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {getConfig} from 'mattermost-redux/selectors/entities/general';

import {createSelector} from 'reselect';

import {GlobalState, PluginState} from './types/store';
import {GithubIssueData, SidebarData, PrsDetailsData, UnreadsData, ReviewThreadData} from './types/github_types';

const emptyArray: GithubIssueData[] | UnreadsData[] = [];

export const getPluginState = (state: GlobalState): PluginState => state['plugins-github'];

export const getServerRoute = (state: GlobalState) => {
    const config = getConfig(state as any);
    let basePath = '';
    if (config && config.SiteURL) {
        basePath = new URL(config.SiteURL).pathname;
        if (basePath && basePath[basePath.length - 1] === '/') {
            basePath = basePath.substr(0, basePath.length - 1);
        }
    }

    return basePath;
};

function mapPrsToDetails(prs: GithubIssueData[], details: PrsDetailsData[]) {
    if (!prs) {
        return [];
    }

    return prs.map((pr: GithubIssueData) => {
        let foundDetails;
        if (details) {
            foundDetails = details.find((prDetails: PrsDetailsData) => {
                return (pr.repository_url === prDetails.url) && (pr.number === prDetails.number);
            });
        }
        if (!foundDetails) {
            return pr;
        }

        return {
            ...pr,
            status: foundDetails.status,
            mergeable: foundDetails.mergeable,
            requestedReviewers: foundDetails.requestedReviewers,
            reviews: foundDetails.reviews,
        };
    });
}

export const getSidebarData = createSelector(
    getPluginState,
    (pluginState): SidebarData => {
        const {username, sidebarContent, reviewDetails, yourPrDetails, organizations, rhsState} = pluginState;
        return {
            username,
            reviews: mapPrsToDetails(sidebarContent.reviews || emptyArray, reviewDetails),
            yourPrs: mapPrsToDetails(sidebarContent.prs || emptyArray, yourPrDetails),
            yourAssignments: sidebarContent.assignments || emptyArray,
            unreads: sidebarContent.unreads || emptyArray,
            orgs: organizations,
            rhsState,
        };
    },
);

export const configuration = (state: GlobalState) => getPluginState(state).configuration;

export const getSelectedPR = (state: GlobalState) => getPluginState(state).selectedPR;

export const getPRReviewThreads = (state: GlobalState) => getPluginState(state).prReviewThreads;

export const getPRReviewThreadsLoading = (state: GlobalState) => getPluginState(state).prReviewThreadsLoading;

export const getAIAgents = (state: GlobalState) => getPluginState(state).aiAgents;

export const getThreadsGroupedByFile = createSelector(
    getPRReviewThreads,
    (prReviewThreads): Record<string, ReviewThreadData[]> => {
        if (!prReviewThreads || !prReviewThreads.threads) {
            return {};
        }

        const grouped: Record<string, ReviewThreadData[]> = {};
        prReviewThreads.threads.forEach((thread: ReviewThreadData) => {
            if (!grouped[thread.path]) {
                grouped[thread.path] = [];
            }
            grouped[thread.path].push(thread);
        });

        return grouped;
    },
);
