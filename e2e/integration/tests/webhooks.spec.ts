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
    /**
     * @objective Verify subscribing to a repository with no webhook pointing at this server posts a setup hint.
     *
     * @precondition
     * The user connects with the `repo` scope so the plugin can list repository hooks.
     */
    test('warns on subscribe when the repository has no webhook pointing at this server', {tag: '@github_plugin'}, async ({pw, github}) => {
        await fakeGitHub.createRepo(`${github.login}/widgets`);
        await github.connect(undefined, {scopes: [...defaultScopes, 'repo']});
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);

        // # Subscribe the channel to the repository
        await runSlashCommand(channelsPage, `/github subscriptions add ${github.login}/widgets`);

        // * Verify the subscription is confirmed with a webhook setup hint
        await expect(postWithText(page, 'subscribed this channel to')).toContainText(`${github.login}/widgets`);
        await expect(postWithText(page, noWebhookNote)).toContainText('/github setup webhook');
    });

    /**
     * @objective Verify subscribing to a repository whose webhook already points at this server posts no warning.
     *
     * @precondition
     * The user connects with the `repo` scope so the plugin can list repository hooks.
     */
    test('omits the no-webhook warning on subscribe when a webhook points at this server', {tag: '@github_plugin'}, async ({pw, github}) => {
        await fakeGitHub.createRepo(`${github.login}/widgets`);
        await fakeGitHub.createHook(github.login, 'widgets', webhookURL, webhookSecret);
        await github.connect(undefined, {scopes: [...defaultScopes, 'repo']});
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);

        // # Subscribe the channel to the repository
        await runSlashCommand(channelsPage, `/github subscriptions add ${github.login}/widgets`);

        // * Verify the subscription is confirmed after the hooks are checked, with no warning
        await expect(postWithText(page, 'subscribed this channel to')).toContainText(`${github.login}/widgets`);
        await expect.poll(() => fakeGitHub.calls()).toContain(`GET /api/repos/${github.login}/widgets/hooks`);
        await expect(postWithText(page, noWebhookNote)).toHaveCount(0);
    });

    // Listing repository hooks needs `repo` or `read:repo_hook`; the plugin's public-repo scopes have neither,
    // GitHub answers 404, and the plugin treats that as "can't tell" instead of warning.
    /**
     * @objective Verify the plugin skips the no-webhook warning when the default scopes cannot list repository hooks.
     */
    test('skips the no-webhook warning on subscribe with the default public-repo scopes', {tag: '@github_plugin'}, async ({pw, github}) => {
        await fakeGitHub.createRepo(`${github.login}/widgets`);
        await github.connect();
        const {channelsPage, page} = await openChannel(pw, github.user, github.team.name);

        // # Subscribe the channel to the repository
        await runSlashCommand(channelsPage, `/github subscriptions add ${github.login}/widgets`);

        // * Verify the subscription is confirmed, the hooks check got 404 and no warning is posted
        await expect(postWithText(page, 'subscribed this channel to')).toBeVisible();
        await expect.poll(async () => (await fakeGitHub.requests()).find((r) => r.path === `/api/repos/${github.login}/widgets/hooks`)?.status).toBe(404);
        await expect(postWithText(page, noWebhookNote)).toHaveCount(0);
    });
});

