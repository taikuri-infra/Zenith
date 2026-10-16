import { describe, it, expect } from "vitest";
import { isManagedEnv, managedEnvLabel } from "../env-source";

describe("env-source", () => {
  it("recognises variables injected by a managed database", () => {
    expect(isManagedEnv({ source: "managed_service", source_id: "db-1" })).toBe(true);
    expect(isManagedEnv({ source: "manual" })).toBe(false);
    expect(managedEnvLabel({ source: "managed_service", source_id: "db-1" })).toBe("from database");
  });
});
