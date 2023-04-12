import { test } from '@playwright/test';
import core from './github_plugin.spec';
import me from './command/me.spec';
import todo from './command/todo.spec';
import autocomplete from './command/autocomplete.spec';


// Test features when no setup is done
test.describe(autocomplete.noSetup);
test.describe(me.noSetup);
test.describe(todo.noSetup);

// Test /github setup
test.describe(core.setup);

// Test /github connect
test.describe(core.connect);

// Test features that needs connect
test.describe(autocomplete.connected);
test.describe(me.connected);
test.describe(todo.connected);

// Test /github disconnect
test.describe(core.disconnect);

// Test features when setup but no conection
test.describe(me.unconnected);
test.describe(todo.unconnected);
test.describe(autocomplete.unconnected);
