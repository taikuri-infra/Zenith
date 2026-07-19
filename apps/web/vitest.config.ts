import { defineConfig, configDefaults } from "vitest/config";
import react from "@vitejs/plugin-react";
import path from "path";

export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    setupFiles: ["./src/test-setup.ts"],
    globals: true,
    // e2e/*.spec.ts are Playwright tests (import @playwright/test) and must not
    // be collected by vitest — they run under Playwright, not the unit runner.
    exclude: [...configDefaults.exclude, "e2e/**"],
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
      "@zenith/ui": path.resolve(__dirname, "../../packages/ui/src"),
    },
  },
});
