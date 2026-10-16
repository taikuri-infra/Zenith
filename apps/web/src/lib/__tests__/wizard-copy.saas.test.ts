import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/runtime-env", () => ({ IS_STANDALONE: false }));
import { wizardPlanCopy } from "../wizard-copy";

describe("wizardPlanCopy (saas)", () => {
  it("keeps the plan text for free users", () => {
    const copy = wizardPlanCopy(false);
    expect(copy.registryHint).toBe("Upgrade to Pro for Zenith's built-in registry with image browsing");
    expect(copy.s3Hint).toMatch(/^Upgrade to Pro to attach S3 buckets/);
    expect(copy.resourcesNote).toBe("Free plan per-app limit.");
    expect(copy.resourcesUpgrade).toBe("Upgrade to Pro for more resources.");
    expect(copy.reviewPlanLabel).toBe("(Free plan)");
  });
  it("keeps the plan text for pro users", () => {
    const copy = wizardPlanCopy(true);
    expect(copy.registryHint).toBeNull();
    expect(copy.resourcesNote).toBe("Pro plan per-app limit.");
    expect(copy.resourcesUpgrade).toBeNull();
    expect(copy.reviewPlanLabel).toBe("(Pro plan)");
  });
});
