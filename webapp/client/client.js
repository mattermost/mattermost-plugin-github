import request from 'superagent';

export default class Client {
    constructor() {
        this.url = '/plugins/github/api/v1';
    }

    requestReviewers = async (prId, reviewers, org, repo) => {
        return this.doPost(`${this.url}/pr/reviewers`, {pull_request_id: prId, reviewers, org, repo});
    }

    removeReviewers = async (prId, reviewers, org, repo) => {
        return this.doDelete(`${this.url}/pr/reviewers`, {pull_request_id: prId, reviewers, org, repo});
    }

    editPr = async (prId, labels, assignees, milestone, org, repo) => {
        return this.doPost(`${this.url}/pr`, {pull_request_id: prId, labels, assignees, milestone, org, repo});
    }

    doPost = async (url, body, headers = {}) => {
        headers['X-Requested-With'] = 'XMLHttpRequest';

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
