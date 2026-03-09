/**
 * Runtime environment variable helper for mission-control.
 *
 * Next.js inlines process.env.NEXT_PUBLIC_* (dot access) at build time.
 * window.__ENV is set by entrypoint.sh at container start for runtime override.
 * Getter functions ensure window.__ENV is checked lazily (after env.js loads).
 */

declare global {
  interface Window {
    __ENV?: Record<string, string>;
  }
}

function getRuntimeEnv(key: string): string | undefined {
  if (typeof window !== "undefined" && window.__ENV?.[key]) {
    return window.__ENV[key];
  }
  return undefined;
}

// Use direct process.env.NEXT_PUBLIC_* so Next.js inlines at build time.
// Getter functions allow window.__ENV to override at runtime.
export function getApiBaseUrl(): string {
  return (
    getRuntimeEnv("NEXT_PUBLIC_API_URL") ||
    process.env.NEXT_PUBLIC_API_URL ||
    "http://localhost:8080"
  );
}

export function isDemoMode(): boolean {
  return (
    (getRuntimeEnv("NEXT_PUBLIC_DEMO_MODE") ||
      process.env.NEXT_PUBLIC_DEMO_MODE ||
      "false") === "true"
  );
}

export function getLandingUrl(): string {
  return (
    getRuntimeEnv("NEXT_PUBLIC_LANDING_URL") ||
    process.env.NEXT_PUBLIC_LANDING_URL ||
    "https://freezenith.com"
  );
}

// Backwards-compatible exports (lazy — evaluated on each access)
export const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
export const DEMO_MODE = process.env.NEXT_PUBLIC_DEMO_MODE === "true";
export const LANDING_URL = process.env.NEXT_PUBLIC_LANDING_URL || "https://freezenith.com";
