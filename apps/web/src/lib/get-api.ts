/**
 * Returns either the demo API or the real API depending on
 * the NEXT_PUBLIC_DEMO_MODE environment variable.
 */

import { auth, projects, apps, databases, storage } from "./api";
import { demoApi } from "./demo-api";

const realApi = { auth, projects, apps, databases, storage };

export function getApi() {
  if (process.env.NEXT_PUBLIC_DEMO_MODE === "true") {
    return demoApi;
  }
  return realApi;
}

export function isDemoMode(): boolean {
  return process.env.NEXT_PUBLIC_DEMO_MODE === "true";
}
