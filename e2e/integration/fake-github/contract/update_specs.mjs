// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Trims GitHub's published REST description (OpenAPI, with webhook payloads) and GraphQL schema
// down to what fake-github serves.
//   node update_specs.mjs          rewrite specs/ from upstream
//   node update_specs.mjs --check  compare upstream with the committed copies: exit 1 on breaking
//                                  changes (drift.mjs), 2 when a download fails, and 0 with additive
//                                  changes listed for the next update

/* eslint-disable no-console */

import fs from 'node:fs/promises';
import path from 'node:path';
import {fileURLToPath} from 'node:url';

import {Kind, assertValidSchema, buildASTSchema, lexicographicSortSchema, parse, printSchema, specifiedScalarTypes} from 'graphql';

import {compareGraphQL, compareRest} from './drift.mjs';

const dir = path.dirname(fileURLToPath(import.meta.url));
const specsDir = path.join(dir, 'specs');
const check = process.argv.includes('--check');
const manifest = JSON.parse(await fs.readFile(path.join(dir, 'operations.json'), 'utf8'));

// A download failure says nothing about drift, so it gets its own exit code; the nightly workflow
// only opens a drift issue on exit 1.
let restSpec;
let graphqlSdl;
try {
    restSpec = await download(manifest.rest.url, 'json');
    graphqlSdl = await download(manifest.graphql.url, 'text');
} catch (err) {
    console.error(err.message);
    process.exit(2);
}

// trimRest and trimGraphQL throw a ContractGone when an operation, webhook, type or field the fake
// serves is gone upstream. That is breaking drift. Any other error (say, an unexpected document
// shape) is a script failure: exit 2, so it doesn't open a drift issue.
class ContractGone extends Error {}

const breaking = [];
const additive = [];
const outputs = {};
for (const [file, trim] of [
    ['rest.json', () => `${JSON.stringify(trimRest(restSpec, manifest.rest), null, 2)}\n`],
    ['schema.graphql', () => trimGraphQL(graphqlSdl, manifest.graphql.fields)],
]) {
    try {
        outputs[file] = trim();
    } catch (err) {
        if (!check) {
            throw err;
        }
        if (!(err instanceof ContractGone)) {
            console.error(`${file}: trimming failed: ${err.stack}`);
            process.exit(2);
        }
        breaking.push(`${file}: ${err.message}`);
    }
}

for (const [file, next] of Object.entries(outputs)) {
    const outPath = path.join(specsDir, file);
    if (!check) {
        await fs.mkdir(specsDir, {recursive: true});
        await fs.writeFile(outPath, next);
        console.log(`${file}: updated`);
        continue;
    }
    const current = await fs.readFile(outPath, 'utf8').catch(() => null);
    if (current === null) {
        breaking.push(`${file}: no committed copy; run \`npm run contract:update\``);
        continue;
    }
    if (current === next) {
        continue;
    }
    let drift;
    try {
        drift = file === 'rest.json' ? compareRest(JSON.parse(current), JSON.parse(next)) : compareGraphQL(current, next);
    } catch (err) {
        // A bug in the comparison is not drift: exit 2 so it doesn't open a drift issue.
        console.error(`${file}: comparing failed: ${err.stack}`);
        process.exit(2);
    }
    breaking.push(...drift.breaking.map((line) => `${file}: ${line}`));
    additive.push(...drift.additive.map((line) => `${file}: ${line}`));
    if (!drift.breaking.length && !drift.additive.length) {
        additive.push(`${file}: changed, but only in key order or formatting`);
    }
}

if (check) {
    for (const line of breaking) {
        console.error(`BREAKING ${line}`);
    }
    for (const line of additive) {
        console.log(`additive ${line}`);
    }
    if (breaking.length) {
        console.error(`\n${breaking.length} breaking change(s). Run \`npm run contract:update\`, review the diff, and update fake-github and the plugin to match.`);
        process.exit(1);
    }
    console.log(additive.length ? `\n${additive.length} additive change(s); run \`npm run contract:update\` to pull them in.` : 'No drift.');
}

async function download(url, as) {
    const res = await fetch(url);
    if (!res.ok) {
        throw new Error(`Failed to download ${url}: ${res.status}`);
    }
    return as === 'json' ? res.json() : res.text();
}

function trimRest(spec, {operations, webhooks}) {
    const paths = {};
    for (const operation of operations) {
        const [method, opPath] = operation.split(' ');
        const op = spec.paths[opPath]?.[method.toLowerCase()];
        if (!op) {
            throw new ContractGone(`rest: ${operation} is not in the upstream description`);
        }
        paths[opPath] = {...paths[opPath], [method.toLowerCase()]: stripDocs({requestBody: op.requestBody, responses: op.responses})};
    }

    const hooks = {};
    for (const name of webhooks) {
        const op = spec['x-webhooks'][name]?.post;
        if (!op) {
            throw new ContractGone(`rest: webhook ${name} is not in the upstream description`);
        }
        hooks[name] = {post: stripDocs({requestBody: op.requestBody})};
    }

    const components = {};
    const queue = collectRefs([paths, hooks]);
    while (queue.length) {
        const [kind, name] = queue.pop().replace('#/components/', '').split('/');
        if (components[kind]?.[name]) {
            continue;
        }
        components[kind] = {...components[kind], [name]: stripDocs(spec.components[kind][name])};
        queue.push(...collectRefs(components[kind][name]));
    }

    return {
        openapi: spec.openapi,
        info: {title: spec.info.title},
        paths,
        'x-webhooks': hooks,
        components: Object.fromEntries(Object.keys(components).sort().map((kind) => [kind, sortKeys(components[kind])])),
    };
}

