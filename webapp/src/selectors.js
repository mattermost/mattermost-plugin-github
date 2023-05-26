import {getConfig} from 'mattermost-redux/selectors/entities/general';

import {createSelector} from 'reselect';

import {id as pluginId} from './manifest';

const emptyArray = [];

const getPluginState = (state) => state['plugins-' + pluginId] || {};

export const isEnabled = (state) => getPluginState(state).enabled;

export const getServerRoute = (state) => {
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

function mapPrsToDetails(prs, details) {
    if (!prs) {
        return [];
    }

    return prs.map((pr) => {
        let foundDetails;
        if (details) {
            foundDetails = details.find((prDetails) => {
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
        const {username, sidebarContent, reviewsDetails, yourPrsDetails, organization, rhsState} = pluginState;
        return {
            username,
            reviews: mapPrsToDetails(sidebarContent.reviews || emptyArray, reviewsDetails),
            yourPrs: mapPrsToDetails(sidebarContent.prs || emptyArray, yourPrsDetails),
            yourAssignments: sidebarContent.assignments || emptyArray,
            unreads: sidebarContent.unreads || emptyArray,
            org: organization,
            rhsState,
        };
    },
);

export const configuration = (state) => getPluginState(state).configuration;
