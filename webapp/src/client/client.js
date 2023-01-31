// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Client4} from 'mattermost-redux/client';
import {ClientError} from 'mattermost-redux/client/client4';

import {id as pluginId} from '../manifest';

export default class Client {
    editIssueModal = async (payload) => {
        return this.doPost(`${this.url}/editissuemodal`, payload);
    }

    closeOrReopenIssueModal = async (payload) => {
        return this.doPost(`${this.url}/closereopenissuemodal`, payload);
    }

    attachCommentIssueModal = async (payload) => {
        return this.doPost(`${this.url}/attachcommentissuemodal`, payload);
    }

    setServerRoute(url) {
        this.url = url + `/plugins/${pluginId}/api/v1`;
    }

    getConnected = async (reminder = false) => {
        return this.doGet(`${this.url}/connected?reminder=${reminder}`);
    }

    getReviews = async () => {
        return this.doGet(`${this.url}/reviews`);
    }

    getYourPrs = async () => {
        return this.doGet(`${this.url}/yourprs`);
    }

    getPrsDetails = async (prList) => {
        return this.doPost(`${this.url}/prsdetails`, prList);
    }

    getYourAssignments = async () => {
        return this.doGet(`${this.url}/yourassignments`);
    }

    getMentions = async () => {
        return this.doGet(`${this.url}/mentions`);
    }

    getUnreads = async () => {
        return this.doGet(`${this.url}/unreads`);
    }

    getGitHubUser = async (userID) => {
        return this.doPost(`${this.url}/user`, {user_id: userID});
    }

    getRepositories = async () => {
        return this.doGet(`${this.url}/repositories`);
    }

    getLabels = async (repo) => {
        return this.doGet(`${this.url}/labels?repo=${repo}`);
    }

    getAssignees = async (repo) => {
        return this.doGet(`${this.url}/assignees?repo=${repo}`);
    }

    getMilestones = async (repo) => {
        return this.doGet(`${this.url}/milestones?repo=${repo}`);
    }

    createIssue = async (payload) => {
        return this.doPost(`${this.url}/createissue`, payload);
    }

    closeOrReopenIssue = async (payload) => {
        return this.doPost(`${this.url}/closeorreopenissue`, payload);
    }

    updateIssue = async (payload) => {
        return this.doPost(`${this.url}/updateissue`, payload);
    }

    searchIssues = async (searchTerm) => {
        return this.doGet(`${this.url}/searchissues?term=${searchTerm}`);
    }

    attachCommentToIssue = async (payload) => {
        return this.doPost(`${this.url}/createissuecomment`, payload);
    }

    getIssue = async (owner, repo, issueNumber) => {
        return this.doGet(`${this.url}/issue?owner=${owner}&repo=${repo}&number=${issueNumber}`);
    }

    getPullRequest = async (owner, repo, prNumber) => {
        return this.doGet(`${this.url}/pr?owner=${owner}&repo=${repo}&number=${prNumber}`);
    }

    doGet = async (url, body, headers = {}) => {
        headers['X-Timezone-Offset'] = new Date().getTimezoneOffset();

        const options = {
            method: 'get',
            headers,
        };

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
    }

    doPost = async (url, body, headers = {}) => {
        headers['X-Timezone-Offset'] = new Date().getTimezoneOffset();

        const options = {
            method: 'post',
            body: JSON.stringify(body),
            headers,
        };

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
    }

    doDelete = async (url, body, headers = {}) => {
        headers['X-Timezone-Offset'] = new Date().getTimezoneOffset();

        const options = {
            method: 'delete',
            headers,
        };

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
    }

    doPut = async (url, body, headers = {}) => {
        headers['X-Timezone-Offset'] = new Date().getTimezoneOffset();

        const options = {
            method: 'put',
            body: JSON.stringify(body),
            headers,
        };

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
    }
}
