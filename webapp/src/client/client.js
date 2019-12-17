// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Client4} from 'mattermost-redux/client';
import {ClientError} from 'mattermost-redux/client/client4';

export default class Client {
    constructor() {
        this.url = '/plugins/github/api/v1';
    }

    getConnected = async (reminder = false) => {
        return this.doGet(`${this.url}/connected?reminder=` + reminder);
    }

    getReviews = async () => {
        return this.doGet(`${this.url}/reviews`);
    }

    getYourPrs = async () => {
        return this.doGet(`${this.url}/yourprs`);
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

    searchIssues = async (searchTerm) => {
        return this.doGet(`${this.url}/searchissues?term=${searchTerm}`);
    }

    attachCommentToIssue = async (payload) => {
        return this.doPost(`${this.url}/createissuecomment`, payload);
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
