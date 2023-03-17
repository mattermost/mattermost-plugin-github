import type {Handler} from 'express';

// Express middleware used to log URLs, headers, JSON requests and responses

const loggerMiddleware: Handler = (req, res, next) => {
    const headers = req.headers ? stringify(req.headers) : 'no headers';

    const body = req.body ? stringify(req.body) : 'no body';

    const output = `\n${req.method} ${req.url}\nHeaders: ${headers}\nRequest: ${body}`;
    console.log(output);

    // type Send = typeof res.send;

    // Override send method to log response
    const send = res.send.bind(res);
    res.send = ((...args: any[]) => {
        const output = stringify(body);
        console.log(`Response: ${res.statusCode} ${output}`);

        // Finish sending response
        send(...args);
    }) as any;

    next();
}

const stringify = (data: unknown): string => {
    try {
        return JSON.stringify(data, null, 2);
    } catch (e) {}

    return new String(data).toString();
}

export default loggerMiddleware;
