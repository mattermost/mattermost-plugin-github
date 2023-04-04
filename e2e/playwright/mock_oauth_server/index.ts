require('dotenv').config();

import {ExpiryAlgorithms, makeOAuthServer} from './mock_oauth_server';

const mockOAuthAccessToken = process.env.MOCK_OAUTH_ACCESS_TOKEN;
if (!mockOAuthAccessToken) {
    console.error('Please provide an OAuth access token to use');
    process.exit(0);
}

const defaultAuthorizePrefix = '/login/oauth' // Used by GitHub
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
