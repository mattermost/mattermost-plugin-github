// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Self-tests for drift.mjs: the nightly check must fail on breaking changes and only report additive ones.

import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import {fileURLToPath} from 'node:url';

import {compareGraphQL, compareRest} from './drift.mjs';

const specsDir = path.join(path.dirname(fileURLToPath(import.meta.url)), 'specs');
const rest = JSON.parse(fs.readFileSync(path.join(specsDir, 'rest.json'), 'utf8'));
const graphql = fs.readFileSync(path.join(specsDir, 'schema.graphql'), 'utf8');

const issue = (s) => s.components.schemas.issue;
const createIssue = (s) => s.paths['/repos/{owner}/{repo}/issues'].post;
const userResponse = (s) => s.paths['/user'].get.responses['200'].content['application/json'].schema;

// [description, edit to a copy of the committed spec, expected result]
const restCases = [
    ['nothing changed', () => {}, 'none'],
    ['an optional field added', (s) => { issue(s).properties.zzz_added = {type: 'object', nullable: true}; }, 'additive'],
    ['a field removed', (s) => { delete issue(s).properties.title; }, 'breaking'],
    ['a field type changed', (s) => { issue(s).properties.number = {type: 'string'}; }, 'breaking'],
    ['a field made nullable', (s) => { issue(s).properties.title.nullable = true; }, 'breaking'],
    ['a field no longer required', (s) => { issue(s).required = issue(s).required.slice(1); }, 'breaking'],
    ['a request field newly required', (s) => { createIssue(s).requestBody.content['application/json'].schema.required.push('body'); }, 'breaking'],
    ['an enum value added', (s) => { issue(s).properties.state.enum = ['open', 'closed', 'archived']; }, 'additive'],
    ['an operation removed', (s) => { delete s.paths['/repos/{owner}/{repo}/issues'].post; }, 'breaking'],
    ['a webhook removed', (s) => { delete s['x-webhooks'][Object.keys(s['x-webhooks'])[0]]; }, 'breaking'],
    ['an optional field named like a keyword added', (s) => { issue(s).properties.items = {type: 'array', items: {type: 'string'}}; }, 'additive'],
    ['a constraint added through allOf', (s) => { issue(s).properties.title.allOf = [{enum: ['open']}]; }, 'breaking'],
    ['a not constraint added', (s) => { issue(s).properties.title.not = {enum: ['']}; }, 'breaking'],
    ['oneOf variants reordered', (s) => { userResponse(s).oneOf.reverse(); }, 'none'],
    ['a oneOf variant added', (s) => { userResponse(s).oneOf.push({type: 'string'}); }, 'additive'],
    ['a oneOf variant removed', (s) => { userResponse(s).oneOf.pop(); }, 'breaking'],
];

for (const [name, edit, expected] of restCases) {
    test(`REST: ${name} is ${expected}`, () => {
        // Give `state` an enum on both sides, so the enum case compares two lists.
        const base = structuredClone(rest);
        issue(base).properties.state.enum = ['open', 'closed'];
        const next = structuredClone(base);
        edit(next);
        const {breaking, additive} = compareRest(base, next);
        assert.equal(breaking.length > 0, expected === 'breaking', `breaking: ${breaking.join('; ')}`);
        assert.equal(additive.length > 0, expected === 'additive', `additive: ${additive.join('; ')}`);
    });
}

test('REST: a removed enum value is breaking', () => {
    const base = structuredClone(rest);
    issue(base).properties.state.enum = ['open', 'closed'];
    const next = structuredClone(base);
    issue(next).properties.state.enum = ['open'];
    assert.match(compareRest(base, next).breaking.join('\n'), /state\.enum removed: closed/);
});

test('GraphQL: a removed field is breaking and an added enum value is additive', () => {
    assert.match(compareGraphQL(graphql, graphql.replace(/\n {2}changedFiles: Int!/, '')).breaking.join('\n'), /changedFiles was removed/);
    const [firstEnum] = /enum \w+ \{\n {2}\w+/.exec(graphql);
    const added = compareGraphQL(graphql, graphql.replace(firstEnum, `${firstEnum}\n  ZZZ_ADDED`));
    assert.deepEqual(added.breaking, []);
    assert.match(added.additive.join('\n'), /ZZZ_ADDED was added/);
});
