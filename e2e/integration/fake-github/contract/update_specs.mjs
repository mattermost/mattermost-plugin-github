// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Trims GitHub's published REST description (OpenAPI, with webhook payloads) and GraphQL schema
// down to what fake-github serves.
//   node update_specs.mjs          rewrite specs/ from upstream
//   node update_specs.mjs --check  exit 1 if upstream drifted from the committed copies

/* eslint-disable no-console */

import fs from 'node:fs/promises';
import path from 'node:path';
import {fileURLToPath} from 'node:url';

import {Kind, assertValidSchema, buildASTSchema, lexicographicSortSchema, parse, printSchema, specifiedScalarTypes} from 'graphql';

const dir = path.dirname(fileURLToPath(import.meta.url));
const specsDir = path.join(dir, 'specs');
const check = process.argv.includes('--check');
const manifest = JSON.parse(await fs.readFile(path.join(dir, 'operations.json'), 'utf8'));

const outputs = {
    'rest.json': `${JSON.stringify(trimRest(await download(manifest.rest.url, 'json'), manifest.rest), null, 2)}\n`,
    'schema.graphql': trimGraphQL(await download(manifest.graphql.url, 'text'), manifest.graphql.fields),
};

let drifted = false;
for (const [file, next] of Object.entries(outputs)) {
    const outPath = path.join(specsDir, file);
    if (check) {
        if (await fs.readFile(outPath, 'utf8').catch(() => '') !== next) {
            drifted = true;
            console.error(`${file}: GitHub changed what fake-github serves`);
        }
        continue;
    }
    await fs.mkdir(specsDir, {recursive: true});
    await fs.writeFile(outPath, next);
    console.log(`${file}: updated`);
}

if (drifted) {
    console.error('Run `npm run contract:update`, review the diff, and update fake-github to match.');
    process.exit(1);
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
            throw new Error(`rest: ${operation} is not in the upstream description`);
        }
        paths[opPath] = {...paths[opPath], [method.toLowerCase()]: stripDocs({requestBody: op.requestBody, responses: op.responses})};
    }

    const hooks = {};
    for (const name of webhooks) {
        const op = spec['x-webhooks'][name]?.post;
        if (!op) {
            throw new Error(`rest: webhook ${name} is not in the upstream description`);
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
            throw new Error(`graphql: ${name} is not in the upstream schema`);
        }

        let trimmed = stripDirectives({...def, description: undefined});
        if (def.kind === Kind.OBJECT_TYPE_DEFINITION || def.kind === Kind.INTERFACE_TYPE_DEFINITION) {
            const wanted = fields[name];
            if (!wanted) {
                throw new Error(`graphql: ${name} is reachable from a listed field; add it to operations.json`);
            }
            const fieldDefs = def.fields.filter((f) => wanted.includes(f.name.value));
            const missing = wanted.filter((f) => !fieldDefs.some((d) => d.name.value === f));
            if (missing.length) {
                throw new Error(`graphql: ${name} has no field(s) ${missing.join(', ')}`);
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
