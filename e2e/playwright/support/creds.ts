export const mockOAuthAccessToken = process.env.PLUGIN_E2E_MOCK_OAUTH_TOKEN;
if (!mockOAuthAccessToken) {
    console.error('Please provide an OAuth access token to use via env var PLUGIN_E2E_MOCK_OAUTH_TOKEN');
    process.exit(1);
}
