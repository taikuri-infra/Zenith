/**
 * Returns either the demo API or the real API depending on
 * the NEXT_PUBLIC_DEMO_MODE environment variable.
 */

import { api } from "./api";
import { demoApi } from "./demo-api";
import { DEMO_MODE } from "./runtime-env";

export function getApi(): typeof api {
  if (DEMO_MODE) {
    return demoApi as unknown as typeof api;
  }
  return api;
}

export function isDemoMode(): boolean {
  return DEMO_MODE;
}
