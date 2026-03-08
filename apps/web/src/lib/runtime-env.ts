/**
 * Runtime environment variable helper.
 *
 * In Docker deployments, NEXT_PUBLIC_* env vars are baked at build time
 * and cannot be changed at runtime. This module reads from window.__ENV
 * (injected by entrypoint.sh) on the client, falling back to process.env
 * for SSR and development.
 */

declare global {
  interface Window {
    __ENV?: Record<string, string>;
  }
}

function getEnv(key: string, fallback: string = ""): string {
  // Client-side: read from runtime injection
  if (typeof window !== "undefined" && window.__ENV?.[key]) {
    return window.__ENV[key];
  }
  // Server-side / development: read from process.env
  return process.env[key] || fallback;
}

export const API_BASE_URL = getEnv(
  "NEXT_PUBLIC_API_URL",
  "http://localhost:8080"
);
export const DEMO_MODE = getEnv("NEXT_PUBLIC_DEMO_MODE", "false") === "true";
export const ZENITH_MODE = getEnv("NEXT_PUBLIC_ZENITH_MODE", "standalone");
export const LANDING_URL = getEnv(
  "NEXT_PUBLIC_LANDING_URL",
  "https://freezenith.com"
);
export const IS_STANDALONE = ZENITH_MODE !== "saas";
