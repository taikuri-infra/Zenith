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
  customerCount?: number;
  activeCustomers?: number;
  mrr?: string;
  newCustomersThisMonth?: number;
}

// ---------- Plans & Customers ----------

export interface Plan {
  id: string;
  name: string;
  cpuCores: number;
  ramGb: number;
  s3Tb: number;
  dbStorageGb: number;
  volumeGb: number;
  lbCount: number;
  priceCents: number;
  currency: string;
  billingCycle: string;
  active: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface Customer {
  id: string;
  name: string;
  domain: string;
  planId: string;
  contactEmail: string;
  contactName: string;
  status: "active" | "suspended";
  clusterStatus: "pending" | "provisioning" | "installing" | "running" | "error" | "deleting";
  capiClusterName: string;
  clusterRegion: string;
  clusterNodes: number;
  clusterK8sVersion: string;
  notes: string;
  createdAt: string;
  updatedAt: string;
  plan?: Plan;
}

export interface CreateCustomerInput {
  name: string;
  domain: string;
  planId: string;
  contactEmail: string;
  contactName: string;
}

export interface UpdateCustomerInput {
  name?: string;
  domain?: string;
  planId?: string;
  contactEmail?: string;
  contactName?: string;
  notes?: string;
}

export interface CreatePlanInput {
  name: string;
  cpuCores: number;
  ramGb: number;
  s3Tb?: number;
  dbStorageGb?: number;
  volumeGb?: number;
  lbCount?: number;
  priceCents: number;
  currency?: string;
  billingCycle?: string;
}

export interface UpdatePlanInput {
  name?: string;
  cpuCores?: number;
  ramGb?: number;
  s3Tb?: number;
  dbStorageGb?: number;
  volumeGb?: number;
  lbCount?: number;
  priceCents?: number;
  currency?: string;
  billingCycle?: string;
  active?: boolean;
}

export interface CustomerStats {
  totalCustomers: number;
  activeCustomers: number;
  mrr: string;
  newThisMonth: number;
}

// ---------- Metering ----------

export interface CustomerUsage {
  cpuCores: number;
  cpuCeiling: number;
  cpuPercent: number;
  ramGb: number;
  ramCeiling: number;
  ramPercent: number;
  s3Tb: number;
  s3Ceiling: number;
  s3Percent: number;
  dbStorageGb: number;
  dbCeiling: number;
  dbPercent: number;
  volumeGb: number;
  volCeiling: number;
  volPercent: number;
  lbCount: number;
  lbCeiling: number;
  lbPercent: number;
  recordedAt: string;
}

export interface UsageHistoryEntry {
  date: string;
  cpuAvg: number;
  cpuMax: number;
  ramAvg: number;
  ramMax: number;
  dbStorageGb: number;
  volumeGb: number;
  lbCount: number;
}

export interface PlatformUsageSummary {
  totalCpu: number;
  totalRam: number;
  totalStorage: number;
  customersReporting: number;
}

// ---------- Token helpers ----------

const ACCESS_TOKEN_KEY = "mc_token";
const REFRESH_TOKEN_KEY = "mc_refresh_token";

export function getAccessToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(ACCESS_TOKEN_KEY);
}

export function getRefreshToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(REFRESH_TOKEN_KEY);
}

export function setTokens(accessToken: string, refreshToken: string): void {
  if (typeof window === "undefined") return;
  localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
  localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
}

export function clearTokens(): void {
  if (typeof window === "undefined") return;
  localStorage.removeItem(ACCESS_TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
}

export function isAuthenticated(): boolean {
  return !!getAccessToken();
}

// ---------- Auth response type ----------

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
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
      localStorage.setItem(ACCESS_TOKEN_KEY, token);
    }
  }

  clearToken() {
    this.token = null;
    clearTokens();
  }

  getToken(): string | null {
    if (this.token) return this.token;
    if (typeof window !== "undefined") {
      return localStorage.getItem(ACCESS_TOKEN_KEY);
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

  // Auth
  auth = {
    login: async (email: string, password: string): Promise<LoginResponse> => {
      const res = await this.request<LoginResponse>("/api/v1/auth/login", {
        method: "POST",
        body: JSON.stringify({ email, password }),
      });
      setTokens(res.access_token, res.refresh_token);
      this.token = res.access_token;
      return res;
    },
    logout: () => {
      this.clearToken();
      if (typeof window !== "undefined") {
        window.location.href = "/login";
      }
    },
    refresh: async (): Promise<boolean> => {
      const refreshToken = getRefreshToken();
      if (!refreshToken) return false;
      try {
        const res = await this.request<LoginResponse>("/api/v1/auth/refresh", {
          method: "POST",
          body: JSON.stringify({ refresh_token: refreshToken }),
        });
        setTokens(res.access_token, res.refresh_token);
        this.token = res.access_token;
        return true;
      } catch {
        return false;
      }
    },
  };

  // Dashboard
  dashboard = {
    stats: () => this.request<DashboardStats>("/api/v1/admin/dashboard/stats"),
    usage: () => this.request<PlatformUsageSummary>("/api/v1/admin/dashboard/usage"),
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

  // Customers
  customers = {
    list: () => this.request<Customer[]>("/api/v1/admin/customers"),
    get: (id: string) =>
      this.request<Customer>(`/api/v1/admin/customers/${encodeURIComponent(id)}`),
    create: (data: CreateCustomerInput) =>
      this.request<Customer>("/api/v1/admin/customers", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    update: (id: string, data: UpdateCustomerInput) =>
      this.request<Customer>(`/api/v1/admin/customers/${encodeURIComponent(id)}`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    delete: (id: string) =>
      this.request<void>(`/api/v1/admin/customers/${encodeURIComponent(id)}`, {
        method: "DELETE",
      }),
    suspend: (id: string) =>
      this.request<Customer>(
        `/api/v1/admin/customers/${encodeURIComponent(id)}/suspend`,
        { method: "POST" }
      ),
    activate: (id: string) =>
      this.request<Customer>(
        `/api/v1/admin/customers/${encodeURIComponent(id)}/activate`,
        { method: "POST" }
      ),
    stats: () => this.request<CustomerStats>("/api/v1/admin/customers/stats"),
    getCluster: (id: string) =>
      this.request<Cluster>(`/api/v1/admin/customers/${encodeURIComponent(id)}/cluster`),
    scaleCluster: (id: string, nodes: number) =>
      this.request<void>(
        `/api/v1/admin/customers/${encodeURIComponent(id)}/cluster/scale`,
        { method: "POST", body: JSON.stringify({ nodes }) }
      ),
    upgradeCluster: (id: string, version: string) =>
      this.request<void>(
        `/api/v1/admin/customers/${encodeURIComponent(id)}/cluster/upgrade`,
        { method: "POST", body: JSON.stringify({ version }) }
      ),
    usage: (id: string) =>
      this.request<CustomerUsage>(
        `/api/v1/admin/customers/${encodeURIComponent(id)}/usage`
      ),
    usageHistory: (id: string, days = 30) =>
      this.request<UsageHistoryEntry[]>(
        `/api/v1/admin/customers/${encodeURIComponent(id)}/usage/history?days=${days}`
      ),
  };

  // Plans
  plans = {
    list: () => this.request<Plan[]>("/api/v1/admin/plans"),
    create: (data: CreatePlanInput) =>
      this.request<Plan>("/api/v1/admin/plans", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    update: (id: string, data: UpdatePlanInput) =>
      this.request<Plan>(`/api/v1/admin/plans/${encodeURIComponent(id)}`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
  };
}

// Export singleton
export const api = new ApiClient(API_BASE_URL);
