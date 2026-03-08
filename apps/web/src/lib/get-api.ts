/**
 * Returns either the demo API or the real API depending on
 * the NEXT_PUBLIC_DEMO_MODE environment variable.
 */

import { auth, projects, apps, databases, storage, storageBuckets, appsDeploy, userDatabases, standaloneDatabases, notifications, activity, userPlan, apiKeys, sessions, mfa, webhooks, roles, ipWhitelist, compliance, dpa, branding, sso, previews, autoscaler, billing, registry, gateways, authPools } from "./api";
import { demoApi } from "./demo-api";

const realApi = { auth, projects, apps, databases, storage, storageBuckets, appsDeploy, userDatabases, standaloneDatabases, notifications, activity, userPlan, apiKeys, sessions, mfa, webhooks, roles, ipWhitelist, compliance, dpa, branding, sso, previews, autoscaler, billing, registry, gateways, authPools };

export function getApi() {
  if (process.env.NEXT_PUBLIC_DEMO_MODE === "true") {
    return demoApi;
  }
  return realApi;
}

export function isDemoMode(): boolean {
  return process.env.NEXT_PUBLIC_DEMO_MODE === "true";
}
