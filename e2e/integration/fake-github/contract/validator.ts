// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import fs from 'node:fs';
import path from 'node:path';

import Ajv, {type ErrorObject, type ValidateFunction} from 'ajv';
import addFormats from 'ajv-formats';
import {type DocumentNode, type GraphQLSchema, buildSchema, parse, validate} from 'graphql';

type JsonObject = Record<string, any>;

const specsDir = path.join(__dirname, 'specs');
const rest: JsonObject = JSON.parse(fs.readFileSync(path.join(specsDir, 'rest.json'), 'utf8'));

export const graphQLSchema: GraphQLSchema = buildSchema(fs.readFileSync(path.join(specsDir, 'schema.graphql'), 'utf8'));

// GitHub answers any authenticated route with these, documented or not.
const basicError = {
    type: 'object',
    required: ['message'],
    properties: {message: {type: 'string'}, documentation_url: {type: 'string'}, status: {type: 'string'}},
};
const apiWideResponses: Record<string, JsonObject> = {401: basicError, 403: basicError, 404: basicError};

const ajv = new Ajv({strict: false, allErrors: true, validateFormats: true});
addFormats(ajv);
for (const format of ['uri-template', 'repo.nwo', 'ssh-key', 'ssh-key fingerprint']) {
    ajv.addFormat(format, true);
}
ajv.addSchema({components: fromOpenAPI(rest.components)}, 'rest');

type Operation = {method: string; template: string; pattern: RegExp; request?: ValidateFunction; responses: Map<string, ValidateFunction>};

const operations: Operation[] = Object.entries(rest.paths as JsonObject).flatMap(([template, methods]) => Object.entries(methods as JsonObject).map(([method, op]) => ({
    method: method.toUpperCase(),
    template,
    pattern: templateToRegExp(template),
    request: jsonSchema(op.requestBody) && ajv.compile(fromOpenAPI(jsonSchema(op.requestBody))),
    responses: new Map(Object.entries(op.responses as JsonObject).flatMap(([status, response]) => {
        const schema = jsonSchema(resolve(response));
        return schema ? [[status, ajv.compile(fromOpenAPI(schema))]] : [];
    })),
})));

const webhooks = new Map<string, ValidateFunction>(Object.entries(rest['x-webhooks'] as JsonObject).map(([name, hook]) => [name, ajv.compile(fromOpenAPI(jsonSchema(hook.post.requestBody)))]));
const apiWide = new Map(Object.entries(apiWideResponses).map(([status, schema]) => [status, ajv.compile(schema)]));

export function findOperation(method: string, pathname: string): Operation | undefined {
    return operations.find((op) => op.method === method && op.pattern.test(pathname));
}

export function validateRequest(method: string, pathname: string, body: unknown): string[] {
    const op = findOperation(method, pathname);
    if (!op) {
        return [`${method} ${pathname} is not in operations.json`];
    }
    return op.request ? describe(op.request, body, `${method} ${op.template} request`) : [];
}

export function validateResponse(method: string, pathname: string, status: number, body: unknown): string[] {
    const op = findOperation(method, pathname);
    if (!op) {
        return [`${method} ${pathname} is not in operations.json`];
    }
    const validator = op.responses.get(String(status)) ?? apiWide.get(String(status));
    if (!validator) {
        if (op.responses.has(String(status)) || status === 204 || status === 304) {
            return [];
        }
        return [`${method} ${op.template}: GitHub does not document a ${status} response`];
    }
    return describe(validator, body, `${method} ${op.template} ${status} response`);
}

// GitHub names webhook payloads after the X-GitHub-Event header plus the action, e.g. pull-request-opened.
export function webhookSchemaName(event: string, action?: string): string | undefined {
    const base = event.replaceAll('_', '-');
    return [action && `${base}-${action.replaceAll('_', '-')}`, base].find((name) => name && webhooks.has(name)) || undefined;
}

export function validateWebhook(event: string, payload: JsonObject): string[] {
    const name = webhookSchemaName(event, payload.action);
    if (!name) {
        return [`webhook ${event}${payload.action ? `.${payload.action}` : ''} is not in operations.json`];
    }
    return describe(webhooks.get(name)!, payload, `webhook ${name}`);
}

export function validateGraphQLQuery(query: string): {document?: DocumentNode; errors: string[]} {
    let document: DocumentNode;
    try {
        document = parse(query);
    } catch (err) {
        return {errors: [(err as Error).message]};
    }
    return {document, errors: validate(graphQLSchema, document).map((e) => e.message).filter((m) => !githubAcceptsShapeConflict.test(m))};
}

// graphql-js rejects `... on Issue { state } ... on PullRequest { state }` (IssueState vs PullRequestState);
// GitHub's graphql-ruby server answers it, and the plugin's sidebar query depends on that.
const githubAcceptsShapeConflict = /^Fields "state" conflict because they return conflicting types "(IssueState|PullRequestState)!?" and "(IssueState|PullRequestState)!?"\./;

