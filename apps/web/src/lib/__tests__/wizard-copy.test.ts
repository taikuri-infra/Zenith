import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/runtime-env", () => ({ IS_STANDALONE: true }));
import { wizardPlanCopy } from "../wizard-copy";

const PLAN_TEXT = /Pro|Free plan|Upgrade/;

describe("wizardPlanCopy (standalone)", () => {
  it.each([false, true])("has no plan or upgrade text (isPro=%s)", (isPro) => {
    const copy = wizardPlanCopy(isPro);
    for (const v of Object.values(copy)) {
      if (typeof v === "string") expect(v).not.toMatch(PLAN_TEXT);
    }
    expect(copy.registryHint).toBeNull();
    expect(copy.resourcesNote).toBeNull();
    expect(copy.reviewPlanLabel).toBeNull();
    expect(copy.s3Crown).toBe(false);
    expect(copy.s3Hint).toMatch(/Storage page/);
  });
});
