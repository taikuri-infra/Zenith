"use client";

import { useCallback, useEffect, useState } from "react";
import { auth, isAuthenticated, getAccessToken, clearTokens } from "@/lib/api";

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
 */
export function useAuth(): AuthState & {
  login: (email: string, password: string) => Promise<boolean>;
  register: (email: string, password: string, name: string) => Promise<boolean>;
  logout: () => void;
} {
  const [state, setState] = useState<AuthState>({
    user: null,
    isAuthenticated: false,
    loading: true,
  });

  useEffect(() => {
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
  }, []);

  const login = useCallback(async (email: string, password: string) => {
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
  }, []);

  const register = useCallback(
    async (email: string, password: string, name: string) => {
      try {
        const response = await auth.register({ email, password, name });
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
      } catch {
        return false;
      }
    },
    []
  );

  const logout = useCallback(() => {
    auth.logout();
    setState({ user: null, isAuthenticated: false, loading: false });
  }, []);

  return { ...state, login, register, logout };
}
