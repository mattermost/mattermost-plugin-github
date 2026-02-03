// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

module.exports = (api) => {
    const isTest = api.env('test');

    return {
        presets: [
            ['@babel/preset-env', {
                targets: {
                    chrome: 66,
                    firefox: 60,
                    edge: 42,
                    safari: 12,
                },
                modules: isTest ? 'auto' : false,
                corejs: 3,
                debug: false,
                useBuiltIns: 'usage',
                shippedProposals: true,
            }],
            ['@babel/preset-react', {
                runtime: 'automatic',
            }],
            ['@babel/preset-typescript', {
                allExtensions: true,
                isTSX: true,
            }],
            ['@emotion/babel-preset-css-prop'],
        ],
        plugins: [
            '@babel/plugin-transform-class-properties',
            '@babel/plugin-syntax-dynamic-import',
            '@babel/plugin-transform-object-rest-spread',
            '@babel/plugin-transform-optional-chaining',
            'babel-plugin-typescript-to-proptypes',
        ],
    };
};
