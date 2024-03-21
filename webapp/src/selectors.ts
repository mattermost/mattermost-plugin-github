import {getConfig} from 'mattermost-redux/selectors/entities/general';

import {createSelector} from 'reselect';

import {GlobalState} from './types/store';

const getPluginState = (state: GlobalState) => state['plugins-github'] || {};

export const getServerRoute = (state: GlobalState) => {
    const config = getConfig(state);
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
    (pluginState) => {
        const {username, sidebarContent, reviewDetails, yourPrDetails, organization, rhsState} = pluginState;
        return {
            username,
            reviews: mapPrsToDetails(sidebarContent.reviews || [], reviewDetails),
            yourPrs: mapPrsToDetails(sidebarContent.prs || [], yourPrDetails),
            yourAssignments: sidebarContent.assignments || [],
            unreads: sidebarContent.unreads || [],
            org: organization,
            rhsState,
        };
    },
);

export const configuration = (state: GlobalState) => getPluginState(state).configuration;
