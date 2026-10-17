import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import type { DeployApp } from "@/lib/api";

vi.mock("@/lib/get-api", () => ({
  getApi: () => ({
    appsDeploy: { listDomains: vi.fn().mockResolvedValue([]) },
  }),
}));

import { SourceCard } from "../source-card";

const base: DeployApp = {
  id: "a1",
  project_id: "p1",
  user_id: "u1",
  name: "whoami",
  repo_url: "https://github.com/acme/whoami",
  branch: "develop",
  framework: "go",
  status: "running",
  subdomain: "whoami",
  port: 8080,
  url: "https://whoami.example.com",
  app_type: "web",
  created_at: "2026-10-01T00:00:00Z",
  updated_at: "2026-10-01T00:00:00Z",
};

describe("SourceCard", () => {
  it("shows repository and branch for a git app", () => {
    render(<SourceCard app={base} />);
    expect(screen.getByRole("heading", { name: "Source" })).toBeInTheDocument();
    const repo = screen.getByRole("link", { name: "acme/whoami" });
    expect(repo).toHaveAttribute("href", "https://github.com/acme/whoami");
    expect(screen.getByText("develop")).toBeInTheDocument();
  });

  it("shows the image line for an image app without repo_url", () => {
    render(<SourceCard app={{ ...base, repo_url: "" }} />);
    expect(screen.getByText("Image")).toBeInTheDocument();
    expect(screen.getByText("deployed from an image")).toBeInTheDocument();
    expect(screen.queryByText("Repository")).not.toBeInTheDocument();
  });

  it("shows the schedule for a cron app", () => {
    render(
      <SourceCard app={{ ...base, app_type: "cron", cron_schedule: "*/5 * * * *", command: "node job.js" }} />,
    );
    expect(screen.getByText("Schedule")).toBeInTheDocument();
    expect(screen.getByText("*/5 * * * *")).toBeInTheDocument();
    expect(screen.getByText("node job.js")).toBeInTheDocument();
    expect(screen.queryByText("Port")).not.toBeInTheDocument();
  });
});
