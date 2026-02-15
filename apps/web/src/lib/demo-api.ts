/**
 * Demo API client that returns mock data with a realistic delay.
 * Mirrors the shape of the real api.ts exports so pages can use it as a drop-in.
 */

import type {
  LoginResponse,
  Project,
  App,
  Database,
  StorageBucket,
} from "./api";

import {
  mockApps,
  mockDatabases,
  mockStorage,
} from "./mock-data";

// Simulate a short network delay so skeleton states flash briefly
const delay = (ms = 300) => new Promise<void>((r) => setTimeout(r, ms));

// Map mock-data App → api.ts App shape
function toApiApp(m: (typeof mockApps)[number]): App {
  return {
    name: m.name,
    image: m.source,
    replicas: m.replicas.ready,
    port: m.port,
    status: m.status,
    cpu: m.cpu,
    memory: m.memory,
    domain: m.domain,
    created_at: m.lastDeploy,
  };
}

// Map mock-data Database → api.ts Database shape
function toApiDatabase(m: (typeof mockDatabases)[number]): Database {
  const portMap: Record<string, number> = {
    postgresql: 5432,
    mysql: 3306,
    mongodb: 27017,
    redis: 6379,
  };
  return {
    name: m.name,
    engine: m.engine,
    version: m.version,
    storage: m.storageTotal,
    status: m.status,
    connection_string: `${m.engine}://${m.name}.internal:${portMap[m.engine] ?? 5432}/${m.name}`,
    port: portMap[m.engine] ?? 5432,
    created_at: m.lastBackup ?? "1 week ago",
  };
}

// Map mock-data StorageBucket → api.ts StorageBucket shape
function toApiStorage(m: (typeof mockStorage)[number]): StorageBucket {
  return {
    name: m.name,
    access: "private",
    region: "fsn1",
    size: m.used,
    objects: m.objects,
    status: m.status === "active" ? "active" : "creating",
    created_at: "2 weeks ago",
  };
}

const demoProject: Project = {
  id: "demo-project",
  name: "demo-project",
  display_name: "My Startup",
  owner: "demo@zenith.dev",
  plan: "Starter",
  region: "fsn1",
  status: "active",
  created_at: "2026-01-15T00:00:00Z",
};

export const demoAuth = {
  async login(): Promise<LoginResponse> {
    await delay();
    return {
      access_token: "demo.eyJlbWFpbCI6ImRlbW9AemVuaXRoLmRldiIsIm5hbWUiOiJEZW1vIFVzZXIiLCJyb2xlIjoiYWRtaW4ifQ.demo",
      refresh_token: "demo-refresh-token",
      token_type: "Bearer",
      expires_in: 86400,
    };
  },

  async register(): Promise<LoginResponse> {
    throw new Error("Not available in demo mode");
  },

  logout(): void {
    // no-op in demo mode
  },

  getOAuthUrl(): string {
    return "#";
  },
};

export const demoProjects = {
  list: async (): Promise<{ items: Project[] }> => {
    await delay();
    return { items: [demoProject] };
  },
  get: async (): Promise<Project> => {
    await delay();
    return demoProject;
  },
  create: async (): Promise<Project> => {
    throw new Error("Not available in demo mode");
  },
  update: async (): Promise<Project> => {
    throw new Error("Not available in demo mode");
  },
  delete: async (): Promise<void> => {
    throw new Error("Not available in demo mode");
  },
};

export const demoApps = {
  list: async (): Promise<{ items: App[] }> => {
    await delay();
    return { items: mockApps.map(toApiApp) };
  },
  get: async (_projectId: string, name: string): Promise<App> => {
    await delay();
    const app = mockApps.find((a) => a.name === name);
    if (!app) throw new Error(`App "${name}" not found`);
    return toApiApp(app);
  },
  create: async (): Promise<App> => {
    throw new Error("Not available in demo mode");
  },
  update: async (): Promise<App> => {
    throw new Error("Not available in demo mode");
  },
  delete: async (): Promise<void> => {
    throw new Error("Not available in demo mode");
  },
  redeploy: async (): Promise<void> => {
    throw new Error("Not available in demo mode");
  },
};

export const demoDatabases = {
  list: async (): Promise<{ items: Database[] }> => {
    await delay();
    return { items: mockDatabases.map(toApiDatabase) };
  },
  get: async (_projectId: string, name: string): Promise<Database> => {
    await delay();
    const db = mockDatabases.find((d) => d.name === name);
    if (!db) throw new Error(`Database "${name}" not found`);
    return toApiDatabase(db);
  },
  create: async (): Promise<Database> => {
    throw new Error("Not available in demo mode");
  },
  delete: async (): Promise<void> => {
    throw new Error("Not available in demo mode");
  },
};

export const demoStorage = {
  list: async (): Promise<{ items: StorageBucket[] }> => {
    await delay();
    return { items: mockStorage.map(toApiStorage) };
  },
  create: async (): Promise<StorageBucket> => {
    throw new Error("Not available in demo mode");
  },
  delete: async (): Promise<void> => {
    throw new Error("Not available in demo mode");
  },
};

// Re-export as a unified object matching the real API import pattern
export const demoApi = {
  auth: demoAuth,
  projects: demoProjects,
  apps: demoApps,
  databases: demoDatabases,
  storage: demoStorage,
};
