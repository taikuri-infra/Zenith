"use client";

import { createContext, useContext } from "react";
import { isDemoMode } from "@/lib/get-api";

export const ProjectContext = createContext<string>("");

export function useProject(): string {
  const projectId = useContext(ProjectContext);

  if (isDemoMode()) return "demo-project";

  if (projectId) return projectId;

  // Fallback: read from localStorage or return empty string
  if (typeof window !== "undefined") {
    return localStorage.getItem("currentProjectId") || "";
  }
  return "";
}
