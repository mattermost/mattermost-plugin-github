import request from 'superagent';

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

    doGet = async (url, body, headers = {}) => {
        headers['X-Requested-With'] = 'XMLHttpRequest';
        headers['X-Timezone-Offset'] = new Date().getTimezoneOffset();

        try {
            const response = await request.
                get(url).
                set(headers).
                accept('application/json');

            return response.body;
        } catch (err) {
            throw err;
        }
    }

    doPost = async (url, body, headers = {}) => {
        headers['X-Requested-With'] = 'XMLHttpRequest';
        headers['X-Timezone-Offset'] = new Date().getTimezoneOffset();

        try {
            const response = await request.
                post(url).
                send(body).
                set(headers).
                type('application/json').
                accept('application/json');

            return response.body;
        } catch (err) {
            throw err;
        }
    }

    doDelete = async (url, body, headers = {}) => {
        headers['X-Requested-With'] = 'XMLHttpRequest';
        headers['X-Timezone-Offset'] = new Date().getTimezoneOffset();

        try {
            const response = await request.
                delete(url).
                send(body).
                set(headers).
                type('application/json').
                accept('application/json');

            return response.body;
        } catch (err) {
            throw err;
        }
    }

    doPut = async (url, body, headers = {}) => {
        headers['X-Requested-With'] = 'XMLHttpRequest';
        headers['X-Timezone-Offset'] = new Date().getTimezoneOffset();

        try {
            const response = await request.
                put(url).
                send(body).
                set(headers).
                type('application/json').
                accept('application/json');

            return response.body;
        } catch (err) {
            throw err;
        }
    }
}
