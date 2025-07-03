// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Client4} from 'mattermost-redux/client';
import {ClientError} from 'mattermost-redux/client/client4';

import {ConnectedData, GithubIssueData, GithubLabel, GithubUsersData, GitHubIssueCommentData, MentionsData, MilestoneData, PrsDetailsData, SidebarContentData, YourReposData, GitHubPullRequestData} from '../types/github_types';

import manifest from '../manifest';

export type ApiError = {
    id: string;
    message: string;
    status_code: number;
};

export default class Client {
    url!: string;

    setServerRoute(url: string) {
        this.url = url + `/plugins/${manifest.id}/api/v1`;
    }

    getConnected = async (reminder = false) => {
        return this.doGet<ConnectedData>(`${this.url}/connected?reminder=${reminder}`);
    }

    getSidebarContent = async () => {
        return this.doGet<SidebarContentData>(`${this.url}/lhs-content`);
    }

    getOrganizations = async () => {
        return this.doGet(`${this.url}/organizations?includeLoggedInUser=true`);
    }

    getRepositoriesByChannelID = async (channelId: string) => {
        return this.doGet(`${this.url}/repositories?channelId=${channelId}`);
    }

    getRepositoriesByOrganization = async (organization: string) => {
        return this.doGet(`${this.url}/repos_by_org?organization=${organization}`);
    }

    getPrsDetails = async (prList: {url: string, number: number}[]) => {
        return this.doPost<PrsDetailsData[]>(`${this.url}/prsdetails`, prList);
    }

    getMentions = async () => {
        return this.doGet<MentionsData[]>(`${this.url}/mentions`);
    }

    getGitHubUser = async (userID: string) => {
        return this.doPost<GithubUsersData>(`${this.url}/user`, {user_id: userID});
    }

    getRepositories = async () => {
        return this.doGet<YourReposData[]>(`${this.url}/repositories`);
    }

    getLabels = async (repo: string) => {
        return this.doGet<GithubLabel[]>(`${this.url}/labels?repo=${repo}`);
    }

    getAssignees = async (repo: string) => {
        return this.doGet<GithubUsersData[]>(`${this.url}/assignees?repo=${repo}`);
    }

    getMilestones = async (repo: string) => {
        return this.doGet<MilestoneData[]>(`${this.url}/milestones?repo=${repo}`);
    }

    createIssue = async (payload: CreateIssuePayload) => {
        return this.doPost<GithubIssueData>(`${this.url}/createissue`, payload);
    }

    searchIssues = async (searchTerm: string) => {
        return this.doGet<GithubIssueData[]>(`${this.url}/searchissues?term=${searchTerm}`);
    }

    attachCommentToIssue = async (payload: AttachCommentToIssuePayload) => {
        return this.doPost<GitHubIssueCommentData>(`${this.url}/createissuecomment`, payload);
    }

    getIssue = async (owner: string, repo: string, issueNumber: number) => {
        return this.doGet<GithubIssueData>(`${this.url}/issue?owner=${owner}&repo=${repo}&number=${issueNumber}`);
    }

    getPullRequest = async (owner: string, repo: string, prNumber: number) => {
        return this.doGet<GitHubPullRequestData>(`${this.url}/pr?owner=${owner}&repo=${repo}&number=${prNumber}`);
    }

    private doGet = async <Response>(url: string): Promise<Response | ApiError> => {
        const headers = {
            'X-Timezone-Offset': new Date().getTimezoneOffset().toString(),
        };

        const options = {
            method: 'get',
            headers,
        };

        try {
            const response = await fetch(url, Client4.getOptions(options));

            if (response.ok) {
                return response.json();
            }

            const text = await response.text();
            throw new ClientError(Client4.url, {
                message: text || '',
                status_code: response.status,
                url,
            });
        } catch (e) {
            throw new ClientError(Client4.url, {
                message: (e as {toString: () => string}).toString(),
                status_code: 500,
                url,
            });
        }
    }

    doPost = async <Response>(url: string, body: Object): Promise<Response | ApiError> => {
        const headers = {
            'X-Timezone-Offset': new Date().getTimezoneOffset().toString(),
        };

        const options = {
            method: 'post',
            body: JSON.stringify(body),
            headers,
        };

        try {
            const response = await fetch(url, Client4.getOptions(options));

            if (response.ok) {
                return response.json();
            }

            const text = await response.text();
            throw new ClientError(Client4.url, {
                message: text || '',
                status_code: response.status,
                url,
            });
        } catch (e) {
            throw new ClientError(Client4.url, {
                message: (e as {toString: () => string}).toString(),
                status_code: 500,
                url,
            });
        }
    }

    doDelete = async (url: string) => {
        const headers = {
            'X-Timezone-Offset': new Date().getTimezoneOffset().toString(),
        };

        const options = {
            method: 'delete',
            headers,
        };

        try {
            const response = await fetch(url, Client4.getOptions(options));

            if (response.ok) {
                return response.json();
            }

            const text = await response.text();
            throw new ClientError(Client4.url, {
                message: text || '',
                status_code: response.status,
                url,
            });
        } catch (e) {
            throw new ClientError(Client4.url, {
                message: (e as {toString: () => string}).toString(),
                status_code: 500,
                url,
            });
        }
    }

    doPut = async <Response, Body>(url: string, body: Body): Promise<Response | ApiError> => {
        const headers = {
            'X-Timezone-Offset': new Date().getTimezoneOffset().toString(),
        };

        const options = {
            method: 'put',
            body: JSON.stringify(body),
            headers,
        };

        try {
            const response = await fetch(url, Client4.getOptions(options));

            if (response.ok) {
                return response.json();
            }

            const text = await response.text();
            throw new ClientError(Client4.url, {
                message: text || '',
                status_code: response.status,
                url,
            });
        } catch (e) {
            throw new ClientError(Client4.url, {
                message: (e as {toString: () => string}).toString(),
                status_code: 500,
                url,
            });
        }
    }
}
