/**
 * Returns either the demo API or the real API depending on
 * the NEXT_PUBLIC_DEMO_MODE environment variable.
 */

import { api } from "./api";
import { demoApi } from "./demo-api";

export function getApi() {
  if (process.env.NEXT_PUBLIC_DEMO_MODE === "true") {
    return demoApi;
  }
  return api;
}

export function isDemoMode(): boolean {
  return process.env.NEXT_PUBLIC_DEMO_MODE === "true";
}
