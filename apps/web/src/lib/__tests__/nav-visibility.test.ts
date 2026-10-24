import { describe, it, expect } from "vitest";
import { isNavItemVisible } from "../nav-visibility";

describe("isNavItemVisible", () => {
  const feature = {};
  const cloud = { cloudOnly: true };

  it("standalone shows every feature and hides cloud-only items", () => {
    const ctx = { standalone: true };
    expect(isNavItemVisible(feature, ctx)).toBe(true);
    expect(isNavItemVisible(cloud, ctx)).toBe(false);
  });

  it("saas mode shows everything", () => {
    const ctx = { standalone: false };
    expect(isNavItemVisible(feature, ctx)).toBe(true);
    expect(isNavItemVisible(cloud, ctx)).toBe(true);
  });
});
