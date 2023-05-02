// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {test} from '@playwright/test';

import core from './github_plugin.spec';
import me from './command/me.spec';
import todo from './command/todo.spec';
import autocomplete from './command/autocomplete.spec';
import sidebar from './ui/sidebar.spec';
import rhs from './ui/rhs.spec';

import '../support/init_test';

// Test features when no setup is done
test.describe(autocomplete.noSetup);
test.describe(me.noSetup);
test.describe(todo.noSetup);
test.describe(sidebar.noSetup);

// test.describe(rhs.noSetup);

// Test /github setup
test.describe(core.setup);

// Test /github connect
test.describe(core.connect);

// Test features that needs connect
test.describe(autocomplete.connected);
test.describe(me.connected);
test.describe(todo.connected);
test.describe(sidebar.connected);

// test.describe(rhs.connected);

// Test /github disconnect
test.describe(core.disconnect);

// Test features when setup but no conection
test.describe(me.unconnected);
test.describe(todo.unconnected);
test.describe(autocomplete.unconnected);
test.describe(sidebar.unconnected);

// test.describe(rhs.unconnected);