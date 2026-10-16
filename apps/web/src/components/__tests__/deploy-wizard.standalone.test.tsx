import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";

vi.mock("@/lib/runtime-env", () => ({ IS_STANDALONE: true }));
vi.mock("@/components/toast", () => ({ useToast: () => ({ toast: vi.fn() }) }));
vi.mock("@/lib/get-api", () => ({
  getApi: () => ({
    appsDeploy: { checkName: vi.fn().mockResolvedValue({ available: true, url: "" }) },
    registry: { listImages: vi.fn().mockResolvedValue({ items: [] }) },
    storage: { list: vi.fn().mockResolvedValue({ items: [] }) },
    databases: { list: vi.fn().mockResolvedValue({ items: [] }) },
  }),
}));

import { DeployWizard } from "../deploy-wizard";

const PLAN_TEXT = /\bPro\b|Free plan|Upgrade/;

describe("DeployWizard (standalone)", () => {
  it("shows no plan or upgrade text on any step", () => {
    render(<DeployWizard onClose={() => {}} isPro={false} projectId="p1" />);
    const next = () => fireEvent.click(screen.getByRole("button", { name: "Next" }));
    const seen: string[] = [];
    const snap = () => seen.push(document.body.textContent ?? "");

    snap(); // Type
    next();
    fireEvent.change(screen.getByPlaceholderText("nginx, user/app, or ghcr.io/org/image:tag"), { target: { value: "nginx:1" } });
    snap(); // Image
    next();
    fireEvent.change(screen.getByPlaceholderText("my-app"), { target: { value: "web" } });
    snap(); // Config (resources tile)
    next();
    expect(screen.getByText("S3 Object Storage")).toBeInTheDocument();
    snap(); // Resources (S3 tile)
    next();
    expect(screen.getByText("nginx:1")).toBeInTheDocument();
    snap(); // Review

    seen.forEach((text, i) => expect(text, `step ${i}`).not.toMatch(PLAN_TEXT));
    expect(seen[3]).toMatch(/Storage page/);
  });
});
