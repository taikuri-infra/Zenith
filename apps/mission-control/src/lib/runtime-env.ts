/**
 * Runtime environment variable helper for mission-control.
 */

declare global {
  interface Window {
    __ENV?: Record<string, string>;
  }
}

function getEnv(key: string, fallback: string = ""): string {
  if (typeof window !== "undefined" && window.__ENV?.[key]) {
    return window.__ENV[key];
  }
  return process.env[key] || fallback;
}

export const API_BASE_URL = getEnv(
  "NEXT_PUBLIC_API_URL",
  "http://localhost:8080"
);
export const DEMO_MODE = getEnv("NEXT_PUBLIC_DEMO_MODE", "false") === "true";
export const LANDING_URL = getEnv(
  "NEXT_PUBLIC_LANDING_URL",
  "https://freezenith.com"
);
