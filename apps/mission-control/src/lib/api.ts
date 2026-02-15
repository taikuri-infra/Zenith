const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

// ---------- Error ----------

export class ApiError extends Error {
  status: number;
  body: string;

  constructor(status: number, body: string) {
    super(`API error ${status}: ${body}`);
    this.name = "ApiError";
    this.status = status;
    this.body = body;
  }
}

// ---------- Types ----------

export interface Cluster {
  name: string;
  k8sVersion: string;
  nodes: number;
  region: string;
  type: "shared" | "dedicated";
  tenant?: string;
  cpuPercent: number;
  ramPercent: number;
  pods: { used: number; total: number };
  pvcs: { used: number; total: number };
  status: "healthy" | "warning" | "error";
  upgradeAvailable?: string;
}

export interface CreateClusterInput {
  name: string;
  region: string;
  type: "shared" | "dedicated";
  tenant?: string;
  nodes: number;
  k8sVersion: string;
}

export interface Module {
  name: string;
  installed: string;
  latest: string;
  status: "up_to_date" | "update_available";
  description: string;
}

export interface Tenant {
  name: string;
  plan: "starter" | "pro";
  apps: number;
  databases: number;
  cpuUsed: string;
  cpuLimit: string;
  ramUsed: string;
  ramLimit: string;
  status: "active" | "idle" | "suspended";
}

export interface AuditEntry {
  time: string;
  actor: string;
  action: string;
  cluster?: string;
}

export interface PlatformUpdate {
  version: string;
  current: string;
  releasedAt: string;
  features: string[];
  breakingChanges: boolean;
}

export interface UpdateHistoryEntry {
  version: string;
  date: string;
  status: "installed" | "superseded";
}

export interface InfraNode {
  name: string;
  type: string;
  count: number;
  cluster: string;
  monthlyCost: string;
}

export interface InfraOverview {
  servers: number;
  volumes: number;
  volumeSize: string;
  loadBalancers: number;
  lbPublic: number;
  lbInternal: number;
  monthlyCost: string;
  resources: InfraNode[];
}

export interface PlatformState {
  platformVersion: string;
  updateAvailable?: string;
  installedDate: string;
  installedDaysAgo: number;
  managementK8sVersion: string;
  managementK8sUpToDate: boolean;
  domain: string;
  wildcardTls: boolean;
}

export interface PlatformSettings {
  platformName: string;
  baseDomain: string;
  provider: string;
  defaultRegion: string;
  regionLabel: string;
  autoBackups: boolean;
  retentionDays: number;
}

export interface DashboardStats {
  clusterCount: number;
  allHealthy: boolean;
  tenantCount: number;
  activeToday: number;
  monthlyCost: string;
  costProvider: string;
  updatesAvailable: number;
}

// ---------- API Client ----------

class ApiClient {
  private baseUrl: string;
  private token: string | null = null;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  setToken(token: string) {
    this.token = token;
    if (typeof window !== "undefined") {
      localStorage.setItem("mc_token", token);
    }
  }

  clearToken() {
    this.token = null;
    if (typeof window !== "undefined") {
      localStorage.removeItem("mc_token");
    }
  }

  getToken(): string | null {
    if (this.token) return this.token;
    if (typeof window !== "undefined") {
      return localStorage.getItem("mc_token");
    }
    return null;
  }

  private async request<T>(path: string, options?: RequestInit): Promise<T> {
    const token = this.getToken();
    const res = await fetch(`${this.baseUrl}${path}`, {
      ...options,
      headers: {
        "Content-Type": "application/json",
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...options?.headers,
      },
    });
    if (!res.ok) {
      const body = await res.text();
      throw new ApiError(res.status, body);
    }
    // Handle empty responses (204 No Content, etc.)
    const text = await res.text();
    if (!text) return undefined as T;
    return JSON.parse(text) as T;
  }

  // Dashboard
  dashboard = {
    stats: () => this.request<DashboardStats>("/api/v1/admin/dashboard/stats"),
  };

  // Clusters
  clusters = {
    list: () => this.request<Cluster[]>("/api/v1/admin/clusters"),
    get: (name: string) =>
      this.request<Cluster>(`/api/v1/admin/clusters/${encodeURIComponent(name)}`),
    create: (data: CreateClusterInput) =>
      this.request<Cluster>("/api/v1/admin/clusters", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    delete: (name: string) =>
      this.request<void>(`/api/v1/admin/clusters/${encodeURIComponent(name)}`, {
        method: "DELETE",
      }),
    upgrade: (name: string, version: string) =>
      this.request<void>(
        `/api/v1/admin/clusters/${encodeURIComponent(name)}/upgrade`,
        { method: "POST", body: JSON.stringify({ version }) }
      ),
  };

  // Tenants
  tenants = {
    list: () => this.request<Tenant[]>("/api/v1/admin/tenants"),
    get: (id: string) =>
      this.request<Tenant>(`/api/v1/admin/tenants/${encodeURIComponent(id)}`),
    suspend: (id: string) =>
      this.request<void>(
        `/api/v1/admin/tenants/${encodeURIComponent(id)}/suspend`,
        { method: "POST" }
      ),
  };

  // Modules
  modules = {
    list: () => this.request<Module[]>("/api/v1/admin/modules"),
    install: (name: string) =>
      this.request<void>(
        `/api/v1/admin/modules/${encodeURIComponent(name)}/install`,
        { method: "POST" }
      ),
    uninstall: (name: string) =>
      this.request<void>(
        `/api/v1/admin/modules/${encodeURIComponent(name)}/uninstall`,
        { method: "POST" }
      ),
    update: (name: string) =>
      this.request<void>(
        `/api/v1/admin/modules/${encodeURIComponent(name)}/update`,
        { method: "POST" }
      ),
    updateAll: () =>
      this.request<void>("/api/v1/admin/modules/update-all", {
        method: "POST",
      }),
  };

  // Audit Log
  audit = {
    list: (params?: { limit?: number; offset?: number; actor?: string; cluster?: string; period?: string }) => {
      const query = new URLSearchParams();
      if (params?.limit) query.set("limit", String(params.limit));
      if (params?.offset) query.set("offset", String(params.offset));
      if (params?.actor) query.set("actor", params.actor);
      if (params?.cluster) query.set("cluster", params.cluster);
      if (params?.period) query.set("period", params.period);
      const qs = query.toString();
      return this.request<AuditEntry[]>(`/api/v1/admin/audit${qs ? `?${qs}` : ""}`);
    },
  };

  // Platform updates
  updates = {
    check: () => this.request<PlatformUpdate>("/api/v1/admin/updates/check"),
    apply: (version: string) =>
      this.request<void>("/api/v1/admin/updates/apply", {
        method: "POST",
        body: JSON.stringify({ version }),
      }),
    history: () =>
      this.request<UpdateHistoryEntry[]>("/api/v1/admin/updates/history"),
  };

  // Infrastructure
  infrastructure = {
    overview: () => this.request<InfraOverview>("/api/v1/admin/infrastructure"),
  };

  // Platform state
  state = {
    get: () => this.request<PlatformState>("/api/v1/admin/state"),
    export: () => this.request<string>("/api/v1/admin/state/export"),
  };

  // Settings
  settings = {
    get: () => this.request<PlatformSettings>("/api/v1/admin/settings"),
    update: (data: Partial<PlatformSettings>) =>
      this.request<PlatformSettings>("/api/v1/admin/settings", {
        method: "PATCH",
        body: JSON.stringify(data),
      }),
  };
}

// Export singleton
export const api = new ApiClient(API_BASE_URL);
