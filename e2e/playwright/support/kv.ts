import {Client} from 'pg';

const DATABASE_CONNECTION_STRING = process.env.MM_DATABASE_CONNECTION_STRING || 'postgres://mmuser:mostest@postgres/mattermost_test';

export function getClient() {
    const client = new Client({
        host: 'localhost',
        user: 'mmuser',
        password: 'mostest',
        database: 'mattermost_test',
        // connectionString: DATABASE_CONNECTION_STRING,
    });

    client.connect();
    client.end = client.end.bind(client);
    return client;
}

export const getKV = async <T>(key: string, pluginId: string): Promise<T | null> => {
    console.log(`Getting KV entry for '${key}'`)
    const client = await getClient();

    const query = `SELECT pkey, pvalue from PluginKeyValueStore WHERE pkey = $1 AND pluginid = $2`;

    const result = await client.query(query, [key, pluginId])
        .finally(client.end);

    if (!result.rows.length) {
        return null;
    }

    const value = result.rows[0].pvalue?.toString();
    try {
        return JSON.parse(value);
    } catch (e) {
        return value;
    }
}

export const setKV = async <T>(key: string, value: T, pluginId: string): Promise<void> => {
    console.log(`Setting KV entry '${key}'`);

    const existingEntry = await getKV(key, pluginId);

    let query = 'INSERT INTO PluginKeyValueStore (pluginid, pkey, pvalue, expireat) VALUES ($1, $2, $3, $4)';
    let values = [pluginId, key, value, 0];
    if (existingEntry) {
        query = 'UPDATE PluginKeyValueStore SET pvalue = $1 WHERE pkey = $2 AND pluginid = $3';
        values = [value, key, pluginId];
    }

    const client = getClient();
    await client.query(query, values)
        .finally(client.end);
}

export const clearKVStoreForPlugin = async (pluginId: string) => {
    const client = getClient();

    // avoid clearing bot user id
    const botUserKey = 'mmi_botid';

    const query = 'DELETE from PluginKeyValueStore WHERE pkey != $1 AND pluginid = $2';

    await client.query(query, [botUserKey, pluginId])
        .finally(client.end);
}
