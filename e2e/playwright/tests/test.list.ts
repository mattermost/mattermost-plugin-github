import { test } from '@playwright/test';
import core from './github_plugin.spec';
import me from './command/me.spec';
import todo from './command/todo.spec';
import autocomplete from './command/autocomplete.spec';


// Test features when no setup is done
test.describe(me.noSetup);
test.describe(todo.noSetup);
test.describe(autocomplete.noSetup);

// Test Setup & connect
test.describe(core.setup);

// Test Setuip & connect
test.describe(core.connect);

// Test features that needs connect
test.describe(me.connected);
test.describe(todo.connected);
test.describe(autocomplete.connected);

// should disconnect (keeping setup) and keep testing disconnected features
// Test features when setup but no conection
// test.describe(me.unconnected);
