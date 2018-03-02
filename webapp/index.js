// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

import PostTypePullRequest from './components/post_type_pull_request';

class PluginClass {
    initialize(registerComponents, store) {
        registerComponents({}, {custom_github_pull_request: PostTypePullRequest});
    }
}

global.window.plugins['github'] = new PluginClass();
