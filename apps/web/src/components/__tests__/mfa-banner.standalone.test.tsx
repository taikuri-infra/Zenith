import { describe, it, expect, vi } from "vitest";
import { render, waitFor } from "@testing-library/react";

const planGet = vi.fn().mockResolvedValue({ tier: "pro" });
const mfaStatus = vi.fn().mockResolvedValue({ status: "disabled" });

vi.mock("@/lib/runtime-env", () => ({ IS_STANDALONE: true }));
vi.mock("@/lib/get-api", () => ({
  isDemoMode: () => false,
  getApi: () => ({ userPlan: { get: planGet }, mfa: { getStatus: mfaStatus } }),
}));

import { MFABanner } from "../mfa-banner";
import { SHOW_PLANS } from "@/lib/edition";

describe("MFABanner (standalone)", () => {
  it("does not ask for the plan", async () => {
    expect(SHOW_PLANS).toBe(false);
    const { container } = render(<MFABanner />);
    await new Promise((r) => setTimeout(r, 20));
    await waitFor(() => expect(planGet).not.toHaveBeenCalled());
    expect(container.textContent).toBe("");
  });
});
