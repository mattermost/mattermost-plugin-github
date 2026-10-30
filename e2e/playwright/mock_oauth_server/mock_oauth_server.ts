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

    // E2E builds also send REST, GraphQL and upload calls here; this suite expects them to reach GitHub.
    app.use('/api', forwardTo('https://api.github.com'));
    app.use('/uploads', forwardTo('https://uploads.github.com'));

    return app;
};

const hopByHopHeaders = ['host', 'connection', 'content-length', 'content-encoding', 'transfer-encoding'];

function forwardTo(target: string): express.RequestHandler[] {
    return [express.raw({type: () => true, limit: '25mb'}), async (req, res) => {
        const headers = Object.fromEntries(Object.entries(req.headers).filter(([name]) => !hopByHopHeaders.includes(name))) as Record<string, string>;
        const hasBody = Buffer.isBuffer(req.body) && req.body.length > 0;

        // Express 4 doesn't catch rejected handlers; an unreachable GitHub would otherwise crash the mock.
        let upstream: Awaited<ReturnType<typeof fetch>>;
        try {
            upstream = await fetch(target + req.url, {method: req.method, headers, body: hasBody ? req.body : undefined});
        } catch (err) {
            res.status(502).send(`mock OAuth server could not reach ${target}: ${err}`);
            return;
        }

        upstream.headers.forEach((value, name) => {
            if (!hopByHopHeaders.includes(name)) {
                res.setHeader(name, value);
            }
        });
        res.status(upstream.status).send(Buffer.from(await upstream.arrayBuffer()));
    }];
}