test.describe('webhook events', () => {
    /**
     * @objective Verify a pull_request opened webhook posts the new pull request to the subscribed channel.
     */
    test('posts a new pull request to the subscribed channel', {tag: ['@github_plugin', '@synthetic-inbound']}, async ({github}) => {
        const repo = await subscribedRepo(github);
        const pr = await fakeGitHub.createItem(repo, {kind: 'pull', title: 'Add widget polishing', author: github.login, body: 'Polishes widgets.', labels: ['enhancement']});

        // # Deliver a pull_request opened webhook
        const {deliveries} = await fakeGitHub.webhook({event: 'pull_request', action: 'opened', repo, number: pr.number, sender: github.login});

        // * Verify the delivery is accepted and the channel gets the pull request post
        expect(deliveries.map((d) => d.status)).toEqual([200]);
        const post = await waitForChannelPost(github, 'Add widget polishing');
        expect(post.message).toContain('#new-pull-request');
        expect(post.message).toContain(`https://github.com/${repo}/pull/${pr.number}`);
    });

    /**
     * @objective Verify an issues opened webhook posts the new issue to the subscribed channel.
     */
    test('posts a new issue to the subscribed channel', {tag: ['@github_plugin', '@synthetic-inbound']}, async ({github}) => {
        const repo = await subscribedRepo(github);
        const issue = await fakeGitHub.createItem(repo, {kind: 'issue', title: 'Widgets wobble', author: github.login, body: 'They wobble.'});

        // # Deliver an issues opened webhook
        await fakeGitHub.webhook({event: 'issues', action: 'opened', repo, number: issue.number, sender: github.login});

        // * Verify the channel gets the new issue post
        const post = await waitForChannelPost(github, 'Widgets wobble');
        expect(post.message).toContain('#new-issue');
    });

    /**
     * @objective Verify an issue_comment created webhook posts the comment to the subscribed channel.
     */
    test('posts an issue comment to the subscribed channel', {tag: ['@github_plugin', '@synthetic-inbound']}, async ({github}) => {
        const repo = await subscribedRepo(github);
        const issue = await fakeGitHub.createItem(repo, {kind: 'issue', title: 'Widgets squeak', author: github.login});

        // # Deliver an issue_comment created webhook
        await fakeGitHub.webhook({event: 'issue_comment', action: 'created', repo, number: issue.number, sender: github.login, comment: 'Oiled them, still squeaky.'});

        // * Verify the channel gets the comment post
        const post = await waitForChannelPost(github, 'Oiled them, still squeaky.');
        expect(post.message).toContain('New comment by');
    });

    /**
     * @objective Verify a push webhook posts the pushed commits to the subscribed channel.
     */
    test('posts a push to the subscribed channel', {tag: ['@github_plugin', '@synthetic-inbound']}, async ({github}) => {
        const repo = await subscribedRepo(github);

        // # Deliver a push webhook
        await fakeGitHub.webhook({event: 'push', repo, sender: github.login});

        // * Verify the channel gets the push post
        const post = await waitForChannelPost(github, 'Update README');
        expect(post.message).toContain('pushed [1 new commit]');
        expect(post.message).toContain(`${repo}:main`);
    });

    /**
     * @objective Verify a review_requested webhook DMs the requested reviewer when they are connected.
     *
     * @precondition
     * A second user is created and connected as the reviewer.
     */
    test('DMs the connected reviewer when a pull request review is requested', {tag: ['@github_plugin', '@synthetic-inbound']}, async ({pw, github}) => {
        const repo = await subscribedRepo(github);
        const reviewer = await pw.createNewUserProfile(github.adminClient, {prefix: 'reviewer', disableTutorial: true, disableOnboarding: true});
        await github.adminClient.addToTeam(github.team.id, reviewer.id);
        const {login: reviewerLogin} = await github.connect(reviewer);
        const pr = await fakeGitHub.createItem(repo, {kind: 'pull', title: 'Review my widget', author: github.login, reviewers: [reviewerLogin]});

        // # Deliver a pull_request review_requested webhook
        await fakeGitHub.webhook({event: 'pull_request', action: 'review_requested', repo, number: pr.number, sender: github.login, requestedReviewer: reviewerLogin});

        // * Verify the reviewer gets a review request DM naming the pull request
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

    /**
     * @objective Verify the plugin rejects an unsigned webhook delivery and posts nothing.
     */
    test('rejects webhook delivery without X-Hub-Signature', {tag: ['@github_plugin', '@synthetic-inbound']}, async ({github}) => {
        // # Deliver an unsigned issues webhook
        const result = await deliverRaw(github, () => ({}));

        // * Verify it is rejected with 401 and nothing is posted
        expect(result).toEqual({status: 401, posted: false});
    });

    /**
     * @objective Verify the plugin rejects a webhook delivery signed with the wrong secret and posts nothing.
     */
    test('rejects webhook delivery signed with another secret', {tag: ['@github_plugin', '@synthetic-inbound']}, async ({github}) => {
        // # Deliver an issues webhook signed with the wrong secret
        const result = await deliverRaw(github, (body) => signatures('not-the-webhook-secret', body));

        // * Verify it is rejected with 401 and nothing is posted
        expect(result).toEqual({status: 401, posted: false});
    });

    /**
     * @objective Verify the plugin accepts a webhook delivery signed with the configured secret and posts it.
     */
    test('accepts webhook delivery signed with the right secret', {tag: ['@github_plugin', '@synthetic-inbound']}, async ({github}) => {
        // # Deliver an issues webhook signed with the configured secret
        const result = await deliverRaw(github, (body) => signatures(webhookSecret, body));

        // * Verify it is accepted and the issue is posted
        expect(result).toEqual({status: 200, posted: true});
    });
});
