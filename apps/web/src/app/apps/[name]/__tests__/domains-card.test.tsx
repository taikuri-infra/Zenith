import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";

const listDomains = vi.fn();
const addDomain = vi.fn();
vi.mock("@/components/toast", () => ({ useToast: () => ({ toast: vi.fn() }) }));
vi.mock("@/lib/get-api", () => ({
  getApi: () => ({ appsDeploy: { listDomains, addDomain, verifyDomain: vi.fn(), deleteDomain: vi.fn() } }),
}));

import { DomainsCard } from "../domains-card";
import type { DeployApp } from "@/lib/api";

const app = (t?: string) => ({ id: "a1", name: "web", app_type: t }) as unknown as DeployApp;
const dom = { id: "d1", app_id: "a1", domain: "app.example.com", status: "pending", status_message: "", tls_ready: false, last_checked_at: null, created_at: "2026-10-16T00:00:00Z" };

beforeEach(() => {
  listDomains.mockReset().mockResolvedValue([dom]);
  addDomain.mockReset().mockResolvedValue({});
});

describe("DomainsCard", () => {
  it("lists domains and adds one", async () => {
    render(<DomainsCard app={app("web")} />);
    expect(await screen.findByText("app.example.com")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /add domain/i }));
    fireEvent.change(screen.getByPlaceholderText("app.example.com"), { target: { value: "New.Example.com" } });
    fireEvent.click(screen.getByRole("button", { name: /^add$/i }));
    await waitFor(() => expect(addDomain).toHaveBeenCalledWith("a1", "new.example.com"));
  });
  it("renders nothing for a worker app", () => {
    const { container } = render(<DomainsCard app={app("worker")} />);
    expect(container).toBeEmptyDOMElement();
    expect(listDomains).not.toHaveBeenCalled();
  });
  it("shows an error state with retry when listing fails", async () => {
    listDomains.mockRejectedValue(new Error("boom"));
    render(<DomainsCard app={app()} />);
    expect(await screen.findByText(/boom/)).toBeInTheDocument();
  });
});
