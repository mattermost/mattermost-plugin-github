// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Client} from 'pg';

export function getClient() {
    const client = new Client({
        host: 'localhost',
        user: 'mmuser',
        password: 'mostest',
        database: 'mattermost_test',
    });

    client.connect();
    client.end = client.end.bind(client);
    return client;
}

export const clearKVStoreForPlugin = async (pluginId: string) => {
    const client = getClient();

    // avoid clearing bot user id
    const botUserKey = 'mmi_botid';

    const query = 'DELETE from PluginKeyValueStore WHERE pkey != $1 AND pluginid = $2';

    await client.query(query, [botUserKey, pluginId]).
        finally(client.end);
};
