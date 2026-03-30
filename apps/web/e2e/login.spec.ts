import { test, expect } from "@playwright/test";

/**
 * Login flow E2E tests.
 *
 * These tests do NOT use saved auth state — they exercise the login page
 * directly, including error handling and logout.
 */

// Override the storage state so these tests start unauthenticated
test.use({ storageState: { cookies: [], origins: [] } });

test.describe("Login", () => {
  test("login page loads", async ({ page }) => {
    await page.goto("/login");

    // The Zenith branding and sign-in heading should be visible
    await expect(page.getByText("Zenith")).toBeVisible({ timeout: 10_000 });
    await expect(
      page.getByRole("heading", { name: /sign in to your account/i })
    ).toBeVisible();
  });

  test("shows email and password fields", async ({ page }) => {
    await page.goto("/login");

    // Email field
    const emailInput = page.getByPlaceholder("you@example.com");
    await expect(emailInput).toBeVisible({ timeout: 10_000 });
    await expect(emailInput).toHaveAttribute("type", "email");

    // Password field
    const passwordInput = page.getByPlaceholder("••••••••");
    await expect(passwordInput).toBeVisible();
    await expect(passwordInput).toHaveAttribute("type", "password");
  });

  test("invalid credentials show error", async ({ page }) => {
    await page.goto("/login");

    // Fill in bad credentials
    await page.getByPlaceholder("you@example.com").fill("bad@example.com");
    await page.getByPlaceholder("••••••••").fill("wrongpassword123");

    // Submit
    await page.getByRole("button", { name: /sign in/i }).click();

    // Error message should appear
    await expect(
      page.getByText(/invalid email or password|error|failed/i)
    ).toBeVisible({ timeout: 10_000 });

    // Should still be on login page
    await expect(page).toHaveURL(/\/login/);
  });

  test("valid credentials redirect to dashboard", async ({ page }) => {
    const email = process.env.SMOKE_TEST_EMAIL;
    const password = process.env.SMOKE_TEST_PASSWORD;

    if (!email || !password) {
      test.skip(true, "SMOKE_TEST_EMAIL/PASSWORD not set");
      return;
    }

    await page.goto("/login");

    // Fill in valid credentials
    await page.getByPlaceholder("you@example.com").fill(email);
    await page.getByPlaceholder("••••••••").fill(password);

    // Submit
    await page.getByRole("button", { name: /sign in/i }).click();

    // Should redirect to dashboard
    await page.waitForURL("/", { timeout: 15_000 });
    await expect(page.getByText("Overview")).toBeVisible({ timeout: 10_000 });
  });

  test("logout works", async ({ page }) => {
    const email = process.env.SMOKE_TEST_EMAIL;
    const password = process.env.SMOKE_TEST_PASSWORD;

    if (!email || !password) {
      test.skip(true, "SMOKE_TEST_EMAIL/PASSWORD not set");
      return;
    }

    // First, log in
    await page.goto("/login");
    await page.getByPlaceholder("you@example.com").fill(email);
    await page.getByPlaceholder("••••••••").fill(password);
    await page.getByRole("button", { name: /sign in/i }).click();
    await page.waitForURL("/", { timeout: 15_000 });

    // Open user menu — the header has a user avatar/button that toggles a dropdown
    // Look for the user's email or avatar button in the header
    const userMenuTrigger = page
      .getByRole("button", { name: new RegExp(email.split("@")[0], "i") })
      .or(page.locator("header button").last());
    await userMenuTrigger.click();

    // Click "Sign Out" in the dropdown
    const signOutButton = page.getByRole("button", { name: /sign out/i });
    await expect(signOutButton).toBeVisible({ timeout: 5_000 });
    await signOutButton.click();

    // Should be redirected to login page
    await expect(page).toHaveURL(/\/login/, { timeout: 10_000 });
  });
});
