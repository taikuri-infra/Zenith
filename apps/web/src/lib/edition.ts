import { IS_STANDALONE } from "@/lib/runtime-env";

/** Plans, upgrades and referrals exist only in the SaaS edition. */
export const SHOW_PLANS = !IS_STANDALONE;

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
