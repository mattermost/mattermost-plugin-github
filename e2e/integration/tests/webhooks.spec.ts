// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {defaultScopes} from '../fake-github/server';
import {signatures} from '../fake-github/webhooks';
import {webhookSecret, webhookURL} from '../support/constants';
import {expect, test, type GitHubFixture} from '../support/fixture';
import {fakeGitHub} from '../support/fake_github';
import {botDMPosts, channelPosts, openChannel, postWithText, runSlashCommand, waitForPost} from '../support/ui';

const noWebhookNote = 'No webhook was found for this repository or organization';
const allFeatures = 'pulls,issues,issue_comments,pushes';

async function subscribedRepo(github: GitHubFixture) {
    const repo = `${github.login}/widgets`;
    await fakeGitHub.createRepo(repo);
    await fakeGitHub.createHook(github.login, 'widgets', webhookURL, webhookSecret);
    await github.connect();
    await github.userClient.executeCommand(`/github subscriptions add ${repo} --features ${allFeatures}`, {channel_id: github.channel.id, team_id: github.team.id});
    return repo;
}

function waitForChannelPost(github: GitHubFixture, text: string) {
    return waitForPost(() => channelPosts(github.adminClient, github.channel.id), (p) => p.message.includes(text), `channel post containing "${text}"`);
}

test.describe('subscriptions', () => {
    test('warns that the repository has no webhook pointing at this server', async ({pw, github}) => {
        await fakeGitHub.createRepo(`${github.login}/widgets`);
        await github.connect(undefined, {scopes: [...defaultScopes, 'repo']});
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);

        await runSlashCommand(channelsPage, `/github subscriptions add ${github.login}/widgets`);

        await expect(postWithText(page, 'subscribed this channel to')).toContainText(`${github.login}/widgets`);
        await expect(postWithText(page, noWebhookNote)).toContainText('/github setup webhook');
    });

    test('drops the warning once a webhook points at this server', async ({pw, github}) => {
        await fakeGitHub.createRepo(`${github.login}/widgets`);
        await fakeGitHub.createHook(github.login, 'widgets', webhookURL, webhookSecret);
        await github.connect(undefined, {scopes: [...defaultScopes, 'repo']});
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);

        await runSlashCommand(channelsPage, `/github subscriptions add ${github.login}/widgets`);

        await expect(postWithText(page, 'subscribed this channel to')).toContainText(`${github.login}/widgets`);
        await expect.poll(() => fakeGitHub.calls()).toContain(`GET /api/repos/${github.login}/widgets/hooks`);
        await expect(postWithText(page, noWebhookNote)).toHaveCount(0);
    });

    // Listing repository hooks needs `repo` or `read:repo_hook`; the plugin's public-repo scopes have neither,
    // GitHub answers 404, and the plugin treats that as "can't tell" instead of warning.
    test('cannot check for a webhook with the default public-repo scopes', async ({pw, github}) => {
        await fakeGitHub.createRepo(`${github.login}/widgets`);
        await github.connect();
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);

        await runSlashCommand(channelsPage, `/github subscriptions add ${github.login}/widgets`);

        await expect(postWithText(page, 'subscribed this channel to')).toBeVisible();
        await expect.poll(async () => (await fakeGitHub.requests()).find((r) => r.path === `/api/repos/${github.login}/widgets/hooks`)?.status).toBe(404);
        await expect(postWithText(page, noWebhookNote)).toHaveCount(0);
    });
});

