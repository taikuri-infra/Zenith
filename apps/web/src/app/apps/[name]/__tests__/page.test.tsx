import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import type { DeployApp } from "@/lib/api";

const webApp: DeployApp = {
  id: "app-123",
  project_id: "p1",
  user_id: "u1",
  name: "whoami",
  repo_url: "https://github.com/acme/whoami",
  branch: "main",
  framework: "go",
  status: "running",
  subdomain: "whoami",
  port: 8080,
  url: "https://whoami.example.com",
  app_type: "web",
  health_check: { path: "/healthz", interval_seconds: 10, timeout_seconds: 3 },
  created_at: "2026-10-01T00:00:00Z",
  updated_at: "2026-10-01T00:00:00Z",
};

const getApp = vi.fn();
const appsDeploy = {
  get: getApp,
  listDomains: vi.fn(),
  listDeployments: vi.fn(),
  listReleases: vi.fn(),
  listSecrets: vi.fn(),
  listAppDatabases: vi.fn(),
  listAppBuckets: vi.fn(),
};
const environments = { list: vi.fn() };
const envVarsV2 = { list: vi.fn() };

vi.mock("next/navigation", () => ({
  useParams: () => ({ name: "app-123" }),
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
}));
vi.mock("@/components/shell", () => ({
  Shell: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));
vi.mock("@/components/toast", () => ({ useToast: () => ({ toast: vi.fn() }) }));
vi.mock("@/hooks/use-deploy-events", () => ({ useDeployEvents: () => ({}) }));
vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>();
  return { ...actual, monitoring: { getPods: vi.fn().mockResolvedValue({ pods: [] }) } };
});
vi.mock("@/lib/get-api", () => ({
  isDemoMode: () => false,
  getApi: () => ({
    appsDeploy,
    environments,
    envVarsV2,
    monitoring: { getPods: vi.fn().mockResolvedValue({ pods: [] }) },
  }),
}));

import AppDetailPage from "../page";

beforeEach(() => {
  vi.clearAllMocks();
  getApp.mockResolvedValue(webApp);
  appsDeploy.listDomains.mockResolvedValue([]);
  appsDeploy.listDeployments.mockResolvedValue({ items: [] });
  appsDeploy.listReleases.mockResolvedValue({ releases: [] });
  appsDeploy.listSecrets.mockResolvedValue({ secrets: [] });
  appsDeploy.listAppDatabases.mockResolvedValue([]);
  appsDeploy.listAppBuckets.mockResolvedValue([]);
  environments.list.mockResolvedValue({ environments: [] });
  envVarsV2.list.mockResolvedValue({ items: [] });
});

describe("AppDetailPage (cards)", () => {
  it("renders the header and every card for a web app with no tab bar", async () => {
    render(<AppDetailPage />);
    expect(await screen.findByRole("heading", { level: 1, name: "whoami" })).toBeInTheDocument();

    expect(screen.getByRole("link", { name: /Logs/ })).toHaveAttribute("href", "/logs?app=app-123");
    expect(screen.getByRole("link", { name: /Open/ })).toHaveAttribute("href", "https://whoami.example.com");

    const cards = ["Source", "Domains", "Recent Deployments", "Environment", "Resources"];
    const headings = cards.map((name) => screen.getByRole("heading", { level: 2, name }));
    // Card order: Source + Domains share row 1, then the full-width cards.
    const sections = headings.map((h) => h.closest("section") as HTMLElement);
    sections.forEach((s) => expect(s).not.toBeNull());
    for (let i = 1; i < sections.length; i++) {
      expect(sections[i - 1].compareDocumentPosition(sections[i]) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    }
    expect(sections[0]).not.toHaveClass("md:col-span-2");
    expect(sections[1]).not.toHaveClass("md:col-span-2");
    for (const s of sections.slice(2)) expect(s).toHaveClass("md:col-span-2");
    expect(screen.queryByRole("heading", { name: "Health" })).not.toBeInTheDocument();
    expect(screen.queryAllByRole("heading", { level: 3 })).toHaveLength(0);
    await waitFor(() => expect(appsDeploy.listDeployments).toHaveBeenCalledWith("app-123"));
    await waitFor(() => expect(envVarsV2.list).toHaveBeenCalled());
    expect(appsDeploy.listSecrets).toHaveBeenCalledWith("app-123");
    expect(appsDeploy.listAppDatabases).toHaveBeenCalledWith("app-123");
    expect(appsDeploy.listAppBuckets).toHaveBeenCalledWith("app-123");
    expect(await screen.findByText("No deployments yet")).toBeInTheDocument();

    expect(screen.queryAllByRole("tab")).toHaveLength(0);
    for (const name of ["Overview", "Secrets", "Releases", "Logs", "Auth", "Databases", "Storage"]) {
      expect(screen.queryByRole("button", { name })).not.toBeInTheDocument();
    }
    expect(screen.queryByText("staging")).not.toBeInTheDocument();
  });

  it("renders a worker app without Health, Domains or Open; Source shows the command", async () => {
    getApp.mockResolvedValue({
      ...webApp,
      app_type: "worker",
      url: "",
      subdomain: "",
      command: "node worker.js",
    });
    render(<AppDetailPage />);
    expect(await screen.findByRole("heading", { level: 1, name: "whoami" })).toBeInTheDocument();
    const source = screen.getByRole("heading", { level: 2, name: "Source" }).closest("section");
    expect(source).toHaveClass("md:col-span-2");
    expect(screen.getByText("node worker.js")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Health" })).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Domains" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /Open/ })).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Logs/ })).toHaveAttribute("href", "/logs?app=app-123");
    expect(appsDeploy.listDomains).not.toHaveBeenCalled();
  });

  it("renders a cron app with its schedule in Source", async () => {
    getApp.mockResolvedValue({
      ...webApp,
      app_type: "cron",
      url: "",
      subdomain: "",
      cron_schedule: "*/5 * * * *",
    });
    render(<AppDetailPage />);
    expect(await screen.findByRole("heading", { level: 1, name: "whoami" })).toBeInTheDocument();
    // header + Source "Schedule" row
    expect(screen.getAllByText("*/5 * * * *").length).toBeGreaterThanOrEqual(2);
    expect(screen.getByText("Schedule")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Domains" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /Open/ })).not.toBeInTheDocument();
  });

  it("shows the header branch only when the app has a repository", async () => {
    getApp.mockResolvedValue({ ...webApp, repo_url: "", branch: "release-x" });
    render(<AppDetailPage />);
    expect(await screen.findByRole("heading", { level: 1, name: "whoami" })).toBeInTheDocument();
    expect(screen.queryByText("release-x")).not.toBeInTheDocument();
  });
});
