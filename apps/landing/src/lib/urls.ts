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

/** Full URL to the web dashboard register page */
export const registerUrl = `${APP_URL}/register`;

/** Full URL to the web dashboard root */
export const dashboardUrl = APP_URL;
