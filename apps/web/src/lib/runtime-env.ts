/**
 * Runtime environment variable helper.
 *
 * Next.js only inlines process.env.NEXT_PUBLIC_* with LITERAL dot access
 * at build time. Dynamic access like process.env[key] is NOT inlined.
 *
 * We use direct process.env references as build-time fallbacks, and
 * window.__ENV (injected by entrypoint.sh) for runtime overrides.
 */


declare global {
  interface Window {
    __ENV?: Record<string, string>;
  }
}

function getEnv(key: string, buildTimeValue: string | undefined, fallback: string = ""): string {
  // Client-side: read from runtime injection (entrypoint.sh → env.js)
  if (typeof window !== "undefined" && window.__ENV?.[key]) {
    return window.__ENV[key];
  }
  // Build-time value (inlined by Next.js) or fallback
  return buildTimeValue || fallback;
}

/**
 * API base URL. Order: runtime env.js value, build-time value, the page's own
 * origin (self-host: Traefik routes /api on the dashboard host), then a
 * server-side fallback. Empty strings count as unset.
 */
export function resolveApiBaseUrl(
  runtime: string | undefined,
  buildTime: string | undefined,
  origin: string | undefined
): string {
  const pick = [runtime, buildTime, origin].find((v) => v && v.trim() !== "");
  return (pick ?? "http://localhost:8080").replace(/\/+$/, "");
}

// Use direct process.env.NEXT_PUBLIC_* access so Next.js inlines them at build time
export const API_BASE_URL = resolveApiBaseUrl(
  typeof window !== "undefined" ? window.__ENV?.NEXT_PUBLIC_API_URL : undefined,
  process.env.NEXT_PUBLIC_API_URL,
  typeof window !== "undefined" ? window.location.origin : undefined
);
export const DEMO_MODE = getEnv("NEXT_PUBLIC_DEMO_MODE", process.env.NEXT_PUBLIC_DEMO_MODE, "false") === "true";
export const ZENITH_MODE = getEnv("NEXT_PUBLIC_ZENITH_MODE", process.env.NEXT_PUBLIC_ZENITH_MODE, "standalone");
export const LANDING_URL = getEnv(
  "NEXT_PUBLIC_LANDING_URL",
  process.env.NEXT_PUBLIC_LANDING_URL,
  "https://freezenith.com"
);
export const IS_STANDALONE = ZENITH_MODE !== "saas";