// GitHub edits prose and examples often; dropping them keeps --check focused on shape changes.
function stripDocs(node, isPropertyMap = false) {
    if (Array.isArray(node)) {
        return node.map((item) => stripDocs(item));
    }
    if (!node || typeof node !== 'object') {
        return node;
    }
    const out = {};
    for (const [key, value] of Object.entries(node)) {
        if (!isPropertyMap && (['description', 'example', 'examples', 'summary', 'externalDocs', 'title'].includes(key) || key.startsWith('x-'))) {
            continue;
        }
        out[key] = stripDocs(value, !isPropertyMap && key === 'properties');
    }
    return out;
}

function collectRefs(node, found = []) {
    if (Array.isArray(node)) {
        node.forEach((item) => collectRefs(item, found));
    } else if (node && typeof node === 'object') {
        for (const [key, value] of Object.entries(node)) {
            if (key === '$ref' && typeof value === 'string') {
                found.push(value);
            } else {
                collectRefs(value, found);
            }
        }
    }
    return found;
}

function sortKeys(object) {
    return Object.fromEntries(Object.entries(object).sort(([a], [b]) => a.localeCompare(b)));
}

// Keeps the listed object/interface fields, every enum, scalar and input type they reference, and unions
// narrowed to listed members. Querying anything else then fails validation, like an uncovered REST route.
function trimGraphQL(sdl, fields) {
    const definitions = new Map(parse(sdl).definitions.filter((d) => 'name' in d && d.name).map((d) => [d.name.value, d]));
    const builtins = new Set(specifiedScalarTypes.map((t) => t.name));
    const kept = new Map();
    const queue = ['Query'];

    while (queue.length) {
        const name = queue.pop();
        if (kept.has(name) || builtins.has(name)) {
            continue;
        }
        const def = definitions.get(name);
        if (!def) {
            throw new ContractGone(`graphql: ${name} is not in the upstream schema`);
        }

        let trimmed = stripDirectives({...def, description: undefined});
        if (def.kind === Kind.OBJECT_TYPE_DEFINITION || def.kind === Kind.INTERFACE_TYPE_DEFINITION) {
            const wanted = fields[name];
            if (!wanted) {
                throw new ContractGone(`graphql: ${name} is reachable from a listed field; add it to operations.json`);
            }
            const fieldDefs = def.fields.filter((f) => wanted.includes(f.name.value));
            const missing = wanted.filter((f) => !fieldDefs.some((d) => d.name.value === f));
            if (missing.length) {
                throw new ContractGone(`graphql: ${name} has no field(s) ${missing.join(', ')}`);
            }
            trimmed = {
                ...trimmed,
                interfaces: (def.interfaces ?? []).filter((i) => fields[i.name.value]),
                fields: fieldDefs.map((f) => stripDirectives({...f, description: undefined, arguments: f.arguments?.map((a) => stripDirectives({...a, description: undefined}))})),
            };
            for (const f of trimmed.fields) {
                queue.push(namedType(f.type), ...(f.arguments ?? []).map((a) => namedType(a.type)));
            }
            queue.push(...trimmed.interfaces.map((i) => i.name.value));
        } else if (def.kind === Kind.UNION_TYPE_DEFINITION) {
            trimmed = {...trimmed, types: def.types.filter((t) => fields[t.name.value])};
            queue.push(...trimmed.types.map((t) => t.name.value));
        } else if (def.kind === Kind.INPUT_OBJECT_TYPE_DEFINITION) {
            trimmed = {...trimmed, fields: def.fields.map((f) => stripDirectives({...f, description: undefined}))};
            queue.push(...trimmed.fields.map((f) => namedType(f.type)));
        } else if (def.kind === Kind.ENUM_TYPE_DEFINITION) {
            trimmed = {...trimmed, values: def.values.map((v) => stripDirectives({...v, description: undefined}))};
        }
        kept.set(name, trimmed);
    }

    const schema = buildASTSchema({kind: Kind.DOCUMENT, definitions: [...kept.values()]});
    assertValidSchema(schema);
    return `${printSchema(lexicographicSortSchema(schema))}\n`;
}

function namedType(type) {
    return type.kind === Kind.NAMED_TYPE ? type.name.value : namedType(type.type);
}

function stripDirectives(node) {
    return {...node, directives: (node.directives ?? []).filter((d) => d.name.value === 'deprecated')};
}
