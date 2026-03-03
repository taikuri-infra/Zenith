"use client";

import { useCallback, useEffect, useState } from "react";
import { auth, isAuthenticated, getAccessToken, clearTokens } from "@/lib/api";
import { isDemoMode } from "@/lib/get-api";

interface User {
  email: string;
  name: string;
  role: string;
}

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  loading: boolean;
}

/**
 * Hook for managing authentication state.
 *
 * register() returns "verify_email" when email verification is required,
 * or true when the user is auto-logged in (e.g. demo mode).
 */
export function useAuth(): AuthState & {
  login: (email: string, password: string) => Promise<boolean>;
  register: (email: string, password: string, name: string) => Promise<boolean | "verify_email">;
  logout: () => void;
} {
  const demo = isDemoMode();

  const [state, setState] = useState<AuthState>({
    user: demo
      ? { email: "demo@zenith.dev", name: "Demo User", role: "admin" }
      : null,
    isAuthenticated: demo,
    loading: !demo,
  });

  useEffect(() => {
    if (demo) return;

    // Check if user is authenticated on mount
    const authenticated = isAuthenticated();
    if (authenticated) {
      // Decode JWT to get user info
      const token = getAccessToken();
      if (token) {
        try {
          const payload = JSON.parse(atob(token.split(".")[1]));
          setState({
            user: {
              email: payload.email || "",
              name: payload.name || "",
              role: payload.role || "viewer",
            },
            isAuthenticated: true,
            loading: false,
          });
          return;
        } catch {
          // Invalid token
          clearTokens();
        }
      }
    }
    setState({ user: null, isAuthenticated: false, loading: false });
  }, [demo]);

  const login = useCallback(async (email: string, password: string) => {
    if (demo) return true;
    try {
      const response = await auth.login({ email, password });
      const payload = JSON.parse(atob(response.access_token.split(".")[1]));
      setState({
        user: {
          email: payload.email || email,
          name: payload.name || "",
          role: payload.role || "viewer",
        },
        isAuthenticated: true,
        loading: false,
      });
      return true;
    } catch {
      return false;
    }
  }, [demo]);

  const register = useCallback(
    async (email: string, password: string, name: string): Promise<boolean | "verify_email"> => {
      if (demo) return true;
      try {
        const response = await auth.register({ email, password, name });

        // If the response has a message (no tokens), email verification is required
        if (response.message && !response.access_token) {
          return "verify_email";
        }

        // OAuth registration returns tokens directly
        if (response.access_token) {
          const payload = JSON.parse(atob(response.access_token.split(".")[1]));
          setState({
            user: {
              email: payload.email || email,
              name: payload.name || name,
              role: payload.role || "viewer",
            },
            isAuthenticated: true,
            loading: false,
          });
          return true;
        }

        return false;
      } catch {
        return false;
      }
    },
    [demo]
  );

  const logout = useCallback(() => {
    if (demo) return;
    auth.logout();
    setState({ user: null, isAuthenticated: false, loading: false });
  }, [demo]);

  return { ...state, login, register, logout };
}
