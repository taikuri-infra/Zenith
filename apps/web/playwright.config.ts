import { defineConfig, devices } from "@playwright/test";

/**
 * Playwright E2E configuration for Zenith Web Dashboard.
 *
 * Run against staging by default:
 *   npx playwright test
 *
 * Override the base URL for local dev:
 *   PLAYWRIGHT_BASE_URL=http://localhost:3000 npx playwright test
 */
export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: process.env.CI ? "github" : "html",

  timeout: 30_000,
  expect: {
    timeout: 10_000,
  },

  use: {
    baseURL:
      process.env.PLAYWRIGHT_BASE_URL || "https://app.stage.freezenith.com",
    trace: "on-first-retry",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
  },

  projects: [
    // ── Auth setup (runs first, saves storage state) ──
    {
      name: "setup",
      testMatch: /auth\.setup\.ts/,
    },

    // ── Main test suite (chromium only for CI speed) ──
    {
      name: "chromium",
      use: {
        ...devices["Desktop Chrome"],
        storageState: "playwright/.auth/user.json",
      },
      dependencies: ["setup"],
    },
  ],
});
