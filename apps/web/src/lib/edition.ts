import { IS_STANDALONE } from "@/lib/runtime-env";
import type { AppExposure } from "@/lib/api";

/** Plans, upgrades and referrals exist only in the SaaS edition. */
export const SHOW_PLANS = !IS_STANDALONE;

/**
 * Exposure for the deploy wizard's "API service" category. SaaS puts it behind
 * the JWT gateway; self-host has no gateway and treats "protected" as private
 * (no public route), so API services there are public.
 */
export const API_SERVICE_EXPOSURE: AppExposure = IS_STANDALONE ? "public" : "protected";

/** Public host of a deployed app, taken from the URL the API reports. */
export function appHost(app: { url?: string; subdomain?: string }): string {
  if (app.url) {
    try {
      return new URL(app.url).host;
    } catch {
      /* fall through */
    }
  }
  return app.subdomain ?? "";
}

/** DNS instruction shown when adding a custom domain. */
export function customDomainInstruction(): string {
  return "First, add an A record pointing your domain to this server's public IP (the same IP as this dashboard).";
}
