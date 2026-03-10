import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";

// Mock next/navigation
vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
    prefetch: vi.fn(),
  }),
  useSearchParams: () => new URLSearchParams(),
}));

// Mock the auth hook
const mockLogin = vi.fn();
const mockMfaLogin = vi.fn();
const mockRegister = vi.fn();
vi.mock("@/hooks/use-auth", () => ({
  useAuth: () => ({
    login: mockLogin,
    mfaLogin: mockMfaLogin,
    register: mockRegister,
    logout: vi.fn(),
    mfaToken: null,
    user: null,
    isAuthenticated: false,
    loading: false,
  }),
}));

vi.mock("@/lib/api", () => ({
  auth: {
    getOAuthUrl: vi.fn().mockReturnValue("#"),
  },
}));

vi.mock("@/lib/get-api", () => ({
  isDemoMode: vi.fn(() => false),
}));

import LoginPage from "../page";

describe("LoginPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders login form by default", () => {
    render(<LoginPage />);
    expect(screen.getByText(/sign in/i)).toBeInTheDocument();
  });

  it("renders email and password inputs", () => {
    render(<LoginPage />);
    expect(screen.getByPlaceholderText(/email/i)).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/password/i)).toBeInTheDocument();
  });

  it("has a toggle to switch to register mode", () => {
    render(<LoginPage />);
    const registerLink = screen.getByText(/create.*account|sign.*up|register/i);
    expect(registerLink).toBeInTheDocument();
  });

  it("toggles to register form on click", async () => {
    render(<LoginPage />);
    const registerLink = screen.getByText(/create.*account|sign.*up|register/i);
    fireEvent.click(registerLink);
    // After toggle, should show a name field or "Create account" heading
    expect(
      screen.getByPlaceholderText(/name/i) || screen.getByText(/create.*account/i)
    ).toBeTruthy();
  });
});
