/**
 * Returns either the demo API or the real API depending on
 * the NEXT_PUBLIC_DEMO_MODE environment variable.
 */

import { auth, projects, apps, databases, storage, storageBuckets, appsDeploy, userDatabases, standaloneDatabases, notifications, activity, userPlan, apiKeys, sessions, mfa, webhooks, roles, ipWhitelist, compliance, dpa, branding, sso, previews, autoscaler, billing, registry, gateways, authPools, team, support, monitoring, audit, addons, podSessions, waf, networkPolicies, alerts, onboarding, referral, exitSurvey, managedServices, composeImport, imageVerify, registryCredentials, envVarsV2, ai, ciTemplates, environments, deployTokens } from "./api";
import { demoApi } from "./demo-api";
import { DEMO_MODE } from "./runtime-env";

const realApi = { auth, projects, apps, databases, storage, storageBuckets, appsDeploy, userDatabases, standaloneDatabases, notifications, activity, userPlan, apiKeys, sessions, mfa, webhooks, roles, ipWhitelist, compliance, dpa, branding, sso, previews, autoscaler, billing, registry, gateways, authPools, team, support, monitoring, audit, addons, podSessions, waf, networkPolicies, alerts, onboarding, referral, exitSurvey, managedServices, composeImport, imageVerify, registryCredentials, envVarsV2, ai, ciTemplates, environments, deployTokens };

export function getApi() {
  if (DEMO_MODE) {
    return demoApi;
  }
  return realApi;
}

export function isDemoMode(): boolean {
  return DEMO_MODE;
}
