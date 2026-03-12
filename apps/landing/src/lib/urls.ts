/**
 * Centralized URL configuration.
 *
 * All cross-app links (e.g. landing -> web dashboard) MUST use these helpers
 * instead of hardcoding domains. The base URL is read from runtime-env which
 * supports both build-time and runtime injection.
 *
 * Default: https://app.freezenith.com (production)
 */

import { APP_URL } from "./runtime-env";

/** Full URL to the web dashboard login page */
export const loginUrl = `${APP_URL}/login`;

/** Full URL to the web dashboard login/register page */
export const registerUrl = `${APP_URL}/login`;

/**
 * Register URL with UTM and referral params forwarded from the current page URL.
 * Call this from client components to pass marketing attribution data.
 */
export function registerUrlWithParams(): string {
  if (typeof window === "undefined") return registerUrl;
  const current = new URL(window.location.href);
  const forward = new URLSearchParams();
  for (const key of ["utm_source", "utm_medium", "utm_campaign", "utm_content", "utm_term", "ref"]) {
    const val = current.searchParams.get(key);
    if (val) forward.set(key, val);
  }
  const qs = forward.toString();
  return qs ? `${registerUrl}?${qs}` : registerUrl;
}

/** Full URL to the web dashboard root */
export const dashboardUrl = APP_URL;
