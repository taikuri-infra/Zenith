import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";

let searchParams = new URLSearchParams();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), prefetch: vi.fn() }),
  useSearchParams: () => searchParams,
}));
vi.mock("@/hooks/use-auth", () => ({
  useAuth: () => ({
    login: vi.fn(), mfaLogin: vi.fn(), register: vi.fn(), logout: vi.fn(),
    mfaToken: null, user: null, isAuthenticated: false, loading: false,
  }),
}));
vi.mock("@/lib/api", () => ({ auth: { getOAuthUrl: vi.fn().mockReturnValue("#") } }));
vi.mock("@/lib/get-api", () => ({ isDemoMode: vi.fn(() => false) }));
vi.mock("@/lib/runtime-env", () => ({ IS_STANDALONE: true }));

import LoginPage from "../page";

describe("LoginPage (standalone)", () => {
  beforeEach(() => { searchParams = new URLSearchParams(); });

  it("shows only email and password", () => {
    render(<LoginPage />);
    expect(screen.getByPlaceholderText("you@example.com")).toBeInTheDocument();
    expect(screen.queryByText("Google")).toBeNull();
    expect(screen.queryByText("GitHub")).toBeNull();
    expect(screen.queryByText(/create.*account|sign.*up/i)).toBeNull();
    expect(screen.queryByText(/Hetzner|Kubernetes/)).toBeNull();
    expect(screen.queryByText(/Terms of Service/)).toBeNull();
    expect(screen.getByText("Self-hosted PaaS")).toBeInTheDocument();
  });

  it("shows the login form even with ?mode=register", () => {
    searchParams = new URLSearchParams("mode=register");
    render(<LoginPage />);
    expect(screen.getByText(/sign in to your account/i)).toBeInTheDocument();
    expect(screen.queryByPlaceholderText(/name/i)).toBeNull();
  });
});
