import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/runtime-env", () => ({ IS_STANDALONE: true }));
import { SHOW_PLANS, appHost, customDomainInstruction } from "../edition";

describe("edition (standalone)", () => {
  it("hides plans", () => {
    expect(SHOW_PLANS).toBe(false);
  });
  it("uses the host from the app URL", () => {
    expect(appHost({ url: "https://whoami-1a2b.zenith.example.com", subdomain: "whoami-1a2b" })).toBe("whoami-1a2b.zenith.example.com");
  });
  it("falls back to the subdomain when the URL is missing or bad", () => {
    expect(appHost({ subdomain: "web-9" })).toBe("web-9");
    expect(appHost({ url: "not a url", subdomain: "web-9" })).toBe("web-9");
  });
  it("tells self-hosters to use an A record, never freezenith", () => {
    const s = customDomainInstruction();
    expect(s).toMatch(/A record/);
    expect(s).not.toMatch(/freezenith|CNAME/);
  });
});
