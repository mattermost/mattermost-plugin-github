import {getConfig} from 'mattermost-redux/selectors/entities/general';

import {createSelector} from 'reselect';

import {GlobalState, PluginState} from './types/store';
import {ForgejoIssueData, SidebarData, PrsDetailsData, UnreadsData} from './types/forgejo_types';

const emptyArray: ForgejoIssueData[] | UnreadsData[] = [];

export const getPluginState = (state: GlobalState): PluginState => state['plugins-forgejo'];

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

function mapPrsToDetails(prs: ForgejoIssueData[], details: PrsDetailsData[]) {
    if (!prs) {
        return [];
    }

    return prs.map((pr: ForgejoIssueData) => {
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
