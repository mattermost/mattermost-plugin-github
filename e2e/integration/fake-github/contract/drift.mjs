// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Sorts the differences between the committed trimmed specs and GitHub's current ones into breaking
// and additive changes, so the nightly check only fails when fake-github or the plugin could stop
// matching GitHub. GitHub updates its REST description several times a week, mostly by adding
// optional fields; those are reported, not failed on.

import {buildSchema, findBreakingChanges, findDangerousChanges} from 'graphql';

// Schema keywords that narrow what is valid when added to a schema that lacked them.
const RESTRICTIONS = new Set([
    'type', '$ref', 'allOf', 'anyOf', 'oneOf', 'not', 'const', 'enum', 'required', 'nullable', 'pattern', 'format',
    'minLength', 'maxLength', 'minimum', 'maximum', 'exclusiveMinimum', 'exclusiveMaximum', 'multipleOf',
    'minItems', 'maxItems', 'uniqueItems', 'minProperties', 'maxProperties', 'additionalProperties', 'items', 'requestBody',
]);
// Maps whose keys are names (fields, status codes, media types), not schema keywords.
const NAME_MAPS = new Set(['properties', 'responses', 'content', 'paths', 'x-webhooks', 'schemas']);
const VARIANTS = new Set(['allOf', 'anyOf', 'oneOf']);

/**
 * Compares two trimmed REST descriptions as JSON trees: anything removed or changed is breaking,
 * anything added is additive unless it is a restriction keyword. In `required`, any change is
 * breaking (a request may now lack a field, or a response may omit one); in `enum`, only removals
 * are. allOf/anyOf/oneOf variants are compared regardless of order.
 * @returns {{breaking: string[], additive: string[]}}
 */
export function compareRest(current, next) {
    const out = {breaking: [], additive: []};
    diff(current, next, '', '', out);
    return out;
}

// `key` is the keyword `a` and `b` sit under; `inNames` says their own keys are names, not keywords.
function diff(a, b, at, key, out, inNames = false) {
    if (Array.isArray(a) && Array.isArray(b) && (key === 'required' || key === 'enum')) {
        const removed = a.filter((v) => !b.includes(v));
        const added = b.filter((v) => !a.includes(v));
        if (removed.length) {
            out.breaking.push(`${at} removed: ${removed.join(', ')}`);
        }
        if (added.length) {
            (key === 'required' ? out.breaking : out.additive).push(`${at} added: ${added.join(', ')}`);
        }
        return;
    }
    if (Array.isArray(a) && Array.isArray(b) && VARIANTS.has(key)) {
        diffVariants(a, b, at, key, out);
        return;
    }
    if (!isContainer(a) || !isContainer(b) || Array.isArray(a) !== Array.isArray(b)) {
        if (JSON.stringify(a) !== JSON.stringify(b)) {
            out.breaking.push(`${at}: ${JSON.stringify(a)} -> ${JSON.stringify(b)}`);
        }
        return;
    }
    for (const k of Object.keys(a)) {
        if (k in b) {
            diff(a[k], b[k], `${at}.${k}`, k, out, !inNames && NAME_MAPS.has(k));
        } else {
            out.breaking.push(`${at}.${k} removed`);
        }
    }
    for (const k of Object.keys(b)) {
        if (!(k in a)) {
            (!inNames && RESTRICTIONS.has(k) ? out.breaking : out.additive).push(`${at}.${k} added`);
        }
    }
}

// Unordered: a reordered list is no change. A removed variant is breaking; an added one is breaking
// for allOf (one more constraint) and additive for anyOf/oneOf (one more accepted shape). When the
// same number of variants differ, they were edited in place, so compare them by position.
function diffVariants(a, b, at, key, out) {
    const as = a.map(canonical);
    const bs = b.map(canonical);
    const removed = as.filter((v) => !bs.includes(v));
    const added = bs.filter((v) => !as.includes(v));
    if (a.length === b.length && removed.length) {
        a.forEach((variant, i) => diff(variant, b[i], `${at}.${i}`, '', out));
        return;
    }
    if (removed.length) {
        out.breaking.push(`${at}: ${removed.length} variant(s) removed`);
    }
    if (added.length) {
        (key === 'allOf' ? out.breaking : out.additive).push(`${at}: ${added.length} variant(s) added`);
    }
}

/**
 * Compares two trimmed GraphQL schemas (SDL) with graphql-js's own change detection; its "dangerous"
 * changes (an added enum value or optional argument) count as additive.
 * @returns {{breaking: string[], additive: string[]}}
 */
export function compareGraphQL(currentSdl, nextSdl) {
    const current = buildSchema(currentSdl);
    const next = buildSchema(nextSdl);
    const describe = (c) => `${c.type}: ${c.description}`;
    return {breaking: findBreakingChanges(current, next).map(describe), additive: findDangerousChanges(current, next).map(describe)};
}

function canonical(value) {
    if (!isContainer(value)) {
        return JSON.stringify(value);
    }
    if (Array.isArray(value)) {
        return `[${value.map(canonical).join(',')}]`;
    }
    return `{${Object.keys(value).sort().map((k) => `${JSON.stringify(k)}:${canonical(value[k])}`).join(',')}}`;
}

function isContainer(value) {
    return value !== null && typeof value === 'object';
}
