// Express middleware used to log URLs, headers, JSON requests and responses

module.exports = (req, res, next) => {
    const headers = req.headers ? stringify(req.headers) : 'no headers';

    const body = req.body ? stringify(req.body) : 'no body';

    const output = `\n${req.method} ${req.url}\nHeaders: ${headers}\nRequest: ${body}`;
    console.log(output);

    // Override send method to log response
    const send = res.send.bind(res);
    res.send = (...args) => {
        const output = stringify(args[0]);
        console.log(`Response: ${res.statusCode} ${output}`);

        // Finish sending response
        send(...args);
    }

    next();
}

const stringify = (data) => {
    try {
        return JSON.stringify(data, null, 2);
    } catch (e) {
        return data.toString();
    }
}
