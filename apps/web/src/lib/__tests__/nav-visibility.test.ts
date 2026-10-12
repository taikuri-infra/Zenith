import { describe, it, expect } from "vitest";
import { parseLabsFlag, isNavItemVisible } from "../nav-visibility";

describe("parseLabsFlag", () => {
  it.each(["true", "TRUE", " true ", "1", "yes", "Yes"])("%j is on", (v) => {
    expect(parseLabsFlag(v)).toBe(true);
  });
  it.each(["false", "0", "", "  ", "no", "enabled", undefined])("%j is off", (v) => {
    expect(parseLabsFlag(v)).toBe(false);
  });
});

describe("isNavItemVisible", () => {
  const core = {};
  const labs = { labs: true };
  const saas = { saasOnly: true };

  it("standalone, labs off: only core", () => {
    const ctx = { standalone: true, labs: false };
    expect(isNavItemVisible(core, ctx)).toBe(true);
    expect(isNavItemVisible(labs, ctx)).toBe(false);
    expect(isNavItemVisible(saas, ctx)).toBe(false);
  });

  it("standalone, labs on: core + labs, never saas", () => {
    const ctx = { standalone: true, labs: true };
    expect(isNavItemVisible(core, ctx)).toBe(true);
    expect(isNavItemVisible(labs, ctx)).toBe(true);
    expect(isNavItemVisible(saas, ctx)).toBe(false);
  });

  it("saas mode shows everything regardless of labs flag", () => {
    for (const l of [true, false]) {
      const ctx = { standalone: false, labs: l };
      expect(isNavItemVisible(core, ctx)).toBe(true);
      expect(isNavItemVisible(labs, ctx)).toBe(true);
      expect(isNavItemVisible(saas, ctx)).toBe(true);
    }
  });

  it("drops sections with no visible items", () => {
    const ctx = { standalone: true, labs: false };
    const security = [{ labs: true }, { labs: true }, { saasOnly: true }];
    expect(security.filter((i) => isNavItemVisible(i, ctx))).toHaveLength(0);
  });
});
