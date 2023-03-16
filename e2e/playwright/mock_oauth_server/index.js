const express = require('express');
const app = express();

app.use(express.json());
app.use(express.urlencoded());

if (process.env.NODE_ENV === 'development') {
    app.use(require('./logger'));
}

const oauthRouter = express.Router();

const siteURL = process.env.MM_SITE_URL || 'http://localhost:8065';
console.log(`Using Mattermost Site URL: ${siteURL}`);

const pluginId = process.env.MM_PLUGIN_ID || 'github';
console.log(`Running OAuth server for plugin: ${pluginId}`);

const accessToken = process.env.MOCK_OAUTH_ACCESS_TOKEN;
if(!accessToken) {
    console.error('Error: Please provide an access token via environment variable MOCK_OAUTH_ACCESS_TOKEN\n\n');
    process.exit(1);
}

oauthRouter.get('/authorize', function (req, res) {
    const query = req.url.split('?')[1];
    res.redirect(`${siteURL}/plugins/${pluginId}/oauth/complete?${query}&code=1234`);
});

oauthRouter.post('/access_token', function (req, res) {
    const token = {
        access_token: accessToken,
        token_type: 'bearer',
        expiry: '0001-01-01T00:00:00Z',
    };

    res.json(token);
});

app.use('/login/oauth', oauthRouter);

const port = process.env.OAUTH_SERVER_PORT || 8080;
app.listen(port, () => {
    console.log(`Listening on ${port}`);
});
