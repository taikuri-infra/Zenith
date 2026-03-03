"use client";

import { createContext, useContext, useEffect, useState } from "react";
import { isDemoMode } from "@/lib/get-api";

export const ProjectContext = createContext<string>("");

export function useProject(): string {
  const contextId = useContext(ProjectContext);
  const [resolvedId, setResolvedId] = useState<string>(() => {
    if (isDemoMode()) return "demo-project";
    if (contextId) return contextId;
    if (typeof window !== "undefined") {
      return localStorage.getItem("currentProjectId") || "";
    }
    return "";
  });

  useEffect(() => {
    if (isDemoMode() || contextId || resolvedId) return;

    // Auto-fetch first project from API and persist it
    const token =
      typeof window !== "undefined"
        ? localStorage.getItem("access_token")
        : null;
    if (!token) return;

    const apiUrl =
      process.env.NEXT_PUBLIC_API_URL || "https://api.freezenith.com";
    fetch(`${apiUrl}/api/v1/projects`, {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        const first = data?.items?.[0];
        if (first?.name) {
          localStorage.setItem("currentProjectId", first.name);
          setResolvedId(first.name);
        }
      })
      .catch(() => {});
  }, [contextId, resolvedId]);

  if (isDemoMode()) return "demo-project";
  if (contextId) return contextId;
  return resolvedId;
}
