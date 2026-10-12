import { describe, it, expect } from "vitest";
import { resolveApiBaseUrl } from "../runtime-env";

describe("resolveApiBaseUrl", () => {
  it("runtime value wins", () => {
    expect(resolveApiBaseUrl("https://api.example.com", "https://build", "https://origin")).toBe("https://api.example.com");
  });
  it("build-time value when runtime is empty", () => {
    expect(resolveApiBaseUrl("", "https://build", "https://origin")).toBe("https://build");
  });
  it("browser origin when runtime and build-time are empty", () => {
    expect(resolveApiBaseUrl("", "", "https://zenith.example.com")).toBe("https://zenith.example.com");
    expect(resolveApiBaseUrl(undefined, undefined, "http://1.2.3.4")).toBe("http://1.2.3.4");
  });
  it("server fallback when nothing is known", () => {
    expect(resolveApiBaseUrl(undefined, undefined, undefined)).toBe("http://localhost:8080");
  });
  it("strips a trailing slash", () => {
    expect(resolveApiBaseUrl("https://a.example.com/", "", "")).toBe("https://a.example.com");
  });
  it("never returns a freezenith host unless it was configured", () => {
    expect(resolveApiBaseUrl("", "", "https://my.box")).not.toContain("freezenith");
  });
});
