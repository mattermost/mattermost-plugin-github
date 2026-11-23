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
 * are. allOf/anyOf/oneOf variants are compared regardless of order (see diffVariants).
 *
 * A `$ref` that points somewhere new is compared by the schemas it resolves to, so a component GitHub
 * renames without changing its shape is no change. Components are compared by name only where both
 * sides have them; one that appears or disappears is judged where it is referenced.
 * @returns {{breaking: string[], additive: string[]}}
 */
export function compareRest(current, next) {
    const ctx = {out: {breaking: [], additive: []}, current, next, followed: new Set()};
    for (const top of ['paths', 'x-webhooks']) {
        diff(current[top], next[top], `.${top}`, top, ctx, true);
    }
    for (const [kind, named] of Object.entries(current.components ?? {})) {
        for (const name of Object.keys(named)) {
            if (next.components?.[kind]?.[name]) {
                diff(named[name], next.components[kind][name], `.components.${kind}.${name}`, '', ctx);
            }
        }
    }
    return ctx.out;
}

// `key` is the keyword `a` and `b` sit under; `inNames` says their own keys are names, not keywords.
function diff(a, b, at, key, ctx, inNames = false) {
    const {out} = ctx;
    if (isContainer(a) && isContainer(b) && typeof a.$ref === 'string' && typeof b.$ref === 'string' && a.$ref !== b.$ref) {
        // Follow each pair of refs once: that also stops recursive schemas from looping.
        const pair = `${a.$ref} ${b.$ref}`;
        if (!ctx.followed.has(pair)) {
            ctx.followed.add(pair);
            diff(resolve(ctx.current, a.$ref), resolve(ctx.next, b.$ref), `${at} (${a.$ref} -> ${b.$ref})`, key, ctx);
        }
        return;
    }
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
        diffVariants(a, b, at, key, ctx);
        return;
    }
    if (!isContainer(a) || !isContainer(b) || Array.isArray(a) !== Array.isArray(b)) {
        if (JSON.stringify(a) !== JSON.stringify(b)) {
            out.breaking.push(`${at}: ${JSON.stringify(a)} -> ${JSON.stringify(b)}`);
        }
        return;
    }
    for (const k of Object.keys(a)) {
        if (k in b && k === 'not' && !inNames) {
            // `not` flips what an edit means (an enum value added inside it rejects that value), so
            // any change within it is breaking.
            if (canonical(a.not, ctx.current) !== canonical(b.not, ctx.next)) {
                out.breaking.push(`${at}.not changed`);
            }
        } else if (k in b) {
            diff(a[k], b[k], `${at}.${k}`, k, ctx, !inNames && NAME_MAPS.has(k));
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

// Unordered: variants with an exact match on the other side are set aside first, so a reordered list
// is no change. When the same number of variants remain unmatched, they were edited in place and are
// compared pairwise. Otherwise a removed variant is breaking, and so is an added one, except for anyOf:
// an added allOf variant is one more constraint. For oneOf, any change beyond reordering is breaking:
// an added or edited variant can overlap another, and values matching both become invalid.
function diffVariants(a, b, at, key, ctx) {
    const {out} = ctx;
    const unmatchedA = [...a];
    const unmatchedB = [];
    for (const variant of b) {
        const i = unmatchedA.findIndex((v) => canonical(v, ctx.current) === canonical(variant, ctx.next));
        if (i >= 0) {
            unmatchedA.splice(i, 1);
        } else {
            unmatchedB.push(variant);
        }
    }
    // Whether an edited oneOf variant still matches exclusively can't be decided by diffing, so any
    // change to oneOf beyond reordering is breaking.
    if (key === 'oneOf' && (unmatchedA.length || unmatchedB.length)) {
        out.breaking.push(`${at}: ${unmatchedA.length} variant(s) changed or removed, ${unmatchedB.length} changed or added`);
        return;
    }
    if (unmatchedA.length && unmatchedA.length === unmatchedB.length) {
        unmatchedA.forEach((variant, i) => diff(variant, unmatchedB[i], `${at}[${i}]`, '', ctx));
        return;
    }
    if (unmatchedA.length) {
        out.breaking.push(`${at}: ${unmatchedA.length} variant(s) removed`);
    }
    if (unmatchedB.length) {
        (key === 'anyOf' ? out.additive : out.breaking).push(`${at}: ${unmatchedB.length} variant(s) added`);
    }
}

/**
 * Compares two trimmed GraphQL schemas (SDL) with graphql-js's own change detection; its "dangerous"
 * changes (an added enum value or optional argument) count as additive. graphql-js doesn't report a
 * field, argument or enum value becoming deprecated, so those are listed as additive here: they warn
 * of a removal to come.
 * @returns {{breaking: string[], additive: string[]}}
 */
export function compareGraphQL(currentSdl, nextSdl) {
    const current = buildSchema(currentSdl);
    const next = buildSchema(nextSdl);
    const describe = (c) => `${c.type}: ${c.description}`;
    return {
        breaking: findBreakingChanges(current, next).map(describe),
        additive: [...findDangerousChanges(current, next).map(describe), ...newDeprecations(current, next)],
    };
}

function newDeprecations(current, next) {
    const found = [];
    const check = (name, was, now) => {
        if (was && now && !was.deprecationReason && now.deprecationReason) {
            found.push(`${name} deprecated: ${now.deprecationReason}`);
        }
    };
    for (const [typeName, type] of Object.entries(current.getTypeMap())) {
        const nextType = next.getType(typeName);
        if (typeName.startsWith('__') || !nextType) {
            continue;
        }
        if ('getValues' in type && 'getValues' in nextType) {
            for (const value of type.getValues()) {
                check(`${typeName}.${value.name}`, value, nextType.getValue(value.name));
            }
        }
        if ('getFields' in type && 'getFields' in nextType) {
            const nextFields = nextType.getFields();
            for (const [fieldName, field] of Object.entries(type.getFields())) {
                const nextField = nextFields[fieldName];
                check(`${typeName}.${fieldName}`, field, nextField);
                for (const arg of field.args ?? []) {
                    check(`${typeName}.${fieldName}(${arg.name})`, arg, nextField?.args?.find((x) => x.name === arg.name));
                }
            }
        }
    }
    return found;
}

// A stable string for a schema in `spec`: object keys sorted, `required`/`enum` lists sorted (their
// order carries no meaning), and `$ref`s replaced by what they point to, so a renamed component still
// matches. A ref already being expanded is left as is, to stop recursive schemas from looping.
function canonical(value, spec, key = '', expanding = new Set()) {
    if (!isContainer(value)) {
        return JSON.stringify(value);
    }
    if (typeof value.$ref === 'string' && !expanding.has(value.$ref)) {
        return canonical(resolve(spec, value.$ref), spec, key, new Set([...expanding, value.$ref]));
    }
    if (Array.isArray(value)) {
        const items = value.map((v) => canonical(v, spec, '', expanding));
        return `[${(key === 'required' || key === 'enum' ? items.sort() : items).join(',')}]`;
    }
    return `{${Object.keys(value).sort().map((k) => `${JSON.stringify(k)}:${canonical(value[k], spec, k, expanding)}`).join(',')}}`;
}

// '#/components/schemas/issue' -> spec.components.schemas.issue (undefined if it isn't there).
function resolve(spec, ref) {
    return ref.replace(/^#\//, '').split('/').reduce((node, part) => node?.[part.replaceAll('~1', '/').replaceAll('~0', '~')], spec);
}

function isContainer(value) {
    return value !== null && typeof value === 'object';
}
