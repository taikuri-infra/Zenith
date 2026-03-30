import { test as setup, expect } from "@playwright/test";
import path from "path";

const authFile = path.join(__dirname, "../playwright/.auth/user.json");

setup("authenticate", async ({ page }) => {
  const email = process.env.SMOKE_TEST_EMAIL;
  const password = process.env.SMOKE_TEST_PASSWORD;

  if (!email || !password) {
    throw new Error(
      "SMOKE_TEST_EMAIL and SMOKE_TEST_PASSWORD env vars are required for auth setup"
    );
  }

  // Navigate to login page
  await page.goto("/login");

  // Wait for the login form to be visible
  await expect(
    page.getByRole("heading", { name: /sign in/i })
  ).toBeVisible();

  // Fill in credentials
  await page.getByPlaceholder("you@example.com").fill(email);
  await page.getByPlaceholder("••••••••").fill(password);

  // Click the sign in button
  await page.getByRole("button", { name: /sign in/i }).click();

  // Wait for redirect to dashboard — the overview page should load
  await page.waitForURL("/", { timeout: 15_000 });

  // Verify we landed on the dashboard (Shell renders the sidebar)
  await expect(page.getByText("Overview")).toBeVisible({ timeout: 10_000 });

  // Save signed-in state for reuse across tests
  await page.context().storageState({ path: authFile });
});
