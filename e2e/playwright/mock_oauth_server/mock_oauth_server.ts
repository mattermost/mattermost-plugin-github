import express from 'express';

export enum ExpiryAlgorithms {
    NO_EXPIRY = 'no_expiry',
}

export type OAuthServerOptions = {
    mattermostSiteURL: string;
    pluginId: string;
    mockOAuthAccessToken: string;
    authorizeURLPrefix: string;
    expiryAlgorithm: ExpiryAlgorithms;
}

export const makeOAuthServer = ({
    mattermostSiteURL,
    pluginId,
    mockOAuthAccessToken,
    authorizeURLPrefix,
    expiryAlgorithm,
}: OAuthServerOptions): express.Express => {
    if (!mockOAuthAccessToken) {
        throw new Error('MockOAuthServer: Please provide an OAuth access token to use');
    }

    if (expiryAlgorithm !== ExpiryAlgorithms.NO_EXPIRY) {
        throw new Error(`MockOAuthServer: Unsupported OAuth token expiry algorithm: ${expiryAlgorithm}`);
    }

    const app = express();

    // eslint-disable-next-line new-cap
    const oauthRouter = express.Router();

    oauthRouter.get('/authorize', (req, res) => {
        const query = req.url.split('?')[1];
        res.redirect(`${mattermostSiteURL}/plugins/${pluginId}/oauth/complete?${query}&code=1234`);
    });

    oauthRouter.post('/access_token', (req, res) => {
        const token = {
            access_token: mockOAuthAccessToken,
            token_type: 'bearer',
            expiry: '0001-01-01T00:00:00Z',
        };

        res.json(token);
    });

    app.use(authorizeURLPrefix, oauthRouter);

    return app;
};
