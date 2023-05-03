// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {test} from '@playwright/test';

import core from './github_plugin.spec';
import me from './command/me.spec';
import todo from './command/todo.spec';
import autocomplete from './command/autocomplete.spec';
import issueCreate from './command/issue_create.spec';

import '../support/init_test';

// Test features when no setup is done
test.describe(autocomplete.noSetup);
test.describe(me.noSetup);
test.describe(todo.noSetup);
test.describe(issueCreate.noSetup);

// Test /github setup
test.describe(core.setup);

// Test /github connect
test.describe(core.connect);

// Test features that needs connect
test.describe(autocomplete.connected);
test.describe(me.connected);
test.describe(todo.connected);
test.describe(issueCreate.connected);

// Test /github disconnect
test.describe(core.disconnect);

// Test features when setup but no conection
test.describe(me.unconnected);
test.describe(todo.unconnected);
test.describe(autocomplete.unconnected);
test.describe(issueCreate.unconnected);
