import { test, expect } from "@playwright/test";

/**
 * Project CRUD E2E tests.
 *
 * These tests create a project, verify it, then clean up by deleting it.
 * They run sequentially within the describe block (test.describe.serial).
 */

const PROJECT_NAME = `e2e-project-${Date.now()}`;

test.describe.serial("Project CRUD", () => {
  test("can create a new project", async ({ page }) => {
    await page.goto("/projects/new");

    // Step 1 of the wizard: project name input
    // The new project wizard starts with a name/slug input
    const nameInput = page.getByPlaceholder(/project name/i).or(
      page.getByLabel(/name/i).first()
    );
    await expect(nameInput).toBeVisible({ timeout: 15_000 });
    await nameInput.fill(PROJECT_NAME);

    // Look for a "Next" or "Continue" or "Create" button to advance the wizard
    const advanceButton = page
      .getByRole("button", { name: /next|continue|create/i })
      .first();
    await advanceButton.click();

    // The wizard should advance — wait for step 2 or a success indicator
    // Step 2 typically shows compose/image configuration
    await page.waitForTimeout(2000);

    // If we reached step 2+, the project was created (draft status)
    // Verify we are still on the wizard or redirected
    await expect(page).not.toHaveURL("/projects/new", { timeout: 10_000 });
  });

  test("project appears in project list", async ({ page }) => {
    // Go to dashboard which shows projects
    await page.goto("/");
    await page.waitForLoadState("networkidle");

    // The project name or slug should appear somewhere on the dashboard
    // Projects are shown in a dropdown/selector in the sidebar
    const projectText = page.getByText(PROJECT_NAME).or(
      page.getByText(PROJECT_NAME.toLowerCase())
    );
    await expect(projectText.first()).toBeVisible({ timeout: 15_000 });
  });

  test("can open project detail page", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");

    // Click on the project name to navigate to its detail page
    const projectLink = page
      .getByText(PROJECT_NAME)
      .or(page.getByText(PROJECT_NAME.toLowerCase()))
      .first();
    await projectLink.click();

    // Should see project overview content
    await expect(
      page.getByText(/project overview/i).or(page.getByText(PROJECT_NAME))
    ).toBeVisible({ timeout: 10_000 });
  });

  test("can delete the project", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");

    // Select the project first if needed
    const projectLink = page
      .getByText(PROJECT_NAME)
      .or(page.getByText(PROJECT_NAME.toLowerCase()))
      .first();
    await projectLink.click();
    await page.waitForLoadState("networkidle");

    // Click the "Delete Project" button
    const deleteButton = page.getByRole("button", {
      name: /delete project/i,
    });
    await expect(deleteButton).toBeVisible({ timeout: 10_000 });
    await deleteButton.click();

    // Confirmation modal appears — click the final "Delete" button
    const confirmDelete = page
      .getByRole("button", { name: /^delete$/i })
      .or(page.getByRole("button", { name: /^delete$/i }));
    await expect(confirmDelete).toBeVisible({ timeout: 5_000 });
    await confirmDelete.click();

    // Wait for redirect back to dashboard
    await page.waitForURL("/", { timeout: 15_000 });
  });

  test("project no longer appears in list", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");

    // The deleted project name should not appear anymore
    const projectText = page
      .getByText(PROJECT_NAME)
      .or(page.getByText(PROJECT_NAME.toLowerCase()));
    await expect(projectText).toHaveCount(0, { timeout: 10_000 });
  });
});
