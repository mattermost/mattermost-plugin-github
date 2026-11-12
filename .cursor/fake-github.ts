// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Runs the E2E suite's fake GitHub on its own so an agent can use the plugin from a browser, and
// seeds a small world to start from. The Playwright suite starts its own copy in global_setup, so
// stop this one (Ctrl-C in the github-stack terminal) before `make e2e`.
//
//   npx --yes tsx@4.23.15 ../../.cursor/fake-github.ts          start on :8093 and seed (run from e2e/integration)
//   npx --yes tsx@4.23.15 ../../.cursor/fake-github.ts --seed   reset a running fake and seed it again

import {startFakeGitHub} from '../e2e/integration/fake-github/server';
import {fakeGitHub} from '../e2e/integration/support/fake_github';
import {fakeGitHubPort, fakeGitHubURL, oauthApp, webhookSecret, webhookURL} from '../e2e/integration/support/constants';

const login = 'octo-agent';
const repo = 'acme/widgets';

async function seed() {
    await fakeGitHub.reset();
    await fakeGitHub.createUser(login, {name: 'Octo Agent', email: 'octo-agent@fake-github.test'});
    await fakeGitHub.createUser('octo-author', {name: 'Octo Author'});
    await fakeGitHub.createOrg('acme', {members: [login, 'octo-author'], admins: [login]});
    await fakeGitHub.createRepo(repo, {admins: [login], writers: ['octo-author'], labels: ['bug', 'enhancement'], milestones: ['v1.0']});
    await fakeGitHub.createRepo(`${login}/sandbox`, {admins: [login]});
    const pr = await fakeGitHub.createItem(repo, {kind: 'pull', title: 'Polish the widgets', author: login, labels: ['enhancement'], body: 'Splits polishing into stages.'});
    await fakeGitHub.createItem(repo, {kind: 'pull', title: 'Please review the gizmo', author: 'octo-author', reviewers: [login]});
    await fakeGitHub.createItem(repo, {kind: 'issue', title: 'Widgets squeak on Mondays', author: 'octo-author', assignees: [login], labels: ['bug'], milestone: 1});
    await fakeGitHub.createThread(login, repo, {number: pr.number, reason: 'mention'});

    // Deliveries from POST /__control/webhooks reach the plugin once its webhook secret matches.
    await fakeGitHub.createHook('acme', repo.split('/')[1], webhookURL, webhookSecret);

    // The OAuth page then goes straight to "Authorize"; POST /__control/sign-in {"login": null} shows the sign-in form instead.
    await fakeGitHub.signIn(login);
}

async function main() {
    if (process.argv.includes('--seed')) {
        await seed();
        console.log(`[fake-github] Reset and seeded the fake at ${fakeGitHubURL}.`);
        return;
    }

    const fake = await startFakeGitHub({port: fakeGitHubPort, publicURL: fakeGitHubURL, app: oauthApp});
    await seed();
    console.log(`[fake-github] Listening at ${fakeGitHubURL} (control API at http://localhost:${fakeGitHubPort}/__control/), signed in as ${login}; repos ${repo} and ${login}/sandbox.`);

    const stop = async () => {
        await fake.close();
        process.exit(0);
    };
    process.on('SIGINT', stop);
    process.on('SIGTERM', stop);
}

main().catch((err) => {
    console.error(err);
    process.exit(1);
});
