// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// ***********************************************************
// Read more at: https://on.cypress.io/configuration
// ***********************************************************

// import 'mattermost-webapp/e2e/cypress/support';

import '@testing-library/cypress/add-commands';

import 'cypress-wait-until';

const merge = require('deepmerge');

const TIMEOUTS = require('../fixtures/timeouts');

Cypress.Commands.add('apiLogin', (user, requestOptions = {}) => {
    return cy.request({
        headers: {'X-Requested-With': 'XMLHttpRequest'},
        url: '/api/v4/users/login',
        method: 'POST',
        body: {login_id: user.username || user.email, password: user.password},
        ...requestOptions,
    }).then((response) => {
        if (requestOptions.failOnStatusCode) {
            expect(response.status).to.equal(200);
        }

        if (response.status === 200) {
            return cy.wrap({
                user: {
                    ...response.body,
                    password: user.password,
                },
            });
        }

        return cy.wrap({error: response.body});
    });
});

Cypress.Commands.add('apiAdminLogin', (requestOptions = {}) => {
    const admin = getAdminAccount();

    // First, login with username
    cy.apiLogin(admin, requestOptions).then((resp) => {
        if (resp.error) {
            if (resp.error.id === 'mfa.validate_token.authenticate.app_error') {
                // On fail, try to login via MFA
                return cy.dbGetUser({username: admin.username}).then(({user: {mfasecret}}) => {
                    const token = authenticator.generateToken(mfasecret);
                    return cy.apiLoginWithMFA(admin, token);
                });
            }

            // Or, try to login via email
            delete admin.username;
            return cy.apiLogin(admin, requestOptions);
        }

        return resp;
    });
});

Cypress.Commands.add('apiEnablePluginById', (pluginId) => {
    return cy.request({
        headers: {'X-Requested-With': 'XMLHttpRequest'},
        url: `/api/v4/plugins/${encodeURIComponent(pluginId)}/enable`,
        method: 'POST',
        timeout: TIMEOUTS.TWO_MIN,
        failOnStatusCode: false,
    }).then((response) => {
        expect(response.status).to.equal(200);
        return cy.wrap(response);
    });
});

Cypress.Commands.add('apiDisablePluginById', (pluginId) => {
    return cy.request({
        headers: {'X-Requested-With': 'XMLHttpRequest'},
        url: `/api/v4/plugins/${encodeURIComponent(pluginId)}/disable`,
        method: 'POST',
        timeout: TIMEOUTS.TWO_MIN,
        failOnStatusCode: false,
    }).then((response) => {
        expect(response.status).to.equal(200);
        return cy.wrap(response);
    });
});

Cypress.Commands.add('getLastPostId', () => {
    waitUntilPermanentPost();

    cy.findAllByTestId('postView').last().should('have.attr', 'id').and('not.include', ':')
        .invoke('replace', 'post_', '');
});

Cypress.Commands.add('apiGetConfig', (old = false) => {
    // # Get current settings
    return cy.request(`/api/v4/config${old ? '/client?format=old' : ''}`).then((response) => {
        expect(response.status).to.equal(200);
        return cy.wrap({config: response.body});
    });
});

Cypress.Commands.add('apiUpdateConfig', (newConfig = {}) => {
    // # Get current config
    return cy.apiGetConfig().then(({config: currentConfig}) => {
        const config = merge.all([currentConfig, newConfig]);

        // # Set the modified config
        return cy.request({
            url: '/api/v4/config',
            headers: {'X-Requested-With': 'XMLHttpRequest'},
            method: 'PUT',
            body: config,
        }).then((updateResponse) => {
            expect(updateResponse.status).to.equal(200);
            return cy.apiGetConfig();
        });
    });
});

Cypress.Commands.add('apiRemoveAllPostsInDirectChannel', (username, botUsername) => {
    return cy.request(`/api/v4/users/username/${username}`).then((userResponse) => {
        expect(userResponse.status).to.equal(200);

        cy.request(`/api/v4/users/username/${botUsername}`).then((botResponse) => {
            expect(botResponse.status).to.equal(200);

            cy.request({
                url : '/api/v4/channels/direct',
                body:[userResponse.body.id, botResponse.body.id],
                headers: {'X-Requested-With': 'XMLHttpRequest'},
                method: 'POST',
            }).then((directChannelResponse) => {
                expect(directChannelResponse.status).to.equal(201);

                cy.request(`/api/v4/channels/${directChannelResponse.body.id}/posts`).then((postsResponse) => {
                    expect(postsResponse.status).to.equal(200);

                    const postIds = postsResponse.body.order;
                    for (const postId of postIds) {
                        cy.request({
                            url: `/api/v4/posts/${postId}`,
                            headers: {'X-Requested-With': 'XMLHttpRequest'},
                            method: 'delete'
                        }).then((response) => {
                            expect(response.status).to.equal(200);
                        });
                    }
                });
            });
        });
    });
});

Cypress.Commands.add('apiCreateOrGetTeam', (teamName) => {
    return cy.request({
        url: `/api/v4/teams/name/${teamName}`,
        failOnStatusCode: false,
    }).then((response) => {
        if (response.status === 200) {
            return cy.wrap({team: response.body});
        }

        return cy.request({
            headers: {'X-Requested-With': 'XMLHttpRequest'},
            url: '/api/v4/teams',
            method: 'POST',
            body: {
                name: teamName,
                display_name: teamName,
                type: 'O',
            },
        }).then((response) => {
            expect(response.status).to.equal(201);
            return cy.wrap({team: response.body});
        });
    });
});

function waitUntilPermanentPost() {
    cy.wait(TIMEOUTS.HALF_SEC);
    cy.get('#postListContent', {timeout: TIMEOUTS.ONE_MIN}).should('be.visible');
    cy.waitUntil(() => cy.findAllByTestId('postView').last().then((el) => !(el[0].id.includes(':'))));
}

function getAdminAccount() {
    return {
        username: Cypress.env('adminUsername'),
        password: Cypress.env('adminPassword'),
        email: Cypress.env('adminEmail'),
    };
}
