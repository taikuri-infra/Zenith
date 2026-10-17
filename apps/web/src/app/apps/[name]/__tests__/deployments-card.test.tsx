import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import type { DeployApp } from "@/lib/api";

vi.mock("@/lib/runtime-env", () => ({ IS_STANDALONE: true, API_BASE_URL: "", DEMO_MODE: false }));
vi.mock("@/components/toast", () => ({ useToast: () => ({ toast: vi.fn() }) }));
vi.mock("@/hooks/use-deploy-events", () => ({ useDeployEvents: () => ({}) }));

const api = {
  listDeployments: vi.fn(),
  rollback: vi.fn(),
  listReleases: vi.fn(),
  deployRelease: vi.fn(),
};
vi.mock("@/lib/get-api", () => ({
  getApi: () => ({ appsDeploy: api }),
  isDemoMode: () => false,
}));

import { DeploymentsCard } from "../deployments-card";

const app = { id: "app-123", name: "web" } as unknown as DeployApp;
const dep = (id: string, status: string) => ({
  id, app_id: "app-123", git_sha: "abcdef123456", status, build_log: "", deploy_log: "",
  created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z",
});

describe("DeploymentsCard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    api.listDeployments.mockResolvedValue({ items: [dep("aaaaaaaa-1", "active"), dep("bbbbbbbb-2", "stopped")] });
    api.rollback.mockResolvedValue({});
    api.listReleases.mockResolvedValue({
      releases: [{ id: "rel-1", app_id: "app-123", image: "reg/web:v9", git_sha: "1234567890", branch: "main", message: "ship it", created_at: "2026-01-02T00:00:00Z" }],
    });
    api.deployRelease.mockResolvedValue({});
  });

  it("renders deployments, rollback only on non-active row, and calls rollback", async () => {
    const { container } = render(<DeploymentsCard app={app} />);
    expect(screen.getByText("Recent Deployments")).toBeInTheDocument();
    expect(container.firstElementChild).toHaveClass("md:col-span-2");
    await screen.findByText("aaaaaaaa");
    expect(screen.getByText("bbbbbbbb")).toBeInTheDocument();
    const buttons = screen.getAllByRole("button", { name: /Rollback/ });
    expect(buttons).toHaveLength(1);
    fireEvent.click(buttons[0]);
    await waitFor(() => expect(api.rollback).toHaveBeenCalledWith("app-123", "bbbbbbbb-2"));
  });

  it("lazily loads releases and deploys one", async () => {
    render(<DeploymentsCard app={app} />);
    await screen.findByText("aaaaaaaa");
    expect(api.listReleases).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: /Show releases/ }));
    await screen.findByText("ship it");
    expect(screen.getByRole("button", { name: /Releases \(1\)/ })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Deploy/ }));
    await waitFor(() => expect(api.deployRelease).toHaveBeenCalledWith("app-123", "rel-1"));
  });

  it("shows an error state instead of throwing when the API fails", async () => {
    api.listDeployments.mockRejectedValue(new Error("boom"));
    render(<DeploymentsCard app={app} />);
    expect(await screen.findByText(/boom/)).toBeInTheDocument();
  });
});
