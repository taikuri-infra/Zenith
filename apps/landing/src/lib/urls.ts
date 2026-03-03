/**
 * Centralized URL configuration.
 *
 * All cross-app links (e.g. landing -> web dashboard) MUST use these helpers
 * instead of hardcoding domains. The base URL is read from NEXT_PUBLIC_APP_URL
 * which is set per-environment (staging vs production) at build time.
 *
 * Default: https://app.freezenith.com (production)
 */

const APP_URL =
  process.env.NEXT_PUBLIC_APP_URL?.replace(/\/+$/, "") ||
  "https://app.freezenith.com";

/** Full URL to the web dashboard login page */
export const loginUrl = `${APP_URL}/login`;

/** Full URL to the web dashboard register page */
export const registerUrl = `${APP_URL}/register`;

/** Full URL to the web dashboard root */
export const dashboardUrl = APP_URL;
