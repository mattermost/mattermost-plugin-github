// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Sorts the differences between the committed trimmed specs and GitHub's current ones into breaking
// and additive changes, so the nightly check only fails when fake-github or the plugin could stop
// matching GitHub. GitHub updates its REST description several times a week, mostly by adding
// optional fields; those are reported, not failed on.

import {buildSchema, findBreakingChanges, findDangerousChanges} from 'graphql';

// An added key that narrows what is valid, rather than describing something new.
const RESTRICTIONS = new Set(['enum', 'required', 'nullable', 'pattern', 'format', 'minLength', 'maxLength', 'minimum', 'maximum', 'minItems', 'maxItems', 'additionalProperties', 'requestBody', 'items']);

/**
 * Compares two trimmed REST descriptions as JSON trees: anything removed or changed is breaking,
 * anything added is additive unless it is a restriction. In `required`, any change is breaking (a
 * request may now lack a field, or a response may now omit one); in `enum`, only removals are.
 * @returns {{breaking: string[], additive: string[]}}
 */
export function compareRest(current, next) {
    const out = {breaking: [], additive: []};
    diff(current, next, '', out);
    return out;
}

function diff(a, b, at, out) {
    const key = at.slice(at.lastIndexOf('.') + 1);
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
    if (!isContainer(a) || !isContainer(b) || Array.isArray(a) !== Array.isArray(b)) {
        if (JSON.stringify(a) !== JSON.stringify(b)) {
            out.breaking.push(`${at}: ${JSON.stringify(a)} -> ${JSON.stringify(b)}`);
        }
        return;
    }
    for (const k of Object.keys(a)) {
        if (k in b) {
            diff(a[k], b[k], `${at}.${k}`, out);
        } else {
            out.breaking.push(`${at}.${k} removed`);
        }
    }
    for (const k of Object.keys(b)) {
        if (!(k in a)) {
            (RESTRICTIONS.has(k) ? out.breaking : out.additive).push(`${at}.${k} added`);
        }
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

function isContainer(value) {
    return value !== null && typeof value === 'object';
}
