import { test, expect } from "@playwright/test";

/**
 * App lifecycle E2E tests.
 *
 * Creates an app using a public image (nginx:latest), verifies it appears
 * in the list, checks its detail page, then deletes it.
 *
 * These tests run sequentially since each depends on the previous.
 */

const APP_NAME = `e2e-app-${Date.now()}`;
const APP_IMAGE = "nginx:latest";

test.describe.serial("App Lifecycle", () => {
  test("can create a new app", async ({ page }) => {
    await page.goto("/apps");

    // Click "Deploy App" button to open the deploy wizard
    const deployButton = page.getByRole("button", { name: /deploy app/i });
    await expect(deployButton).toBeVisible({ timeout: 15_000 });
    await deployButton.click();

    // The deploy wizard modal should open
    // Fill in the app name
    const nameInput = page.getByPlaceholder(/app name/i).or(
      page.getByLabel(/name/i).first()
    );
    await expect(nameInput).toBeVisible({ timeout: 10_000 });
    await nameInput.fill(APP_NAME);

    // Fill in the image reference
    const imageInput = page.getByPlaceholder(/image/i).or(
      page.getByLabel(/image/i).first()
    );
    await expect(imageInput).toBeVisible({ timeout: 5_000 });
    await imageInput.fill(APP_IMAGE);

    // The wizard may auto-detect nginx and set port to 80
    // Wait a moment for auto-detection
    await page.waitForTimeout(1000);

    // Click the deploy/create button
    const createButton = page
      .getByRole("button", { name: /deploy|create|launch/i })
      .last();
    await createButton.click();

    // Wait for success — the wizard should close or redirect
    // Look for a success toast or the app appearing in the list
    await page.waitForTimeout(3000);
  });

  test("app appears in app list", async ({ page }) => {
    await page.goto("/apps");
    await page.waitForLoadState("networkidle");

    // The app name should appear in the deployed apps section
    const appText = page.getByText(APP_NAME);
    await expect(appText.first()).toBeVisible({ timeout: 15_000 });
  });

  test("app detail page shows status", async ({ page }) => {
    await page.goto("/apps");
    await page.waitForLoadState("networkidle");

    // Click on the app to go to its detail page
    const appLink = page.getByText(APP_NAME).first();
    await appLink.click();

    // The detail page should show the app name and status info
    await expect(page.getByText(APP_NAME)).toBeVisible({ timeout: 10_000 });

    // Status should be visible (running, building, deploying, etc.)
    const statusText = page
      .getByText(/running|building|deploying|failed|sleeping|stopped/i)
      .first();
    await expect(statusText).toBeVisible({ timeout: 30_000 });
  });

  test("can delete the app", async ({ page }) => {
    await page.goto("/apps");
    await page.waitForLoadState("networkidle");

    // Find the app card and click its delete button (trash icon)
    const appCard = page.getByText(APP_NAME).first();
    await expect(appCard).toBeVisible({ timeout: 10_000 });

    // The delete button is a trash icon button within or near the app card
    // Navigate to the card's parent and find the delete button
    const deleteButton = page
      .locator(`[title="Delete app"]`)
      .or(page.getByRole("button", { name: /delete/i }))
      .first();
    await deleteButton.click();

    // Confirmation modal: type the app name to confirm
    const confirmInput = page.getByPlaceholder(APP_NAME).or(
      page.locator(`input[placeholder="${APP_NAME}"]`)
    );
    await expect(confirmInput).toBeVisible({ timeout: 5_000 });
    await confirmInput.fill(APP_NAME);

    // Click the final "Delete App" button
    const confirmDelete = page.getByRole("button", { name: /delete app/i });
    await confirmDelete.click();

    // Wait for the page to reload or the app to disappear
    await page.waitForTimeout(3000);
  });

  test("app no longer appears in list", async ({ page }) => {
    await page.goto("/apps");
    await page.waitForLoadState("networkidle");

    // Give the list time to load
    await page.waitForTimeout(2000);

    // The deleted app should not appear
    const appText = page.getByText(APP_NAME);
    await expect(appText).toHaveCount(0, { timeout: 10_000 });
  });
});
