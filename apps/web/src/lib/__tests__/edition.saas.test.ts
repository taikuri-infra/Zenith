import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/runtime-env", () => ({ IS_STANDALONE: false }));
import { SHOW_PLANS } from "../edition";

describe("edition (saas)", () => {
  it("shows plans", () => {
    expect(SHOW_PLANS).toBe(true);
  });
});
