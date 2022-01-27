// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
// <reference path="../support/index.d.ts" />

// ***************************************************************
// - [#] indicates a test step (e.g. # Go to a page)
// - [*] indicates an assertion (e.g. * Check the title)
// - Use element ID when selecting an element. Create one if none.
// ***************************************************************

const defaultPluginConfig = {
    "encryptionkey": "",
    "webhooksecret": "",

    "connecttoprivatebydefault": true,
    "enablecodepreview": "",
    "enableleftsidebar": true,
    "enableprivaterepo": true,
    "enablewebhookeventlogging": false,
    "enterprisebaseurl": "",
    "enterpriseuploadurl": "",
    "githuboauthclientid": "",
    "githuboauthclientsecret": "",
    "githuborg": "",
    "usepreregisteredapplication": false,
}

describe('GitHub setup wizard', () => {
    let settingsWithGenerated;
    let testTeam;

    const adminUsername = Cypress.env('adminUsername');
    const pluginId = 'github';
    const botUsername = 'github';
    const slashCommandName = 'github';

    before(() => {
        cy.apiAdminLogin();
        cy.apiCreateOrGetTeam('test').then(({team}) => {
            testTeam = team;
        });
    })

    beforeEach(() => {
        cy.apiAdminLogin();
        cy.apiRemoveAllPostsInDirectChannel(adminUsername, botUsername);

        cy.apiUpdateConfig({
            PluginSettings: {
                Plugins: {
                    [pluginId]: defaultPluginConfig,
                },
            },
        });

        cy.apiDisablePluginById(pluginId);
        cy.apiEnablePluginById(pluginId);

        cy.apiGetConfig().then(({config}) => {
            const pluginSettings = config.PluginSettings.Plugins[pluginId];
            settingsWithGenerated = {
                ...defaultPluginConfig,
                encryptionkey: pluginSettings.encryptionkey,
                webhooksecret: pluginSettings.webhooksecret,
            }

            // Check if default config values were set
            expect(pluginSettings.encryptionkey).to.not.equal('');
            expect(pluginSettings.webhooksecret).to.not.equal('');

            // Make sure we're starting with a clean config otherwise, for each test
            expect(pluginSettings).to.deep.equal(settingsWithGenerated);
        });

        cy.visit(`/${testTeam.name}/messages/@${botUsername}`);
    });

    it.only('GitHub setup flow, register OAuth application', () => {
        cy.get('#post_textbox').clear().type(`/${slashCommandName} setup`);
        cy.get('#post_textbox').type('{enter}');
        cy.get('#post_textbox').type('{enter}');

        let steps = [
            ['Continue', 'Welcome to GitHub integration'],
            [`myself`, 'Are you setting this GitHub integration up'],
            ['No', 'GitHub Enterprise'],
            ['Continue', 'Register an OAuth Application'],
            ['Continue', 'Please enter the GitHub OAuth Client ID'],
        ]
        steps.forEach(handleClickStep);

        // Enter credentials into interactive dialog
        cy.get('input#client_id').type('a'.repeat(20));
        cy.get('input#client_secret').type('a'.repeat(40));
        cy.get('button#interactiveDialogSubmit').click();

        steps = [
            ['', 'Connect your GitHub account'],
        ]
        steps.forEach(handleClickStep);

        cy.apiGetConfig().then(({config}) => {
            const pluginSettings = config.PluginSettings.Plugins[pluginId];

            // expect(objectsEqual(pluginSettings, {
            //     ...settingsWithGenerated,
            //     githuboauthclientid: 'a'.repeat(20),
            //     githuboauthclientsecret: 'a'.repeat(40),
            // })).to.be.true

            expect(pluginSettings).to.deep.equal({
                ...settingsWithGenerated,
                githuboauthclientid: 'a'.repeat(20),
                githuboauthclientsecret: 'a'.repeat(40),
            });
        });
    });
});

function handleClickStep(testCase) {
    const [buttonText, expectedContent] = testCase;

    cy.getLastPostId().then((lastPostId) => {
        if (expectedContent) {
            cy.getLastPostId().then((lastPostId) => {
                cy.get(`#post_${lastPostId}`).contains(expectedContent);
            });
        }

        if (buttonText) {
            cy.get(`#${lastPostId}_message`).contains('button:enabled', buttonText).click();
        }
    });
}

// Used for debugging deep equal checks, since Cypress doesn't provide a reason why the check fails.
const objectsEqual = (obj1, obj2) => {
    const keys1 = Object.keys(obj1);
    const keys2 = Object.keys(obj2);
    if (keys1.length !== keys2.length) {
        return false;
    }

    for (const key of keys1) {
        if (obj1[key] !== obj2[key]) {
            return key;
        }
    }

    return true;
}
