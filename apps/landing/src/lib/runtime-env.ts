/**
 * Runtime environment variable helper for the landing app.
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

export const APP_URL = getEnv(
  "NEXT_PUBLIC_APP_URL",
  "https://app.freezenith.com"
).replace(/\/+$/, "");
