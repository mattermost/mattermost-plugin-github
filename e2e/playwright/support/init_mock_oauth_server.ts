// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

/* eslint-disable no-process-exit */
/* eslint-disable no-console */
/* eslint-disable no-process-env */
import {ExpiryAlgorithms, makeOAuthServer} from '../mock_oauth_server/mock_oauth_server';

const mockOAuthAccessToken = process.env.PLUGIN_E2E_MOCK_OAUTH_TOKEN;
if (!mockOAuthAccessToken) {
    console.error('Please provide an OAuth access token to use via env var PLUGIN_E2E_MOCK_OAUTH_TOKEN');
    process.exit(1);
}

export const runOAuthServer = async () => {
    const defaultAuthorizePrefix = '/login/oauth'; // Used by GitHub
    const authorizeURLPrefix = process.env.OAUTH_AUTHORIZE_URL_PREFIX || defaultAuthorizePrefix;

    const mattermostSiteURL = process.env.MM_SERVICESETTINGS_SITEURL || 'http://localhost:8065';
    const pluginId = process.env.MM_PLUGIN_ID || 'github';

    const app = makeOAuthServer({
        authorizeURLPrefix,
        mattermostSiteURL,
        mockOAuthAccessToken,
        pluginId,
        expiryAlgorithm: ExpiryAlgorithms.NO_EXPIRY,
    });

    const port = process.env.OAUTH_SERVER_PORT || 8080;
    app.listen(port, () => {
        console.log(`Mock OAuth server listening on port ${port}`);
    });
};
