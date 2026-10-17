import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import type { DeployApp } from "@/lib/api";

const app: DeployApp = {
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
  health_status: {
    status: "healthy",
    uptime_percent: 99.9,
    last_check: "2026-10-17T10:00:00Z",
    response_time_ms: 42,
  },
  created_at: "2026-10-01T00:00:00Z",
  updated_at: "2026-10-01T00:00:00Z",
};

vi.mock("next/navigation", () => ({
  useParams: () => ({ name: "app-123" }),
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
}));
vi.mock("@/components/shell", () => ({
  Shell: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));
vi.mock("@/components/toast", () => ({ useToast: () => ({ toast: vi.fn() }) }));
vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>();
  return { ...actual, monitoring: { getPods: vi.fn().mockResolvedValue({ pods: [] }) } };
});
vi.mock("@/lib/get-api", () => ({
  isDemoMode: () => false,
  getApi: () => ({
    appsDeploy: {
      get: vi.fn().mockResolvedValue(app),
      listDomains: vi.fn().mockResolvedValue([]),
    },
    monitoring: { getPods: vi.fn().mockResolvedValue({ pods: [] }) },
  }),
}));

import AppDetailPage from "../page";

describe("AppDetailPage (cards)", () => {
  it("renders the header and cards for a web app with no tab bar", async () => {
    render(<AppDetailPage />);
    expect(await screen.findByRole("heading", { level: 1, name: "whoami" })).toBeInTheDocument();

    expect(screen.getByRole("link", { name: /Logs/ })).toHaveAttribute("href", "/logs?app=app-123");
    expect(screen.getByRole("link", { name: /Open/ })).toHaveAttribute("href", "https://whoami.example.com");

    expect(screen.getByRole("heading", { name: "Source" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Health" })).toBeInTheDocument();

    expect(screen.queryAllByRole("tab")).toHaveLength(0);
    for (const name of ["Overview", "Secrets", "Releases"]) {
      expect(screen.queryByRole("button", { name })).not.toBeInTheDocument();
    }
    expect(screen.queryByText("staging")).not.toBeInTheDocument();
  });
});
