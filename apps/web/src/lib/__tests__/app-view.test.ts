import { describe, it, expect } from "vitest";
import {
  statusTone, frameworkLabel, sourceLabel, sourceKind, timeAgo, formatMemory, formatPercent, shortRef, appTypeLabel,
} from "../app-view";

describe("statusTone", () => {
  it("maps statuses to tones", () => {
    expect(statusTone("running")).toBe("ok");
    expect(statusTone("ready")).toBe("ok");
    expect(statusTone("building")).toBe("warn");
    expect(statusTone("failed")).toBe("bad");
    expect(statusTone("stopped")).toBe("idle");
    expect(statusTone("whatever")).toBe("idle");
  });
});

describe("frameworkLabel", () => {
  it("hides unknown and empty", () => {
    expect(frameworkLabel("unknown")).toBeNull();
    expect(frameworkLabel(undefined)).toBeNull();
  });
  it("labels known frameworks and passes others through", () => {
    expect(frameworkLabel("nextjs")).toBe("Next.js");
    expect(frameworkLabel("laravel")).toBe("laravel");
  });
});

describe("sourceLabel / sourceKind", () => {
  it("shortens docker hub images", () => {
    const app = { deploy_source: "image" as const, image_url: "docker.io/traefik/whoami:latest", repo_url: "", branch: "" };
    expect(sourceLabel(app)).toBe("traefik/whoami:latest");
    expect(sourceKind(app)).toBe("image");
  });
  it("shows repo and branch for git apps", () => {
    const app = { deploy_source: "git" as const, image_url: "", repo_url: "https://github.com/acme/site.git", branch: "main" };
    expect(sourceLabel(app)).toBe("acme/site @ main");
    expect(sourceKind(app)).toBe("git");
  });
  it("treats an app with only an image as an image app", () => {
    expect(sourceKind({ deploy_source: undefined, repo_url: "" })).toBe("image");
  });
});

describe("timeAgo", () => {
  const now = Date.parse("2026-10-24T12:00:00Z");
  it("formats ranges", () => {
    expect(timeAgo("2026-10-24T11:59:40Z", now)).toBe("just now");
    expect(timeAgo("2026-10-24T11:30:00Z", now)).toBe("30 min ago");
    expect(timeAgo("2026-10-24T07:00:00Z", now)).toBe("5 h ago");
    expect(timeAgo("2026-10-21T12:00:00Z", now)).toBe("3 d ago");
  });
  it("is safe on bad input and future times", () => {
    expect(timeAgo(undefined, now)).toBe("—");
    expect(timeAgo("nope", now)).toBe("—");
    expect(timeAgo("2026-10-24T13:00:00Z", now)).toBe("just now");
  });
});

describe("number formatting", () => {
  it("formats memory", () => {
    expect(formatMemory(64)).toBe("64 MB");
    expect(formatMemory(1536)).toBe("1.5 GB");
    expect(formatMemory(undefined)).toBe("—");
  });
  it("formats percent", () => {
    expect(formatPercent(2.54)).toBe("2.5%");
    expect(formatPercent(37.6)).toBe("38%");
    expect(formatPercent(undefined)).toBe("—");
  });
});

describe("shortRef / appTypeLabel", () => {
  it("shortens a sha, keeps short refs", () => {
    expect(shortRef("a".repeat(40))).toBe("aaaaaaa");
    expect(shortRef("docker.io/traefik/whoami:latest")).toBe("traefik/whoami:latest");
    expect(shortRef(undefined)).toBe("—");
  });
  it("labels app types", () => {
    expect(appTypeLabel(undefined)).toBe("Web");
    expect(appTypeLabel("worker")).toBe("Worker");
    expect(appTypeLabel("cron")).toBe("Cron");
  });
});
