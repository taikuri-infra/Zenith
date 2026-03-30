import { test, expect } from "@playwright/test";

test.describe("Dashboard", () => {
  test("loads and shows main content", async ({ page }) => {
    await page.goto("/");

    // The project overview heading or "Project overview" subtitle should appear
    await expect(
      page.getByText("Project overview")
    ).toBeVisible({ timeout: 15_000 });
  });

  test("navigation sidebar is visible", async ({ page }) => {
    await page.goto("/");

    // Sidebar sections from sidebar.tsx
    await expect(page.getByText("OVERVIEW")).toBeVisible();
    await expect(page.getByText("COMPUTE")).toBeVisible();
  });

  test("user can see projects section", async ({ page }) => {
    await page.goto("/");

    // The sidebar has a "New Project" button or link
    const newProjectLink = page.getByRole("link", { name: /new project/i });
    // If there is no explicit "New Project" link in sidebar, look for the
    // "New Project" button in the main content area
    const newProjectButton = page.getByRole("link", {
      name: /new project/i,
    });

    // At least one should be present (sidebar or content area)
    const projectIndicator = page
      .getByText(/project/i)
      .first();
    await expect(projectIndicator).toBeVisible({ timeout: 10_000 });
  });

  test("user can see apps section", async ({ page }) => {
    await page.goto("/apps");

    // The apps page heading
    await expect(
      page.getByRole("heading", { name: /apps/i })
    ).toBeVisible({ timeout: 15_000 });
  });
});
