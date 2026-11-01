import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ApiError } from "@/lib/api";

const listRules = vi.fn();
const createRule = vi.fn();
const updateRule = vi.fn();
vi.mock("@/lib/get-api", () => ({
  getApi: () => ({
    appsDeploy: { list: () => Promise.resolve({ items: [{ id: "app-1", name: "web" }] }) },
    networkPolicies: { listRules, createRule, updateRule, deleteRule: vi.fn() },
  }),
  isDemoMode: () => false,
}));
vi.mock("@/components/shell", () => ({ Shell: ({ children }: { children: React.ReactNode }) => <div>{children}</div> }));

import FirewallPage from "../page";

const rule = {
  id: "r1", user_id: "u", app_id: "app-1", name: "web-in", direction: "ingress", action: "allow",
  enabled: true, priority: 0, config: { cidrs: ["10.0.0.0/8"] }, created_at: "", updated_at: "",
};

beforeEach(() => {
  listRules.mockReset();
  createRule.mockReset();
  updateRule.mockReset();
});

async function openForm() {
  render(<FirewallPage />);
  fireEvent.click(await screen.findByText("Add Rule"));
  const name = await screen.findByPlaceholderText("e.g. Allow internal traffic");
  fireEvent.change(name, { target: { value: "deny-bad" } });
}

describe("FirewallPage", () => {
  it("shows server validation message", async () => {
    listRules.mockResolvedValue({ rules: [], total: 0 });
    createRule.mockRejectedValue(
      new ApiError(400, "Bad Request", { code: 400, message: "deny rules and FQDN egress need Cilium" })
    );
    await openForm();
    fireEvent.click(screen.getByRole("button", { name: "Create Rule" }));
    expect(await screen.findByText("deny rules and FQDN egress need Cilium")).toBeInTheDocument();
  });

  it("warns when a rule is saved but not applied", async () => {
    listRules.mockResolvedValue({ rules: [rule], total: 1 });
    updateRule.mockRejectedValue(
      new ApiError(502, "Bad Gateway", { code: 502, message: "firewall rule saved but not applied: timeout" })
    );
    render(<FirewallPage />);
    await screen.findByText("web-in");
    fireEvent.click(screen.getByTitle(/disable|enable|toggle/i));
    expect(await screen.findByText("firewall rule saved but not applied: timeout")).toBeInTheDocument();
  });
});
