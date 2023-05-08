import {Octokit} from 'octokit';

import {mockOAuthAccessToken} from './creds';

const octokit = new Octokit({
    auth: mockOAuthAccessToken,
});

export const closeIssue = async (owner: string, repo: string, issueNumber: number) => {
    const res = await octokit.rest.issues.update({
        owner,
        repo,
        issue_number: issueNumber,
        state: 'closed',
    });

    return res;
}
