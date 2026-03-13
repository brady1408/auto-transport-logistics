import { test, expect } from "@playwright/test";

// ── Public endpoints ───────────────────────────────────────────────────────

test("health check", async ({ request }) => {
  const res = await request.get("/health");
  expect(res.status()).toBe(200);
  expect(await res.text()).toContain("ok");
});

test("login page renders", async ({ page }) => {
  // Log out first so we get the real login page
  await page.goto("/login");
  await expect(page.locator('input[name="username"]')).toBeVisible();
  await expect(page.locator('input[name="password"]')).toBeVisible();
  await expect(page.locator('button[type="submit"]')).toBeVisible();
});

test("bad login shows error", async ({ page }) => {
  await page.goto("/login");
  await page.fill('input[name="username"]', "nobody");
  await page.fill('input[name="password"]', "wrongpassword");
  await page.click('button[type="submit"]');
  await expect(page.locator("body")).toContainText(/invalid|incorrect|not found/i);
});

// ── Authenticated pages (use stored admin session) ─────────────────────────

test("dashboard loads", async ({ page }) => {
  await page.goto("/");
  await expect(page).not.toHaveURL(/login/);
  await expect(page.locator("nav")).toBeVisible();
});

test("customers list loads", async ({ page }) => {
  await page.goto("/customers");
  await expect(page).not.toHaveURL(/login/);
  await expect(page.locator("h2, h1")).toContainText(/customer/i);
});

test("orders list loads", async ({ page }) => {
  await page.goto("/orders");
  await expect(page).not.toHaveURL(/login/);
  await expect(page.locator("h2, h1")).toContainText(/order/i);
});

test("trips list loads", async ({ page }) => {
  await page.goto("/trips");
  await expect(page).not.toHaveURL(/login/);
  await expect(page.locator("h2, h1")).toContainText(/trip/i);
});

test("invoices list loads", async ({ page }) => {
  await page.goto("/invoices");
  await expect(page).not.toHaveURL(/login/);
  await expect(page.locator("h2, h1")).toContainText(/invoice/i);
});

test("feedback page loads", async ({ page }) => {
  await page.goto("/feedback");
  await expect(page).not.toHaveURL(/login/);
  await expect(page.locator("h2, h1")).toContainText(/feedback/i);
});

test("admin companies page loads", async ({ page }) => {
  await page.goto("/admin/companies");
  await expect(page).not.toHaveURL(/login/);
  await expect(page.locator("h2, h1")).toContainText(/compan/i);
});

test("admin api-keys page loads", async ({ page }) => {
  await page.goto("/admin/api-keys");
  await expect(page).not.toHaveURL(/login/);
  await expect(page.locator("h2, h1")).toContainText(/api key/i);
});

test("protected routes redirect unauthenticated users to login", async ({
  browser,
}) => {
  // Fresh context with no session
  const ctx = await browser.newContext();
  const page = await ctx.newPage();
  await page.goto("/customers");
  await expect(page).toHaveURL(/login/);
  await ctx.close();
});

// ── API endpoint smoke test ────────────────────────────────────────────────

test("API key auth: valid key returns 200", async ({ request }) => {
  const key = process.env.ATLINKS_API_KEY;
  if (!key) {
    test.skip();
    return;
  }
  const res = await request.get("/api/feedback?status=all", {
    headers: { "X-API-Key": key },
  });
  expect(res.status()).toBe(200);
});

test("API key auth: no key returns 401", async ({ request }) => {
  const res = await request.get("/api/feedback");
  expect(res.status()).toBe(401);
});
