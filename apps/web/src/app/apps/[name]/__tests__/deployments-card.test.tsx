import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import type { DeployApp } from "@/lib/api";

vi.mock("@/lib/runtime-env", () => ({ IS_STANDALONE: true, API_BASE_URL: "", DEMO_MODE: false }));
vi.mock("@/components/toast", () => ({ useToast: () => ({ toast: vi.fn() }) }));
vi.mock("@/hooks/use-deploy-events", () => ({ useDeployEvents: () => ({}) }));
const useDeployLogs = vi.hoisted(() => vi.fn());
vi.mock("@/hooks/use-deploy-logs", () => ({ useDeployLogs }));

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
    // jsdom has no scrollIntoView (BuildLogViewer auto-scrolls).
    Element.prototype.scrollIntoView = vi.fn();
    useDeployLogs.mockImplementation((_a: string, depId: string) => ({
      entries: [{ timestamp: "2026-01-01T00:00:00Z", level: "build", message: `log line for ${depId}` }],
      streaming: false,
    }));
  });

  it("expands one deployment row at a time to show its build log", async () => {
    render(<DeploymentsCard app={app} />);
    await screen.findByText("aaaaaaaa");
    expect(useDeployLogs).not.toHaveBeenCalled();
    const toggles = screen.getAllByRole("button", { name: /Build log/ });
    expect(toggles).toHaveLength(2);
    fireEvent.click(toggles[0]);
    expect(await screen.findByText(/log line for aaaaaaaa-1/)).toBeInTheDocument();
    expect(useDeployLogs).toHaveBeenCalledWith("app-123", "aaaaaaaa-1");
    expect(toggles[0]).toHaveAttribute("aria-expanded", "true");
    fireEvent.click(toggles[1]);
    expect(await screen.findByText(/log line for bbbbbbbb-2/)).toBeInTheDocument();
    expect(screen.queryByText(/log line for aaaaaaaa-1/)).not.toBeInTheDocument();
    fireEvent.click(toggles[1]);
    expect(screen.queryByText(/log line for bbbbbbbb-2/)).not.toBeInTheDocument();
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

  it("shows the empty state for deployments while releases are still available", async () => {
    api.listDeployments.mockResolvedValue({ items: [] });
    render(<DeploymentsCard app={app} />);
    expect(await screen.findByText("No deployments yet")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Show releases/ }));
    expect(await screen.findByText("ship it")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Deploy/ })).toBeEnabled();
  });

  it("refetches the deployments list after a release deploy and clears the timer on unmount", async () => {
    const clearSpy = vi.spyOn(globalThis, "clearTimeout");
    const { unmount } = render(<DeploymentsCard app={app} />);
    await screen.findByText("aaaaaaaa");
    expect(api.listDeployments).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByRole("button", { name: /Show releases/ }));
    await screen.findByText("ship it");
    fireEvent.click(screen.getByRole("button", { name: /Deploy/ }));
    await screen.findByText("Triggered");
    await waitFor(() => expect(api.listDeployments).toHaveBeenCalledTimes(2));
    clearSpy.mockClear();
    unmount();
    expect(clearSpy).toHaveBeenCalled();
    clearSpy.mockRestore();
  });

  it("shows an error state instead of throwing when the API fails", async () => {
    api.listDeployments.mockRejectedValue(new Error("boom"));
    render(<DeploymentsCard app={app} />);
    expect(await screen.findByText(/boom/)).toBeInTheDocument();
  });
});