test.describe('webhook events', () => {
    test('a new pull request is posted to the subscribed channel', async ({github}) => {
        const repo = await subscribedRepo(github);
        const pr = await fakeGitHub.createItem(repo, {kind: 'pull', title: 'Add widget polishing', author: github.login, body: 'Polishes widgets.', labels: ['enhancement']});

        const {deliveries} = await fakeGitHub.webhook({event: 'pull_request', action: 'opened', repo, number: pr.number, sender: github.login});

        expect(deliveries.map((d) => d.status)).toEqual([200]);
        const post = await waitForChannelPost(github, 'Add widget polishing');
        expect(post.message).toContain('#new-pull-request');
        expect(post.message).toContain(`https://github.com/${repo}/pull/${pr.number}`);
    });

    test('a new issue is posted to the subscribed channel', async ({github}) => {
        const repo = await subscribedRepo(github);
        const issue = await fakeGitHub.createItem(repo, {kind: 'issue', title: 'Widgets wobble', author: github.login, body: 'They wobble.'});

        await fakeGitHub.webhook({event: 'issues', action: 'opened', repo, number: issue.number, sender: github.login});

        const post = await waitForChannelPost(github, 'Widgets wobble');
        expect(post.message).toContain('#new-issue');
    });

    test('an issue comment is posted to the subscribed channel', async ({github}) => {
        const repo = await subscribedRepo(github);
        const issue = await fakeGitHub.createItem(repo, {kind: 'issue', title: 'Widgets squeak', author: github.login});

        await fakeGitHub.webhook({event: 'issue_comment', action: 'created', repo, number: issue.number, sender: github.login, comment: 'Oiled them, still squeaky.'});

        const post = await waitForChannelPost(github, 'Oiled them, still squeaky.');
        expect(post.message).toContain('New comment by');
    });

    test('a push is posted to the subscribed channel', async ({github}) => {
        const repo = await subscribedRepo(github);

        await fakeGitHub.webhook({event: 'push', repo, sender: github.login});

        const post = await waitForChannelPost(github, 'Update README');
        expect(post.message).toContain('pushed [1 new commit]');
        expect(post.message).toContain(`${repo}:main`);
    });

    test('a review request DMs the connected reviewer', async ({pw, github}) => {
        const repo = await subscribedRepo(github);
        const reviewer = await pw.createNewUserProfile(github.adminClient, {prefix: 'reviewer', disableTutorial: true, disableOnboarding: true});
        await github.adminClient.addToTeam(github.team.id, reviewer.id);
        const {login: reviewerLogin} = await github.connect(reviewer);
        const pr = await fakeGitHub.createItem(repo, {kind: 'pull', title: 'Review my widget', author: github.login, reviewers: [reviewerLogin]});

        await fakeGitHub.webhook({event: 'pull_request', action: 'review_requested', repo, number: pr.number, sender: github.login, requestedReviewer: reviewerLogin});

        const dm = await waitForPost(() => botDMPosts(github.adminClient, reviewer.id), (p) => p.message.includes('requested your review'), 'review request DM');
        expect(dm.message).toContain('Review my widget');
    });
});

test.describe('webhook signatures', () => {
    async function deliverRaw(github: GitHubFixture, sign: (body: string) => Record<string, string>) {
        const repo = await subscribedRepo(github);
        const issue = await fakeGitHub.createItem(repo, {kind: 'issue', title: 'Tampered issue', author: github.login});
        const {payload} = await fakeGitHub.webhook({event: 'issues', action: 'opened', repo, number: issue.number, sender: github.login, deliver: false});
        const body = JSON.stringify(payload);
        const res = await fetch(webhookURL, {method: 'POST', body, headers: {'Content-Type': 'application/json', 'X-GitHub-Event': 'issues', ...sign(body)}});
        const posts = await channelPosts(github.adminClient, github.channel.id);
        return {status: res.status, posted: posts.some((p) => p.message.includes('Tampered issue'))};
    }

    test('a delivery without X-Hub-Signature is rejected', async ({github}) => {
        expect(await deliverRaw(github, () => ({}))).toEqual({status: 401, posted: false});
    });

    test('a delivery signed with another secret is rejected', async ({github}) => {
        expect(await deliverRaw(github, (body) => signatures('not-the-webhook-secret', body))).toEqual({status: 401, posted: false});
    });

    test('a delivery signed with the right secret is accepted', async ({github}) => {
        expect(await deliverRaw(github, (body) => signatures(webhookSecret, body))).toEqual({status: 200, posted: true});
    });
});
