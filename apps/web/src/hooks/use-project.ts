"use client";

import { createContext, useCallback, useContext, useEffect, useState } from "react";
import type { Project } from "@/lib/api";
import { getApi, isDemoMode } from "@/lib/get-api";

interface ProjectContextValue {
  currentProject: Project | null;
  projects: Project[];
  setCurrentProject: (project: Project) => void;
  createProject: (name: string, description?: string) => Promise<Project>;
  deleteProject: (id: string) => Promise<void>;
  loading: boolean;
}

const defaultValue: ProjectContextValue = {
  currentProject: null,
  projects: [],
  setCurrentProject: () => {},
  createProject: async () => {
    throw new Error("ProjectProvider not mounted");
  },
  deleteProject: async () => {
    throw new Error("ProjectProvider not mounted");
  },
  loading: true,
};

export const ProjectContext = createContext<ProjectContextValue>(defaultValue);

export function useProjectContext(): ProjectContextValue {
  return useContext(ProjectContext);
}

/** Backward-compatible hook that returns the current project ID string. */
export function useProject(): string {
  const { currentProject } = useProjectContext();
  return currentProject?.id || "";
}

/** Hook for managing project state — used by ProjectProvider. */
export function useProjectState(): ProjectContextValue {
  const [projects, setProjects] = useState<Project[]>([]);
  const [currentProject, setCurrentProjectState] = useState<Project | null>(null);
  const [loading, setLoading] = useState(true);

  // Load projects from API on mount
  useEffect(() => {
    const api = getApi();
    api.projects
      .list()
      .then((res) => {
        const items = res.items || [];
        setProjects(items);

        // Restore previously selected project from localStorage
        const savedId =
          typeof window !== "undefined"
            ? localStorage.getItem("currentProjectId")
            : null;
        const saved = savedId ? items.find((p) => p.id === savedId) : null;
        setCurrentProjectState(saved || items[0] || null);
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  const setCurrentProject = useCallback((project: Project) => {
    setCurrentProjectState(project);
    if (typeof window !== "undefined") {
      localStorage.setItem("currentProjectId", project.id);
    }
  }, []);

  const createProject = useCallback(
    async (name: string, description?: string): Promise<Project> => {
      if (isDemoMode()) throw new Error("Not available in demo mode");
      const api = getApi();
      const created = await api.projects.create({ name, description });
      setProjects((prev) => [...prev, created]);
      return created;
    },
    []
  );

  const deleteProject = useCallback(
    async (id: string): Promise<void> => {
      if (isDemoMode()) throw new Error("Not available in demo mode");
      const api = getApi();
      await api.projects.delete(id);
      setProjects((prev) => {
        const remaining = prev.filter((p) => p.id !== id);
        // Switch to another project if we deleted the current one
        if (currentProject?.id === id && remaining.length > 0) {
          setCurrentProjectState(remaining[0]);
          if (typeof window !== "undefined") {
            localStorage.setItem("currentProjectId", remaining[0].id);
          }
        }
        return remaining;
      });
    },
    [currentProject]
  );

  return { currentProject, projects, setCurrentProject, createProject, deleteProject, loading };
}
