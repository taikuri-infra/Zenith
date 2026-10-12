import { describe, it, expect } from "vitest";
import { engineOptions } from "../database-engines";

const byId = (standalone: boolean) => Object.fromEntries(engineOptions(standalone).map((e) => [e.id, e]));

describe("engineOptions", () => {
  it("self-host offers PostgreSQL, MySQL and Redis; MongoDB is unavailable", () => {
    const e = byId(true);
    expect(e.postgresql.enabled).toBe(true);
    expect(e.mysql.enabled).toBe(true);
    expect(e.redis.enabled).toBe(true);
    expect(e.mongodb.enabled).toBe(false);
    expect(e.mongodb.disabledNote).toBe("Not available on self-host");
  });
  it("saas keeps the previous flags: MySQL coming soon, others enabled", () => {
    const e = byId(false);
    expect(e.postgresql.enabled).toBe(true);
    expect(e.mysql.enabled).toBe(false);
    expect(e.mysql.disabledNote).toBe("Coming Soon");
    expect(e.mongodb.enabled).toBe(true);
    expect(e.redis.enabled).toBe(true);
  });
  it("keeps the engine order", () => {
    expect(engineOptions(true).map((e) => e.id)).toEqual(["postgresql", "mysql", "mongodb", "redis"]);
    expect(engineOptions(false).map((e) => e.id)).toEqual(["postgresql", "mysql", "mongodb", "redis"]);
  });
});
