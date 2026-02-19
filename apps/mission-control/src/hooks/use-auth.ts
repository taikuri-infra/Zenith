"use client";

import { useCallback, useEffect, useState } from "react";
import { api, isAuthenticated, getAccessToken, clearTokens } from "@/lib/api";
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

export function useAuth(): AuthState & {
  login: (email: string, password: string) => Promise<boolean>;
  logout: () => void;
} {
  const demo = isDemoMode();

  const [state, setState] = useState<AuthState>({
    user: demo
      ? { email: "admin@zenith.dev", name: "Admin", role: "owner" }
      : null,
    isAuthenticated: demo,
    loading: !demo,
  });

  useEffect(() => {
    if (demo) return;

    const authenticated = isAuthenticated();
    if (authenticated) {
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
          clearTokens();
        }
      }
    }
    setState({ user: null, isAuthenticated: false, loading: false });
  }, [demo]);

  const login = useCallback(
    async (email: string, password: string) => {
      if (demo) return true;
      try {
        const response = await api.auth.login(email, password);
        const payload = JSON.parse(
          atob(response.access_token.split(".")[1])
        );
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
    },
    [demo]
  );

  const logout = useCallback(() => {
    if (demo) return;
    api.auth.logout();
    setState({ user: null, isAuthenticated: false, loading: false });
  }, [demo]);

  return { ...state, login, logout };
}
