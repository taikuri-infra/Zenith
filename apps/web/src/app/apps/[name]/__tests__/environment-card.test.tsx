import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import type { DeployApp } from "@/lib/api";

const m = vi.hoisted(() => ({
  envList: vi.fn(),
  envSet: vi.fn(),
  envDelete: vi.fn(),
  envImport: vi.fn(),
  envApply: vi.fn(),
  environmentsList: vi.fn(),
  listSecrets: vi.fn(),
  getSecretValue: vi.fn(),
  setSecret: vi.fn(),
  deleteSecret: vi.fn(),
}));

vi.mock("@/components/toast", () => ({ useToast: () => ({ toast: vi.fn() }) }));
vi.mock("@/lib/get-api", () => ({
  isDemoMode: () => false,
  getApi: () => ({
    environments: { list: m.environmentsList },
    envVarsV2: { list: m.envList, set: m.envSet, delete: m.envDelete, importDotEnv: m.envImport, apply: m.envApply },
    appsDeploy: {
      listSecrets: m.listSecrets,
      getSecretValue: m.getSecretValue,
      setSecret: m.setSecret,
      deleteSecret: m.deleteSecret,
    },
  }),
}));

import { EnvironmentCard } from "../environment-card";

const app: DeployApp = {
  id: "app-1",
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
  created_at: "2026-10-01T00:00:00Z",
  updated_at: "2026-10-01T00:00:00Z",
};

const envItems = [
  { id: "e1", app_id: "app-1", key: "API_KEY", value: "plain-value", is_secret: false, source: "user" },
  { id: "e2", app_id: "app-1", key: "DATABASE_URL", value: "postgres://x", is_secret: true, source: "managed_service" },
];
const secrets = [
  { id: "s1", app_id: "app-1", key: "API_KEY", created_at: "2026-10-01T00:00:00Z" },
  { id: "s2", app_id: "app-1", key: "STRIPE_KEY", created_at: "2026-10-01T00:00:00Z" },
];

beforeEach(() => {
  Object.values(m).forEach((f) => f.mockReset());
  m.environmentsList.mockResolvedValue({ environments: [] });
  m.envList.mockResolvedValue({ items: envItems, total: 2 });
  m.listSecrets.mockResolvedValue({ secrets });
  m.getSecretValue.mockResolvedValue({ key: "STRIPE_KEY", value: "sk_live_123" });
  m.setSecret.mockResolvedValue({ key: "NEW_TOKEN", status: "ok" });
  m.envSet.mockResolvedValue({ items: [], total: 0 });
});

const rowOf = (text: string) => screen.getByText(text).closest("tr") as HTMLElement;

describe("EnvironmentCard", () => {
  it("renders env vars and secret-store rows in one table, colliding keys as two rows", async () => {
    const { container } = render(<EnvironmentCard app={app} />);
    expect(screen.getByRole("heading", { name: "Environment" })).toBeInTheDocument();
    expect(container.firstElementChild).toHaveClass("md:col-span-2");
    await screen.findByText("STRIPE_KEY");
    expect(screen.getAllByText("API_KEY")).toHaveLength(2);
    expect(screen.getByText("DATABASE_URL")).toBeInTheDocument();
    expect(screen.getByText("from database")).toBeInTheDocument();
    expect(screen.getAllByText("secret store")).toHaveLength(2);
    expect(screen.getAllByRole("table")).toHaveLength(1);
    expect(m.envList).toHaveBeenCalledWith("app-1", "");
  });

  it("reveals a secret-store value on demand, fetching it only once", async () => {
    render(<EnvironmentCard app={app} />);
    await screen.findByText("STRIPE_KEY");
    fireEvent.click(within(rowOf("STRIPE_KEY")).getByRole("button", { name: /reveal/i }));
    expect(await screen.findByText("sk_live_123")).toBeInTheDocument();
    expect(m.getSecretValue).toHaveBeenCalledWith("app-1", "STRIPE_KEY");
    fireEvent.click(within(rowOf("STRIPE_KEY")).getByRole("button", { name: /hide/i }));
    expect(screen.queryByText("sk_live_123")).not.toBeInTheDocument();
    fireEvent.click(within(rowOf("STRIPE_KEY")).getByRole("button", { name: /reveal/i }));
    expect(await screen.findByText("sk_live_123")).toBeInTheDocument();
    expect(m.getSecretValue).toHaveBeenCalledTimes(1);
  });

  it("adds to the secret store via setSecret when chosen", async () => {
    render(<EnvironmentCard app={app} />);
    await screen.findByText("STRIPE_KEY");
    fireEvent.change(screen.getByPlaceholderText("DATABASE_URL"), { target: { value: "new_token" } });
    fireEvent.change(screen.getByPlaceholderText("postgres://..."), { target: { value: "s3cret" } });
    fireEvent.click(screen.getByTitle("Mark as secret"));
    fireEvent.change(screen.getByLabelText("Store"), { target: { value: "store" } });
    fireEvent.click(screen.getByRole("button", { name: "Add" }));
    await waitFor(() => expect(m.setSecret).toHaveBeenCalledWith("app-1", "NEW_TOKEN", "s3cret"));
    expect(m.envSet).not.toHaveBeenCalled();
  });

  it("adds an is_secret env var by default when marked secret", async () => {
    render(<EnvironmentCard app={app} />);
    await screen.findByText("STRIPE_KEY");
    fireEvent.change(screen.getByPlaceholderText("DATABASE_URL"), { target: { value: "TOKEN" } });
    fireEvent.change(screen.getByPlaceholderText("postgres://..."), { target: { value: "v" } });
    fireEvent.click(screen.getByTitle("Mark as secret"));
    fireEvent.click(screen.getByRole("button", { name: "Add" }));
    await waitFor(() =>
      expect(m.envSet).toHaveBeenCalledWith("app-1", [{ key: "TOKEN", value: "v", is_secret: true }], ""),
    );
    expect(m.setSecret).not.toHaveBeenCalled();
  });

  it("shows an error for the secret store while env rows stay", async () => {
    m.listSecrets.mockRejectedValue(new Error("secrets down"));
    render(<EnvironmentCard app={app} />);
    expect(await screen.findByText("secrets down")).toBeInTheDocument();
    expect(await screen.findByText("API_KEY")).toBeInTheDocument();
    expect(screen.queryByText("secret store")).not.toBeInTheDocument();
    m.listSecrets.mockResolvedValue({ secrets });
    fireEvent.click(screen.getByRole("button", { name: "Try Again" }));
    expect(await screen.findByText("STRIPE_KEY")).toBeInTheDocument();
  });
});
