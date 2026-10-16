import { describe, it, expect } from "vitest";
import { databaseSummary } from "../database-summary";

describe("databaseSummary", () => {
  it("counts every engine, including MySQL", () => {
    const s = databaseSummary([
      { engine: "postgresql", status: "ready" },
      { engine: "mysql", status: "ready" },
      { engine: "redis", status: "provisioning" },
    ]);
    expect(s).toBe("3 instances, 2 ready, 1 PostgreSQL, 1 MySQL, 1 Redis");
  });
  it("omits engines with no instances", () => {
    expect(databaseSummary([{ engine: "redis", status: "ready" }])).toBe("1 instances, 1 ready, 1 Redis");
  });
});