export const skeletonTimestamp = '2026-01-01T00:00:00Z';

// A minimal value for a component schema with every required field filled in. Callers overlay real
// state on top, so GitHub adding a required field changes the fixture instead of breaking it.
export function skeleton(schemaName: string): any {
    const schema = rest.components.schemas[schemaName];
    if (!schema) {
        throw new Error(`component schema ${schemaName} is not in the trimmed spec; reference it from an operation in operations.json`);
    }
    return fill(schema, 0);
}

export function webhookSkeleton(name: string): any {
    return fill(jsonSchema(rest['x-webhooks'][name].post.requestBody)!, 0);
}

function fill(node: JsonObject, depth: number): any {
    const schema = resolve(node) ?? {};
    if (schema.nullable || depth > 15) {
        return null;
    }
    if (schema.allOf) {
        return schema.allOf.reduce((acc: any, part: JsonObject) => merge(acc, fill(part, depth + 1)), {});
    }
    const branches = schema.anyOf ?? schema.oneOf;
    if (branches) {
        return fill(branches.find((b: JsonObject) => resolve(b)?.type !== 'null') ?? branches[0], depth + 1);
    }
    if (schema.enum) {
        return schema.enum[0];
    }
    if (schema.default !== undefined && schema.default !== null) {
        return schema.default;
    }
    switch (schema.type ?? (schema.properties ? 'object' : undefined)) {
    case 'object':
        return Object.fromEntries((schema.required ?? []).map((key: string) => [key, fill(schema.properties?.[key] ?? {}, depth + 1)]));
    case 'array':
        return Array.from({length: schema.minItems ?? 0}, () => fill(schema.items ?? {}, depth + 1));
    case 'integer':
    case 'number':
        return schema.minimum ?? 0;
    case 'boolean':
        return false;
    case 'string':
        return ({'date-time': skeletonTimestamp, date: skeletonTimestamp.slice(0, 10), uri: 'https://github.com/', email: 'e2e@example.com'} as Record<string, string>)[schema.format] ?? '';
    default:
        return null;
    }
}

export function merge(base: any, overlay: any): any {
    if (!isPlainObject(base) || !isPlainObject(overlay)) {
        return overlay === undefined ? base : overlay;
    }
    const out = {...base};
    for (const [key, value] of Object.entries(overlay)) {
        out[key] = merge(base[key], value);
    }
    return out;
}

function isPlainObject(value: unknown): value is JsonObject {
    return Boolean(value) && typeof value === 'object' && !Array.isArray(value);
}

function describe(validator: ValidateFunction, value: unknown, label: string): string[] {
    if (validator(value)) {
        return [];
    }
    return (validator.errors ?? []).slice(0, 10).map((e: ErrorObject) => `${label}: ${e.instancePath || '/'} ${e.message}${e.params ? ` ${JSON.stringify(e.params)}` : ''}`);
}

function resolve(node: JsonObject): JsonObject {
    if (!node?.$ref) {
        return node;
    }
    const [kind, name] = node.$ref.replace('#/components/', '').split('/');
    return resolve(rest.components[kind][name]);
}

function jsonSchema(body: JsonObject | undefined): JsonObject | undefined {
    const content = resolve(body as JsonObject)?.content;
    return content?.['application/json']?.schema ?? content?.['application/scim+json']?.schema;
}

// OpenAPI 3.0 marks optional nulls with `nullable`; ajv only understands JSON Schema's `type: [.., 'null']`.
function fromOpenAPI(node: any): any {
    if (Array.isArray(node)) {
        return node.map(fromOpenAPI);
    }
    if (!node || typeof node !== 'object') {
        return node;
    }
    const out: JsonObject = {};
    for (const [key, value] of Object.entries(node)) {
        if (key === '$ref') {
            out.$ref = (value as string).replace(/^#\/components\//, 'rest#/components/');
        } else if (key !== 'nullable') {
            out[key] = key === 'properties' || key === 'patternProperties' ?
                Object.fromEntries(Object.entries(value as JsonObject).map(([k, v]) => [k, fromOpenAPI(v)])) :
                fromOpenAPI(value);
        }
    }
    if (node.nullable !== true) {
        return out;
    }
    if (typeof out.type === 'string' && !out.allOf && !out.anyOf && !out.oneOf) {
        return {...out, type: [out.type, 'null'], ...(out.enum ? {enum: [...out.enum, null]} : {})};
    }
    return {anyOf: [out, {type: 'null'}]};
}

function templateToRegExp(template: string): RegExp {
    const escaped = template.replace(/[.*+?^$()|[\]\\]/g, '\\$&').replace(/\{[^}]+\}/g, '[^/]+');
    return new RegExp(`^${escaped}$`);
}
