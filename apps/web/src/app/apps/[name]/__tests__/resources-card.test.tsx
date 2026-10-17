import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import type { DeployApp } from "@/lib/api";

const appsDeploy: Record<string, ReturnType<typeof vi.fn>> = {};
vi.mock("@/components/toast", () => ({ useToast: () => ({ toast: vi.fn() }) }));
vi.mock("@/lib/get-api", () => ({
  getApi: () => ({ appsDeploy }),
  isDemoMode: () => false,
}));

import { ResourcesCard } from "../resources-card";

const app = { id: "app-1", name: "web", app_type: "web" } as DeployApp;
const db = { id: "d1", name: "web-pg", engine: "postgresql", status: "ready", size_mb: 1, max_size_mb: 100, host: "h", port: 5432, db_user: "u", db_name: "n" };
const bucket = { id: "b1", name: "uploads", access: "private", status: "active", endpoint: "s3.local", size_mb: 0, max_size_mb: 100, objects: 0 };

beforeEach(() => {
  for (const k of Object.keys(appsDeploy)) delete appsDeploy[k];
  Object.assign(appsDeploy, {
    listAppDatabases: vi.fn().mockResolvedValue([db]),
    listAppBuckets: vi.fn().mockResolvedValue([bucket]),
    createAppDatabase: vi.fn().mockResolvedValue({}),
    listBackups: vi.fn().mockResolvedValue([]),
    createAppBucket: vi.fn().mockResolvedValue({}),
    deleteAppDatabase: vi.fn().mockResolvedValue({}),
    createBackup: vi.fn().mockResolvedValue({}),
  });
});

describe("ResourcesCard", () => {
  it("renders one database and one bucket", async () => {
    render(<ResourcesCard app={app} />);
    expect(screen.getByText("Resources")).toBeInTheDocument();
    expect(await screen.findByText("web-pg")).toBeInTheDocument();
    expect(await screen.findByText("uploads")).toBeInTheDocument();
  });

  it("shows a bucket error while database rows still render", async () => {
    appsDeploy.listAppBuckets.mockRejectedValue(new Error("Error 500: boom"));
    render(<ResourcesCard app={app} />);
    expect(await screen.findByText("web-pg")).toBeInTheDocument();
    expect(await screen.findByText(/boom/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /retry|try again/i })).toBeInTheDocument();
  });

  it("does not throw when an API method is missing", async () => {
    delete appsDeploy.listAppBuckets;
    render(<ResourcesCard app={app} />);
    expect(await screen.findByText("web-pg")).toBeInTheDocument();
  });

  it("adds a database with the chosen engine", async () => {
    render(<ResourcesCard app={app} />);
    await screen.findByText("web-pg");
    fireEvent.click(screen.getByRole("button", { name: /add database/i }));
    fireEvent.change(screen.getByLabelText("Engine"), { target: { value: "redis" } });
    fireEvent.click(screen.getByRole("button", { name: "Create database" }));
    await waitFor(() => expect(appsDeploy.createAppDatabase).toHaveBeenCalledWith("app-1", { engine: "redis" }));
  });

  it("creates a bucket with name and access", async () => {
    render(<ResourcesCard app={app} />);
    await screen.findByText("uploads");
    fireEvent.click(screen.getByRole("button", { name: /add bucket/i }));
    fireEvent.change(screen.getByLabelText("Bucket name"), { target: { value: "assets" } });
    fireEvent.change(screen.getByLabelText("Access"), { target: { value: "public" } });
    fireEvent.click(screen.getByRole("button", { name: "Create bucket" }));
    await waitFor(() => expect(appsDeploy.createAppBucket).toHaveBeenCalledWith("app-1", { name: "assets", access: "public" }));
  });

  it("deletes a database", async () => {
    render(<ResourcesCard app={app} />);
    await screen.findByText("web-pg");
    fireEvent.click(screen.getByRole("button", { name: "Delete web-pg" }));
    await waitFor(() => expect(appsDeploy.deleteAppDatabase).toHaveBeenCalledWith("app-1", "d1"));
  });

  it("creates a backup from the expanded row", async () => {
    render(<ResourcesCard app={app} />);
    await screen.findByText("web-pg");
    fireEvent.click(screen.getByRole("button", { name: "Expand web-pg" }));
    fireEvent.click(await screen.findByRole("button", { name: /create backup/i }));
    await waitFor(() => expect(appsDeploy.createBackup).toHaveBeenCalledWith("app-1", "d1"));
  });

  it("shows an error, not the empty state, when backups fail", async () => {
    appsDeploy.listBackups.mockRejectedValue(new Error("backups down"));
    render(<ResourcesCard app={app} />);
    await screen.findByText("web-pg");
    fireEvent.click(screen.getByRole("button", { name: "Expand web-pg" }));
    expect(await screen.findByText(/backups down/)).toBeInTheDocument();
    expect(screen.queryByText("No backups yet.")).not.toBeInTheDocument();
  });

  it("clears the copy-feedback timer on unmount", async () => {
    Object.defineProperty(navigator, "clipboard", { value: { writeText: vi.fn().mockResolvedValue(undefined) }, configurable: true });
    appsDeploy.getAppDatabase = vi.fn().mockResolvedValue({ connection_string: "postgres://u:p@h:5432/n" });
    const clearSpy = vi.spyOn(globalThis, "clearTimeout");
    const { unmount } = render(<ResourcesCard app={app} />);
    await screen.findByText("web-pg");
    fireEvent.click(screen.getByRole("button", { name: "Expand web-pg" }));
    fireEvent.click(await screen.findByRole("button", { name: /show connection string/i }));
    await screen.findByText("postgres://u:p@h:5432/n");
    fireEvent.click(screen.getByTitle("Copy"));
    await waitFor(() => expect(navigator.clipboard.writeText).toHaveBeenCalled());
    clearSpy.mockClear();
    unmount();
    expect(clearSpy).toHaveBeenCalled();
    clearSpy.mockRestore();
  });

  it("uses labels, not headings, for its sub-sections", async () => {
    render(<ResourcesCard app={app} />);
    await screen.findByText("web-pg");
    expect(screen.getAllByRole("heading")).toHaveLength(1);
    expect(screen.getByRole("heading", { level: 2, name: "Resources" })).toBeInTheDocument();
  });
});
