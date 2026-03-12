import { API_BASE_URL } from "./runtime-env";

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

export interface SupportTicket {
  id: string;
  user_id: string;
  subject: string;
  category: string;
  priority: string;
  status: "open" | "in-progress" | "waiting-on-customer" | "resolved" | "closed";
  assigned_to?: string;
  closed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface SupportMessage {
  id: string;
  ticket_id: string;
  sender_id: string;
  sender_role: "user" | "admin";
  body: string;
  created_at: string;
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

// --- War Room ---

export interface WarRoomData {
  kpis: WarRoomKPIs;
  serviceHealth: ServiceHealthItem[];
  recentAlerts: AlertItem[];
  activeTickets: TicketSummaryItem[];
}

export interface WarRoomKPIs {
  mrr: number;
  mrrTrend: number;
  activeCustomers: number;
  totalCustomers: number;
  newSignups: number;
  churnRate: number;
  avgResponseTime: string;
  healthScore: number;
}

// --- Analytics ---

export interface RevenueStats {
  mrr: number;
  arr: number;
  churnRate: number;
  ltv: number;
  revenueByPlan: { plan: string; revenue: number; count: number }[];
  monthlyTrend: { month: string; newRevenue: number; churnedRevenue: number; totalMrr: number }[];
}

export interface GrowthStats {
  totalUsers: number;
  newThisMonth: number;
  churnedThisMonth: number;
  monthlyGrowth: { month: string; new: number; churned: number; total: number }[];
  conversions: { freeToProRate: number; proToTeamRate: number; trialToPayRate: number };
}

export interface UsageAnalytics {
  topFeatures: { feature: string; usageCount: number; userCount: number }[];
  avgAppsPerUser: number;
  avgDbsPerUser: number;
}

export interface CohortData {
  cohort: string;
  month: string;
  retained: number;
  total: number;
  percentage: number;
}

// --- Surveys ---

export interface SurveyResponse {
  user_id: string;
  created_at: string;
  use_case: string;
  role: string;
  team_size: string;
  company_name: string;
  current_provider: string;
  monthly_spend: string;
  biggest_pain: string;
  expected_traffic: string;
  timeline: string;
  most_important: string;
  stack: string[];
  discovery: string;
}

export interface SurveyInsights {
  total_responses: number;
  responses: SurveyResponse[];
  breakdowns: Record<string, Record<string, number>>;
}

// --- CRM ---

export interface CRMPipeline {
  stages: PipelineStage[];
}

export interface PipelineStage {
  name: string;
  count: number;
  customers: PipelineCustomer[];
}

export interface PipelineCustomer {
  id: string;
  name: string;
  email: string;
  plan: string;
  healthScore: number;
  mrr: number;
  lastLogin?: string;
}

export interface HealthScoreItem {
  userId: string;
  score: number;
  usageScore: number;
  supportScore: number;
  loginScore: number;
  riskLevel: string;
}

export interface CustomerNoteItem {
  id: string;
  userId: string;
  authorId: string;
  authorName?: string;
  note: string;
  tags: string[];
  createdAt: string;
  updatedAt: string;
}

// --- Services ---

export interface ServiceHealthItem {
  name: string;
  namespace: string;
  kind: string;
  status: "healthy" | "degraded" | "down" | "unknown";
  readyReplicas: number;
  totalReplicas: number;
  restarts: number;
  version?: string;
  uptime?: string;
  cpuUsage?: string;
  memUsage?: string;
}

export interface ServiceDetailItem extends ServiceHealthItem {
  pods: { name: string; status: string; node?: string; restarts: number; age: string; cpu?: string; memory?: string }[];
  events: { type: string; reason: string; message: string; time: string }[];
}

// --- Databases (Admin) ---

export interface AdminDatabaseCluster {
  name: string;
  namespace: string;
  status: string;
  instances: number;
  readyInstances: number;
  storageSize: string;
  walArchiving: string;
  lastBackup?: string;
  recoveryWindow?: string;
  postgresVersion?: string;
}

// --- Storage (Admin) ---

export interface AdminS3Bucket {
  name: string;
  size: string;
  objectCount: number;
  lastModified?: string;
}

export interface AdminVolume {
  name: string;
  namespace: string;
  size: string;
  status: string;
  storageClass: string;
}

// --- Networking ---

export interface AdminRoute {
  name: string;
  host: string;
  service: string;
  tls: boolean;
  source: string;
}

export interface AdminCertificate {
  name: string;
  namespace: string;
  dnsNames: string[];
  issuer: string;
  status: string;
  expiresAt?: string;
  renewAt?: string;
}

// --- Observability ---

export interface AlertItem {
  name: string;
  state: string;
  severity: string;
  summary?: string;
  labels?: Record<string, string>;
  activeAt?: string;
}

export interface AlertRule {
  name: string;
  group: string;
  query: string;
  duration: string;
  severity: string;
  state: string;
}

export interface TraceItem {
  traceId: string;
  rootService: string;
  rootName: string;
  duration: string;
  spanCount: number;
  startTime: string;
}

// --- Security ---

export interface SecurityPosture {
  overallScore: number;
  mfaAdoption: number;
  imageVulns: { critical: number; high: number; medium: number; low: number };
  policyViolations: number;
  falcoAlerts: number;
  certWarnings: number;
  failedLogins24h: number;
  openIssues: number;
}

export interface PolicyItem {
  name: string;
  kind: string;
  action: string;
  status: string;
  violations: number;
}

export interface ImageScanItem {
  repository: string;
  tag: string;
  scanStatus: string;
  vulns: { critical: number; high: number; medium: number; low: number };
  lastScanned?: string;
}

export interface SessionItem {
  id: string;
  userId: string;
  email?: string;
  ipAddress: string;
  userAgent?: string;
  device?: string;
  lastSeen: string;
  createdAt: string;
}

// --- Backups ---

export interface BackupStatusData {
  veleroSchedules: { name: string; schedule: string; lastBackup?: string; lastStatus: string; backupCount: number }[];
  cnpgBackups: { cluster: string; namespace: string; lastBackup?: string; status: string; walArchiving: string; retentionDays: number }[];
}

// --- GitOps ---

export interface ArgoAppItem {
  name: string;
  namespace: string;
  status: string;
  health: string;
  syncStatus: string;
  revision?: string;
  lastSync?: string;
  repoUrl?: string;
  path?: string;
}

// --- Registry ---

export interface RegistryProject {
  name: string;
  repoCount: number;
  storageUsed?: string;
  storageQuota?: string;
  public: boolean;
}

export interface RegistryRepo {
  name: string;
  tagCount: number;
  pullCount: number;
  pushTime?: string;
}

// --- Admin RBAC ---

export interface AdminRoleItem {
  id: string;
  userId: string;
  email?: string;
  name?: string;
  adminRole: string;
  permissions: string[];
  grantedBy?: string;
  createdAt: string;
  updatedAt: string;
}

// --- Quality ---

export interface QualityMetrics {
  avgResponseTime: string;
  avgResolutionTime: string;
  openTickets: number;
  resolvedThisWeek: number;
  csat: number;
  slaCompliance: number;
  uptime: number;
  errorRate: number;
  p95Latency: string;
  ticketsByPriority: Record<string, number>;
  ticketsByCategory: Record<string, number>;
  weeklyTrend: { week: string; opened: number; resolved: number }[];
}

// --- Ticket summary ---

export interface TicketSummaryItem {
  id: string;
  subject: string;
  priority: string;
  status: string;
  age: string;
}

// --- Page-specific types (used by Mission Control v2 pages) ---

export interface SecurityOverview {
  score: number;
  mfaAdoption: number;
  vulnerabilities: number;
  policyViolations: number;
  failedLogins: number;
  activeSessions: number;
  activeApiKeys: number;
  certsExpiringSoon: number;
}

export interface CrmPipelineStage {
  name: string;
  count: number;
  customers: CrmCustomerCard[];
}

export interface CrmPipeline {
  stages: CrmPipelineStage[];
}

export interface CrmCustomerCard {
  id: string;
  name: string;
  email: string;
  healthScore: number;
  plan: string;
  mrr: number;
}

export interface VeleroSchedule {
  name: string;
  schedule: string;
  lastBackup?: string;
  lastStatus: string;
  retention: string;
  storageLocation: string;
}

export interface CnpgBackup {
  cluster: string;
  namespace: string;
  schedule: string;
  lastBackup?: string;
  status: string;
  s3Destination: string;
}

export interface BackupStats {
  veleroSchedules: number;
  cnpgClusters: number;
  lastBackup?: string;
  totalSize: string;
}

export interface AdminUser {
  id: string;
  email: string;
  name: string;
  role: string;
  mfaEnabled: boolean;
  permissionCount: number;
  grantedBy?: string;
}

export interface Alert {
  name: string;
  state: string;
  severity: string;
  summary: string;
  activeSince: string;
}

export interface AlertStats {
  firing: number;
  pending: number;
  resolvedToday: number;
  totalRules: number;
}

export interface Trace {
  traceId: string;
  service: string;
  operationName: string;
  durationMs: number;
  spanCount: number;
}

export interface Route {
  name: string;
  host: string;
  service: string;
  port: number;
  tls: boolean;
  namespace: string;
}

export interface Certificate {
  name: string;
  domains: string[];
  issuer: string;
  ready: boolean;
  expiresAt: string;
}

export interface DnsRecord {
  name: string;
  type: string;
  value: string;
  ttl: number;
  managedBy: string;
}

export interface S3Bucket {
  name: string;
  region: string;
  size: string;
  objectCount: number;
  createdAt: string;
}

export interface PvcVolume {
  name: string;
  namespace: string;
  status: string;
  capacity: string;
  storageClass: string;
  boundTo?: string;
}

export interface StorageStats {
  totalBuckets: number;
  s3Used: string;
  totalVolumes: number;
  pvcUsed: string;
}

export interface DatabaseCluster {
  name: string;
  namespace: string;
  status: string;
  readyInstances: number;
  totalInstances: number;
  storage: string;
  walArchiving: boolean;
  lastBackup?: string;
  pgVersion: string;
}

export interface DatabaseStats {
  totalClusters: number;
  healthyClusters: number;
  totalStorage: string;
  lastBackup?: string;
}

export interface ArgoApp {
  name: string;
  namespace: string;
  project: string;
  healthStatus: string;
  syncStatus: string;
  revision?: string;
  lastSynced?: string;
}

export interface GitOpsStats {
  totalApps: number;
  synced: number;
  outOfSync: number;
  degraded: number;
}

export interface HarborProject {
  name: string;
  repoCount: number;
  storageUsed: number;
  storageQuota: number;
  storageUsedDisplay: string;
  storageQuotaDisplay: string;
  access: string;
  public: boolean;
  createdAt: string;
}

export interface RegistryStats {
  totalProjects: number;
  totalRepos: number;
  totalTags: number;
  storageUsed: string;
  storageQuota: string;
}

export interface KyvernoPolicy {
  name: string;
  kind: string;
  action: string;
  ready: boolean;
  violations: number;
  updatedAt: string;
}

export interface WafStats {
  totalPolicies: number;
  enforcing: number;
  auditing: number;
  totalViolations: number;
}

export interface ImageScanResult {
  repository: string;
  tag: string;
  scanStatus: string;
  critical: number;
  high: number;
  medium: number;
  low: number;
}

export interface ImageScanStats {
  totalImages: number;
  cleanImages: number;
  criticalCount: number;
  highCount: number;
}

export interface ActiveSession {
  id: string;
  email: string;
  ipAddress: string;
  device: string;
  location: string;
  lastSeen: string;
  mfaEnabled: boolean;
  isAdmin: boolean;
}

export interface QualityTicket {
  id: string;
  subject: string;
  customer: string;
  priority: string;
  status: string;
  age: string;
}

export interface LogQueryResult {
  lines: { timestamp: string; labels: string; message: string }[];
  executionTimeMs: number;
}

export interface GrafanaDashboard {
  uid: string;
  title: string;
  description?: string;
  url: string;
  category?: string;
  starred?: boolean;
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
    exchangeOAuthCode: async (code: string): Promise<LoginResponse> => {
      const res = await this.request<LoginResponse>("/api/v1/auth/exchange", {
        method: "POST",
        body: JSON.stringify({ code }),
      });
      setTokens(res.access_token, res.refresh_token);
      this.token = res.access_token;
      return res;
    },
    getGoogleOAuthUrl: (): string => {
      const mcOrigin = typeof window !== "undefined" ? window.location.origin : "";
      return `${API_BASE_URL}/api/v1/auth/oauth/google?redirect=${encodeURIComponent(mcOrigin)}`;
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

  // Support Tickets (admin)
  support = {
    list: (params?: { status?: string; limit?: number; offset?: number }) => {
      const query = new URLSearchParams();
      if (params?.status) query.set("status", params.status);
      if (params?.limit) query.set("limit", String(params.limit));
      if (params?.offset) query.set("offset", String(params.offset));
      const qs = query.toString();
      return this.request<{ items: SupportTicket[]; total: number }>(
        `/api/v1/admin/support/tickets${qs ? `?${qs}` : ""}`
      );
    },
    get: (id: string) =>
      this.request<{ ticket: SupportTicket; messages: SupportMessage[] }>(
        `/api/v1/admin/support/tickets/${id}`
      ),
    reply: (id: string, body: string) =>
      this.request<SupportMessage>(
        `/api/v1/admin/support/tickets/${id}/reply`,
        { method: "POST", body: JSON.stringify({ body }) }
      ),
    updateStatus: (id: string, status: string) =>
      this.request<void>(
        `/api/v1/admin/support/tickets/${id}/status`,
        { method: "PUT", body: JSON.stringify({ status }) }
      ),
    assign: (id: string, adminUserId: string) =>
      this.request<void>(
        `/api/v1/admin/support/tickets/${id}/assign`,
        { method: "PUT", body: JSON.stringify({ admin_user_id: adminUserId }) }
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

  // War Room
  warRoom = {
    get: () => this.request<WarRoomData>("/api/v1/admin/war-room"),
  };

  // Analytics
  analytics = {
    revenue: () => this.request<RevenueStats>("/api/v1/admin/analytics/revenue"),
    growth: () => this.request<GrowthStats>("/api/v1/admin/analytics/growth"),
    usage: () => this.request<UsageAnalytics>("/api/v1/admin/analytics/usage"),
    cohorts: () => this.request<CohortData[]>("/api/v1/admin/analytics/cohorts"),
  };

  // Survey Insights
  surveys = {
    insights: () => this.request<SurveyInsights>("/api/v1/admin/surveys"),
  };

  // CRM
  crm = {
    pipeline: () => this.request<CrmPipeline>("/api/v1/admin/crm/pipeline"),
    healthScores: () => this.request<HealthScoreItem[]>("/api/v1/admin/crm/health-scores"),
    getNotes: (userId: string) =>
      this.request<CustomerNoteItem[]>(`/api/v1/admin/crm/customers/${userId}/notes`),
    saveNote: (userId: string, note: string, tags: string[]) =>
      this.request<void>(`/api/v1/admin/crm/customers/${userId}/notes`, {
        method: "PUT",
        body: JSON.stringify({ note, tags }),
      }),
    updateTags: (userId: string, tags: string[]) =>
      this.request<void>(`/api/v1/admin/crm/customers/${userId}/tags`, {
        method: "PUT",
        body: JSON.stringify({ tags }),
      }),
  };

  // Services
  services = {
    list: () => this.request<ServiceHealthItem[]>("/api/v1/admin/services"),
    get: (name: string) =>
      this.request<ServiceDetailItem>(`/api/v1/admin/services/${encodeURIComponent(name)}`),
    restart: (name: string) =>
      this.request<void>(`/api/v1/admin/services/${encodeURIComponent(name)}/restart`, {
        method: "POST",
      }),
  };

  // Databases (Admin) — alias for backwards compat
  adminDatabases = {
    list: () => this.request<DatabaseCluster[]>("/api/v1/admin/databases"),
    get: (name: string, namespace?: string) =>
      this.request<DatabaseCluster>(
        `/api/v1/admin/databases/${encodeURIComponent(name)}${namespace ? `?namespace=${namespace}` : ""}`
      ),
  };

  // Storage (Admin) — alias for backwards compat
  adminStorage = {
    s3: () => this.request<S3Bucket[]>("/api/v1/admin/storage/s3"),
    volumes: () => this.request<PvcVolume[]>("/api/v1/admin/storage/volumes"),
  };

  // Networking
  networking = {
    dns: () => this.request<DnsRecord[]>("/api/v1/admin/networking/dns"),
    dnsRecords: () => this.request<DnsRecord[]>("/api/v1/admin/networking/dns"),
    routes: () => this.request<Route[]>("/api/v1/admin/networking/routes"),
    certificates: () => this.request<Certificate[]>("/api/v1/admin/networking/certificates"),
  };

  // Observability
  observability = {
    dashboards: () => this.request<GrafanaDashboard[]>("/api/v1/admin/observability/dashboards"),
    queryLogs: (query: string, limit?: number) =>
      this.request<LogQueryResult>("/api/v1/admin/observability/logs/query", {
        method: "POST",
        body: JSON.stringify({ query, limit: limit || 100 }),
      }),
    logLabels: () => this.request<string[]>("/api/v1/admin/observability/logs/labels"),
    alerts: () => this.request<Alert[]>("/api/v1/admin/observability/alerts"),
    alertRules: () => this.request<AlertRule[]>("/api/v1/admin/observability/alerts/rules"),
    traces: (service?: string, limit?: number) =>
      this.request<Trace[]>(
        `/api/v1/admin/observability/traces?${service ? `service=${service}&` : ""}limit=${limit || 20}`
      ),
    getTrace: (id: string) => this.request<unknown>(`/api/v1/admin/observability/traces/${id}`),
    createSilence: (data: unknown) =>
      this.request<void>("/api/v1/admin/observability/alerts/silence", {
        method: "POST",
        body: JSON.stringify(data),
      }),
  };

  // Security
  security = {
    posture: () => this.request<SecurityPosture>("/api/v1/admin/security/posture"),
    overview: () => this.request<SecurityOverview>("/api/v1/admin/security/posture"),
    policies: () => this.request<KyvernoPolicy[]>("/api/v1/admin/security/policies"),
    wafStats: () => this.request<WafStats>("/api/v1/admin/security/policies/stats"),
    falcoAlerts: () => this.request<unknown[]>("/api/v1/admin/security/falco/alerts"),
    rateLimits: () => this.request<unknown>("/api/v1/admin/security/rate-limits"),
    images: () => this.request<ImageScanResult[]>("/api/v1/admin/security/images"),
    imageStats: () => this.request<ImageScanStats>("/api/v1/admin/security/images/stats"),
    triggerScan: (name: string) =>
      this.request<void>(`/api/v1/admin/security/images/${encodeURIComponent(name)}/scan`, {
        method: "POST",
      }),
    sessions: () => this.request<ActiveSession[]>("/api/v1/admin/security/sessions"),
    terminateSession: (id: string) =>
      this.request<void>(`/api/v1/admin/security/sessions/${id}`, { method: "DELETE" }),
  };

  // Backups
  backups = {
    list: () => this.request<BackupStatusData>("/api/v1/admin/backups"),
    stats: () => this.request<BackupStats>("/api/v1/admin/backups/stats"),
    veleroSchedules: () => this.request<VeleroSchedule[]>("/api/v1/admin/backups/velero"),
    cnpgBackups: () => this.request<CnpgBackup[]>("/api/v1/admin/backups/cnpg"),
    trigger: () =>
      this.request<void>("/api/v1/admin/backups/trigger", { method: "POST" }),
  };

  // GitOps
  gitops = {
    apps: () => this.request<ArgoAppItem[]>("/api/v1/admin/gitops/apps"),
    list: () => this.request<ArgoApp[]>("/api/v1/admin/gitops/apps"),
    stats: () => this.request<GitOpsStats>("/api/v1/admin/gitops/stats"),
    sync: (name: string) =>
      this.request<void>(`/api/v1/admin/gitops/apps/${encodeURIComponent(name)}/sync`, {
        method: "POST",
      }),
    history: (name: string) =>
      this.request<unknown[]>(`/api/v1/admin/gitops/apps/${encodeURIComponent(name)}/history`),
  };

  // Registry
  registry = {
    projects: () => this.request<HarborProject[]>("/api/v1/admin/registry/projects"),
    stats: () => this.request<RegistryStats>("/api/v1/admin/registry/stats"),
    repos: (project: string) =>
      this.request<RegistryRepo[]>(
        `/api/v1/admin/registry/projects/${encodeURIComponent(project)}/repos`
      ),
  };

  // Admin Users (RBAC)
  adminUsers = {
    list: () => this.request<AdminUser[]>("/api/v1/admin/admin-users"),
    invite: (email: string, adminRole: string) =>
      this.request<void>("/api/v1/admin/admin-users", {
        method: "POST",
        body: JSON.stringify({ email, adminRole }),
      }),
    updateRole: (id: string, adminRole: string) =>
      this.request<void>(`/api/v1/admin/admin-users/${id}/role`, {
        method: "PUT",
        body: JSON.stringify({ adminRole }),
      }),
    changeRole: (id: string, adminRole: string) =>
      this.request<void>(`/api/v1/admin/admin-users/${id}/role`, {
        method: "PUT",
        body: JSON.stringify({ adminRole }),
      }),
    remove: (id: string) =>
      this.request<void>(`/api/v1/admin/admin-users/${id}`, { method: "DELETE" }),
  };

  // Quality
  quality = {
    metrics: () => this.request<QualityMetrics>("/api/v1/admin/quality/metrics"),
    tickets: () => this.request<QualityTicket[]>("/api/v1/admin/quality/tickets"),
  };

  // Databases (Admin-level view)
  databases = {
    list: () => this.request<DatabaseCluster[]>("/api/v1/admin/databases"),
    get: (name: string) => this.request<DatabaseCluster>(`/api/v1/admin/databases/${encodeURIComponent(name)}`),
    stats: () => this.request<DatabaseStats>("/api/v1/admin/databases/stats"),
  };

  // Storage (Admin-level view)
  storage = {
    buckets: () => this.request<S3Bucket[]>("/api/v1/admin/storage/s3"),
    volumes: () => this.request<PvcVolume[]>("/api/v1/admin/storage/volumes"),
    stats: () => this.request<StorageStats>("/api/v1/admin/storage/stats"),
  };

  // Dashboards (Grafana)
  dashboards = {
    list: () => this.request<GrafanaDashboard[]>("/api/v1/admin/observability/dashboards"),
  };

  // Logs (Loki)
  logs = {
    query: (query: string, limit?: number) =>
      this.request<LogQueryResult>("/api/v1/admin/observability/logs/query", {
        method: "POST",
        body: JSON.stringify({ query, limit: limit || 100 }),
      }),
    labels: () => this.request<string[]>("/api/v1/admin/observability/logs/labels"),
  };

  // Alerts (Prometheus)
  alerts = {
    list: () => this.request<Alert[]>("/api/v1/admin/observability/alerts"),
    stats: () => this.request<AlertStats>("/api/v1/admin/observability/alerts/stats"),
    rules: () => this.request<AlertRule[]>("/api/v1/admin/observability/alerts/rules"),
  };

  // Traces (Tempo)
  traces = {
    search: (service?: string, minDuration?: string, limit?: number) =>
      this.request<Trace[]>(
        `/api/v1/admin/observability/traces?${service ? `service=${service}&` : ""}${minDuration ? `minDuration=${minDuration}&` : ""}limit=${limit || 20}`
      ),
    get: (id: string) => this.request<unknown>(`/api/v1/admin/observability/traces/${id}`),
  };
}

// Export singleton
export const api = new ApiClient(API_BASE_URL);
