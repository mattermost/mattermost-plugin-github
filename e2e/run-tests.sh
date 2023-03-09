sed -i "s/testDir.*/testDir: '\.\.\/\.\.\/\.\.\/mattermost-plugin-github\/e2e',/g" ../../mattermost-webapp/e2e/playwright/playwright.config.ts
npm run test-slomo --prefix ../../mattermost-webapp/e2e/playwright
