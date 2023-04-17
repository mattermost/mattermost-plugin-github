import { expect, test as setup } from "@e2e-support/test_fixture";

const authFile = __dirname + '/../.auth-user.json';

setup('authenticate', async ({ page, pages, pw }) => {
    const {adminClient, adminUser} = await pw.getAdminClient();
    const config = await adminClient.getConfig();
    const login = new pages.LoginPage(page, config)
    await login.goto();
    await login.login(adminUser!);
    await page.context().storageState({ path: authFile });
});
