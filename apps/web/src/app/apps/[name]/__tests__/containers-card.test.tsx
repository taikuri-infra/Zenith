import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import type { DeployApp } from "@/lib/api";

const getPods = vi.fn();
vi.mock("@/lib/get-api", () => ({
  getApi: () => ({ monitoring: { getPods } }),
  isDemoMode: () => false,
}));

import { ContainersCard } from "../containers-card";

const app = { id: "app-1", name: "web", app_type: "web" } as DeployApp;

beforeEach(() => {
  getPods.mockReset();
});

describe("ContainersCard", () => {
  it("lists pods with status, restarts and reason", async () => {
    getPods.mockResolvedValue({
      pods: [
        { name: "web-abc", status: "Running", ready: true, restarts: 0, cpu_millicores: 1, memory_mb: 1, started_at: "2026-03-08T08:00:00Z" },
        { name: "web-def", status: "Waiting", ready: false, restarts: 3, cpu_millicores: 0, memory_mb: 0, started_at: "", status_reason: "CrashLoopBackOff" },
      ],
    });
    render(<ContainersCard app={app} />);
    expect(await screen.findByText("web-abc")).toBeInTheDocument();
    expect(screen.getByText("web-def")).toBeInTheDocument();
    expect(screen.getByText("CrashLoopBackOff")).toBeInTheDocument();
    expect(screen.getByText("3")).toBeInTheDocument();
    expect(getPods).toHaveBeenCalledWith("app-1");
  });

  it("shows an empty state", async () => {
    getPods.mockResolvedValue({ pods: [] });
    render(<ContainersCard app={app} />);
    expect(await screen.findByText(/No containers/)).toBeInTheDocument();
  });

  it("shows an error with retry", async () => {
    getPods.mockRejectedValue(new Error("boom"));
    render(<ContainersCard app={app} />);
    expect(await screen.findByText(/boom/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /retry|try again/i })).toBeInTheDocument();
  });
});
