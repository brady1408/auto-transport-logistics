import { test as setup, expect } from "@playwright/test";
import path from "path";

const authFile = path.join(__dirname, ".auth/admin.json");

setup("authenticate as admin", async ({ page }) => {
  await page.goto("/login");
  await page.fill('input[name="username"]', process.env.TEST_USER || "admin");
  await page.fill('input[name="password"]', process.env.TEST_PASS || "admin");
  await page.click('button[type="submit"]');

  // Should land on dashboard after login
  await expect(page).toHaveURL(/\/(dashboard)?$/);

  // Save session cookies so other tests start authenticated
  await page.context().storageState({ path: authFile });
});
