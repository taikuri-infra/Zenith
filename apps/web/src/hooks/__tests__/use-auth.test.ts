import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";

// Mock the API module before importing useAuth
vi.mock("@/lib/api", () => {
  const mockAuth = {
    login: vi.fn(),
    mfaLogin: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
  };
  return {
    auth: mockAuth,
    isAuthenticated: vi.fn(() => false),
    getAccessToken: vi.fn(() => null),
    clearTokens: vi.fn(),
  };
});

vi.mock("@/lib/get-api", () => ({
  isDemoMode: vi.fn(() => false),
}));

import { useAuth } from "../use-auth";
import { auth, isAuthenticated, getAccessToken } from "@/lib/api";
import { isDemoMode } from "@/lib/get-api";

// Helper: create a fake JWT with payload
function fakeJwt(payload: Record<string, unknown>): string {
  const header = btoa(JSON.stringify({ alg: "HS256" }));
  const body = btoa(JSON.stringify(payload));
  return `${header}.${body}.signature`;
}

describe("useAuth", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    vi.mocked(isDemoMode).mockReturnValue(false);
    vi.mocked(isAuthenticated).mockReturnValue(false);
    vi.mocked(getAccessToken).mockReturnValue(null);
  });

  it("starts as unauthenticated", () => {
    const { result } = renderHook(() => useAuth());
    expect(result.current.isAuthenticated).toBe(false);
    expect(result.current.user).toBeNull();
  });

  it("login success sets user state", async () => {
    const jwt = fakeJwt({ email: "test@test.com", name: "Test", role: "admin" });
    vi.mocked(auth.login).mockResolvedValue({
      access_token: jwt,
      refresh_token: "refresh",
      mfa_required: false,
      mfa_token: "",
    });

    const { result } = renderHook(() => useAuth());
    let loginResult: boolean | "mfa_required";

    await act(async () => {
      loginResult = await result.current.login("test@test.com", "pass");
    });

    expect(loginResult!).toBe(true);
    expect(result.current.isAuthenticated).toBe(true);
    expect(result.current.user?.email).toBe("test@test.com");
  });

  it("login returns mfa_required when MFA is needed", async () => {
    vi.mocked(auth.login).mockResolvedValue({
      access_token: "",
      refresh_token: "",
      mfa_required: true,
      mfa_token: "mfa-token-123",
    });

    const { result } = renderHook(() => useAuth());
    let loginResult: boolean | "mfa_required";

    await act(async () => {
      loginResult = await result.current.login("test@test.com", "pass");
    });

    expect(loginResult!).toBe("mfa_required");
    expect(result.current.mfaToken).toBe("mfa-token-123");
    expect(result.current.isAuthenticated).toBe(false);
  });

  it("logout clears user state", async () => {
    const jwt = fakeJwt({ email: "test@test.com", name: "Test", role: "admin" });
    vi.mocked(auth.login).mockResolvedValue({
      access_token: jwt,
      refresh_token: "refresh",
      mfa_required: false,
      mfa_token: "",
    });

    const { result } = renderHook(() => useAuth());

    await act(async () => {
      await result.current.login("test@test.com", "pass");
    });
    expect(result.current.isAuthenticated).toBe(true);

    act(() => {
      result.current.logout();
    });

    expect(result.current.isAuthenticated).toBe(false);
    expect(result.current.user).toBeNull();
  });

  it("demo mode bypasses auth", () => {
    vi.mocked(isDemoMode).mockReturnValue(true);

    const { result } = renderHook(() => useAuth());
    expect(result.current.isAuthenticated).toBe(true);
    expect(result.current.user?.email).toBe("demo@zenith.dev");
    expect(result.current.loading).toBe(false);
  });
});
