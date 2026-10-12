import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/runtime-env", () => ({ IS_STANDALONE: false }));
import { SHOW_PLANS, API_SERVICE_EXPOSURE } from "../edition";

describe("edition (saas)", () => {
  it("shows plans", () => {
    expect(SHOW_PLANS).toBe(true);
  });
  it("API services are protected by the JWT gateway", () => {
    expect(API_SERVICE_EXPOSURE).toBe("protected");
  });
});
