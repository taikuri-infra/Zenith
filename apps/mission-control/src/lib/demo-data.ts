/**
 * Comprehensive mock data for Mission Control demo/showroom mode.
 *
 * When NEXT_PUBLIC_DEMO_MODE=true, all pages use this data instead of calling
 * the real API. The data is designed to look realistic and showcase every
 * feature of the platform.
 */

import type {
  Cluster,
  Module,
  Tenant,
  AuditEntry,
  PlatformUpdate,
  UpdateHistoryEntry,
  InfraOverview,
  PlatformState,
  PlatformSettings,
  DashboardStats,
  Plan,
  Customer,
  CustomerStats,
  CustomerUsage,
  UsageHistoryEntry,
  PlatformUsageSummary,
} from "./api";
import { DEMO_MODE } from "./runtime-env";

// ---------------------------------------------------------------------------
// Helper: whether we are in demo mode
// ---------------------------------------------------------------------------
export const isDemoMode = (): boolean => DEMO_MODE;

// ---------------------------------------------------------------------------
// Clusters (3)
// ---------------------------------------------------------------------------
export const demoClusters: Cluster[] = [
  {
    name: "production-eu",
    k8sVersion: "v1.31.2",
    nodes: 12,
    region: "fsn1",
    type: "shared",
    cpuPercent: 67,
    ramPercent: 72,
    pods: { used: 348, total: 500 },
    pvcs: { used: 94, total: 200 },
    status: "healthy",
  },
  {
    name: "staging-us",
    k8sVersion: "v1.31.2",
    nodes: 4,
    region: "ash1",
    type: "shared",
    cpuPercent: 38,
    ramPercent: 41,
    pods: { used: 87, total: 300 },
    pvcs: { used: 22, total: 100 },
    status: "healthy",
  },
  {
    name: "dev-local",
    k8sVersion: "v1.30.4",
    nodes: 3,
    region: "nbg1",
    type: "dedicated",
    tenant: "embermind",
    cpuPercent: 82,
    ramPercent: 76,
    pods: { used: 156, total: 200 },
    pvcs: { used: 31, total: 50 },
    status: "warning",
    upgradeAvailable: "v1.31.2",
  },
];

// ---------------------------------------------------------------------------
// Tenants (5)
// ---------------------------------------------------------------------------
export const demoTenants: Tenant[] = [
  {
    name: "embermind",
    plan: "pro",
    apps: 28,
    databases: 6,
    cpuUsed: "12.4",
    cpuLimit: "16",
    ramUsed: "14.1",
    ramLimit: "16",
    status: "active",
  },
  {
    name: "acme-corp",
    plan: "pro",
    apps: 45,
    databases: 8,
    cpuUsed: "8.2",
    cpuLimit: "16",
    ramUsed: "11.8",
    ramLimit: "16",
    status: "active",
  },
  {
    name: "starship-io",
    plan: "starter",
    apps: 12,
    databases: 3,
    cpuUsed: "2.4",
    cpuLimit: "4",
    ramUsed: "3.1",
    ramLimit: "4",
    status: "active",
  },
  {
    name: "devhub",
    plan: "starter",
    apps: 5,
    databases: 2,
    cpuUsed: "1.1",
    cpuLimit: "4",
    ramUsed: "1.6",
    ramLimit: "4",
    status: "idle",
  },
  {
    name: "cloudnine",
    plan: "pro",
    apps: 67,
    databases: 11,
    cpuUsed: "18.6",
    cpuLimit: "32",
    ramUsed: "22.4",
    ramLimit: "32",
    status: "active",
  },
];

// ---------------------------------------------------------------------------
// Modules (8)
// ---------------------------------------------------------------------------
export const demoModules: Module[] = [
  {
    name: "Monitoring",
    installed: "v0.72.0",
    latest: "v0.72.0",
    status: "up_to_date",
    description: "Prometheus + Grafana monitoring stack",
  },
  {
    name: "Logging",
    installed: "v3.0.1",
    latest: "v3.1.0",
    status: "update_available",
    description: "Loki log aggregation & search",
  },
  {
    name: "Gateway",
    installed: "v3.6.0",
    latest: "v3.6.0",
    status: "up_to_date",
    description: "APISIX API gateway with rate limiting",
  },
  {
    name: "Auth",
    installed: "v2.1.0",
    latest: "v2.1.0",
    status: "up_to_date",
    description: "Zenith Auth (OpenID Connect + SAML)",
  },
  {
    name: "Registry",
    installed: "v2.10.0",
    latest: "v2.10.2",
    status: "update_available",
    description: "Harbor container image registry",
  },
  {
    name: "Backup",
    installed: "v1.13.1",
    latest: "v1.13.1",
    status: "up_to_date",
    description: "Velero cluster & volume backups",
  },
  {
    name: "Service Mesh",
    installed: "v2.14.0",
    latest: "v2.14.0",
    status: "up_to_date",
    description: "Linkerd lightweight service mesh",
  },
  {
    name: "GitOps",
    installed: "v2.11.0",
    latest: "v2.11.0",
    status: "up_to_date",
    description: "Flux CD declarative delivery",
  },
];

// ---------------------------------------------------------------------------
// Platform Updates (2 available)
// ---------------------------------------------------------------------------
export const demoPlatformUpdate: PlatformUpdate = {
  version: "v1.4.0",
  current: "v1.3.2",
  releasedAt: "February 12, 2026",
  features: [
    "Multi-region failover with automatic DNS switchover",
    "GPU workload scheduling for AI/ML tenants",
    "Consolidated billing dashboard with per-tenant cost breakdown",
    "Improved zen CLI with interactive TUI wizards",
  ],
  breakingChanges: false,
};

export const demoUpdateHistory: UpdateHistoryEntry[] = [
  { version: "v1.3.2", date: "January 28, 2026", status: "installed" },
  { version: "v1.3.1", date: "January 14, 2026", status: "superseded" },
  { version: "v1.3.0", date: "December 20, 2025", status: "superseded" },
  { version: "v1.2.1", date: "November 15, 2025", status: "superseded" },
  { version: "v1.2.0", date: "October 30, 2025", status: "superseded" },
  { version: "v1.1.0", date: "September 22, 2025", status: "superseded" },
  { version: "v1.0.0", date: "August 10, 2025", status: "superseded" },
];

// ---------------------------------------------------------------------------
// Audit Log (15 entries)
// ---------------------------------------------------------------------------
export const demoAuditLog: AuditEntry[] = [
  {
    time: "2026-02-15 14:32",
    actor: "admin",
    action: "Enabled GPU scheduling on production-eu",
    cluster: "production-eu",
  },
  {
    time: "2026-02-15 13:18",
    actor: "system",
    action: "Auto-scaled nodes 11 -> 12",
    cluster: "production-eu",
  },
  {
    time: "2026-02-15 11:45",
    actor: "admin",
    action: 'Created tenant "cloudnine" (pro plan)',
  },
  {
    time: "2026-02-15 10:02",
    actor: "CAPI",
    action: "Node health check passed (all nodes healthy)",
    cluster: "staging-us",
  },
  {
    time: "2026-02-15 09:30",
    actor: "system",
    action: "Nightly backup completed: 34 databases, 2.4 TB total",
  },
  {
    time: "2026-02-14 22:15",
    actor: "admin",
    action: "Enabled module: GitOps (Flux CD v2.11.0)",
    cluster: "production-eu",
  },
  {
    time: "2026-02-14 18:40",
    actor: "admin",
    action: 'Added user "sarah@acme-corp.com" to tenant acme-corp',
  },
  {
    time: "2026-02-14 16:23",
    actor: "system",
    action: "TLS certificate renewed for *.freezenith.com",
  },
  {
    time: "2026-02-14 14:05",
    actor: "CAPI",
    action: "Cluster upgrade completed: v1.30.4 -> v1.31.2",
    cluster: "staging-us",
  },
  {
    time: "2026-02-14 11:30",
    actor: "admin",
    action: "Updated platform settings: retention period 30 -> 90 days",
  },
  {
    time: "2026-02-13 20:12",
    actor: "system",
    action: 'Tenant "devhub" marked as idle (no activity for 7 days)',
  },
  {
    time: "2026-02-13 15:48",
    actor: "admin",
    action: "Created cluster dev-local in nbg1 (dedicated, 3 nodes)",
    cluster: "dev-local",
  },
  {
    time: "2026-02-13 10:17",
    actor: "system",
    action: "Module update available: Registry v2.10.0 -> v2.10.2",
  },
  {
    time: "2026-02-12 23:00",
    actor: "system",
    action: "Platform update available: Zenith v1.4.0",
  },
  {
    time: "2026-02-12 09:30",
    actor: "admin",
    action: 'Suspended tenant "test-project" (billing issue)',
  },
];

// ---------------------------------------------------------------------------
// Infrastructure
// ---------------------------------------------------------------------------
export const demoInfrastructure: InfraOverview = {
  servers: 19,
  volumes: 48,
  volumeSize: "4.2 TB",
  loadBalancers: 4,
  lbPublic: 3,
  lbInternal: 1,
  monthlyCost: "\u20AC287.40",
  resources: [
    {
      name: "Control Plane",
      type: "CX22",
      count: 1,
      cluster: "management",
      monthlyCost: "\u20AC5.39",
    },
    {
      name: "Worker Nodes",
      type: "CX32",
      count: 12,
      cluster: "production-eu",
      monthlyCost: "\u20AC155.88",
    },
    {
      name: "Worker Nodes",
      type: "CX22",
      count: 4,
      cluster: "staging-us",
      monthlyCost: "\u20AC21.56",
    },
    {
      name: "Worker Nodes",
      type: "CX22",
      count: 3,
      cluster: "dev-local",
      monthlyCost: "\u20AC16.17",
    },
    {
      name: "Persistent Volumes",
      type: "SSD",
      count: 48,
      cluster: "all",
      monthlyCost: "\u20AC52.80",
    },
    {
      name: "Load Balancers",
      type: "LB11",
      count: 4,
      cluster: "all",
      monthlyCost: "\u20AC23.60",
    },
    {
      name: "Floating IPs",
      type: "IPv4",
      count: 3,
      cluster: "all",
      monthlyCost: "\u20AC12.00",
    },
  ],
};

// ---------------------------------------------------------------------------
// Platform State
// ---------------------------------------------------------------------------
export const demoPlatformState: PlatformState = {
  platformVersion: "v1.3.2",
  updateAvailable: "v1.4.0",
  installedDate: "Aug 10, 2025",
  installedDaysAgo: 189,
  managementK8sVersion: "v1.31.2",
  managementK8sUpToDate: true,
  domain: "freezenith.com",
  wildcardTls: true,
};

// ---------------------------------------------------------------------------
// Platform Settings
// ---------------------------------------------------------------------------
export const demoPlatformSettings: PlatformSettings = {
  platformName: "Zenith Production",
  baseDomain: "freezenith.com",
  provider: "Hetzner Cloud",
  defaultRegion: "fsn1",
  regionLabel: "Falkenstein, DE",
  autoBackups: true,
  retentionDays: 90,
};

// ---------------------------------------------------------------------------
// Dashboard Stats
// ---------------------------------------------------------------------------
export const demoDashboardStats: DashboardStats = {
  clusterCount: 3,
  allHealthy: false,
  tenantCount: 5,
  activeToday: 4,
  monthlyCost: "\u20AC287",
  costProvider: "Hetzner Cloud",
  updatesAvailable: 2,
  customerCount: 5,
  activeCustomers: 4,
  mrr: "\u20AC4,595",
  newCustomersThisMonth: 1,
};

// ---------------------------------------------------------------------------
// Plans (3)
// ---------------------------------------------------------------------------
export const demoPlans: Plan[] = [
  {
    id: "plan-starter",
    name: "Starter",
    cpuCores: 4,
    ramGb: 8,
    s3Tb: 0,
    dbStorageGb: 10,
    volumeGb: 50,
    lbCount: 1,
    priceCents: 9900,
    currency: "EUR",
    billingCycle: "monthly",
    active: true,
    createdAt: "2025-08-10T00:00:00Z",
    updatedAt: "2025-08-10T00:00:00Z",
  },
  {
    id: "plan-pro",
    name: "Pro",
    cpuCores: 16,
    ramGb: 32,
    s3Tb: 1,
    dbStorageGb: 100,
    volumeGb: 500,
    lbCount: 3,
    priceCents: 49900,
    currency: "EUR",
    billingCycle: "monthly",
    active: true,
    createdAt: "2025-08-10T00:00:00Z",
    updatedAt: "2025-08-10T00:00:00Z",
  },
  {
    id: "plan-enterprise",
    name: "Enterprise",
    cpuCores: 64,
    ramGb: 128,
    s3Tb: 10,
    dbStorageGb: 1000,
    volumeGb: 5000,
    lbCount: 10,
    priceCents: 199900,
    currency: "EUR",
    billingCycle: "monthly",
    active: true,
    createdAt: "2025-08-10T00:00:00Z",
    updatedAt: "2025-08-10T00:00:00Z",
  },
];

// ---------------------------------------------------------------------------
// Customers (5)
// ---------------------------------------------------------------------------
export const demoCustomers: Customer[] = [
  {
    id: "cust-001",
    name: "Embermind",
    domain: "embermind.app",
    planId: "plan-pro",
    contactEmail: "ops@embermind.app",
    contactName: "Sarah Chen",
    status: "active",
    clusterStatus: "running",
    capiClusterName: "embermind-app",
    clusterRegion: "fsn1",
    clusterNodes: 3,
    clusterK8sVersion: "v1.31.2",
    notes: "Launch partner. VIP support.",
    createdAt: "2025-09-15T10:00:00Z",
    updatedAt: "2026-02-10T14:00:00Z",
    plan: demoPlans[1],
  },
  {
    id: "cust-002",
    name: "Acme Corp",
    domain: "acme-corp.com",
    planId: "plan-pro",
    contactEmail: "infra@acme-corp.com",
    contactName: "James Wilson",
    status: "active",
    clusterStatus: "running",
    capiClusterName: "acme-corp-com",
    clusterRegion: "fsn1",
    clusterNodes: 5,
    clusterK8sVersion: "v1.31.2",
    notes: "",
    createdAt: "2025-10-22T09:00:00Z",
    updatedAt: "2026-01-15T11:00:00Z",
    plan: demoPlans[1],
  },
  {
    id: "cust-003",
    name: "Starship IO",
    domain: "starship.io",
    planId: "plan-starter",
    contactEmail: "admin@starship.io",
    contactName: "Alex Rivera",
    status: "active",
    clusterStatus: "running",
    capiClusterName: "starship-io",
    clusterRegion: "fsn1",
    clusterNodes: 3,
    clusterK8sVersion: "v1.31.2",
    notes: "",
    createdAt: "2025-12-01T08:00:00Z",
    updatedAt: "2026-02-01T16:00:00Z",
    plan: demoPlans[0],
  },
  {
    id: "cust-004",
    name: "DevHub",
    domain: "devhub.dev",
    planId: "plan-starter",
    contactEmail: "team@devhub.dev",
    contactName: "Maria Santos",
    status: "suspended",
    clusterStatus: "running",
    capiClusterName: "devhub-dev",
    clusterRegion: "fsn1",
    clusterNodes: 3,
    clusterK8sVersion: "v1.31.2",
    notes: "Billing issue. Follow up Feb 20.",
    createdAt: "2025-11-10T12:00:00Z",
    updatedAt: "2026-02-12T09:30:00Z",
    plan: demoPlans[0],
  },
  {
    id: "cust-005",
    name: "CloudNine",
    domain: "cloudnine.cloud",
    planId: "plan-enterprise",
    contactEmail: "platform@cloudnine.cloud",
    contactName: "Tom Baker",
    status: "active",
    clusterStatus: "provisioning",
    capiClusterName: "cloudnine-cloud",
    clusterRegion: "fsn1",
    clusterNodes: 10,
    clusterK8sVersion: "v1.31.2",
    notes: "Enterprise onboarding in progress.",
    createdAt: "2026-02-05T14:00:00Z",
    updatedAt: "2026-02-15T10:00:00Z",
    plan: demoPlans[2],
  },
];

// ---------------------------------------------------------------------------
// Customer Stats
// ---------------------------------------------------------------------------
export const demoCustomerStats: CustomerStats = {
  totalCustomers: 5,
  activeCustomers: 4,
  mrr: "\u20AC4,595",
  newThisMonth: 1,
};

// ---------------------------------------------------------------------------
// Customer Usage (per customer)
// ---------------------------------------------------------------------------
export const demoCustomerUsage: Record<string, CustomerUsage> = {
  "cust-001": {
    cpuCores: 10.4,
    cpuCeiling: 16,
    cpuPercent: 65.0,
    ramGb: 22.1,
    ramCeiling: 32,
    ramPercent: 69.1,
    s3Tb: 0.3,
    s3Ceiling: 1,
    s3Percent: 30.0,
    dbStorageGb: 42,
    dbCeiling: 100,
    dbPercent: 42.0,
    volumeGb: 180,
    volCeiling: 500,
    volPercent: 36.0,
    lbCount: 2,
    lbCeiling: 3,
    lbPercent: 66.7,
    recordedAt: "2026-02-20T12:00:00Z",
  },
  "cust-002": {
    cpuCores: 6.8,
    cpuCeiling: 16,
    cpuPercent: 42.5,
    ramGb: 14.2,
    ramCeiling: 32,
    ramPercent: 44.4,
    s3Tb: 0.1,
    s3Ceiling: 1,
    s3Percent: 10.0,
    dbStorageGb: 25,
    dbCeiling: 100,
    dbPercent: 25.0,
    volumeGb: 95,
    volCeiling: 500,
    volPercent: 19.0,
    lbCount: 1,
    lbCeiling: 3,
    lbPercent: 33.3,
    recordedAt: "2026-02-20T12:00:00Z",
  },
  "cust-003": {
    cpuCores: 2.8,
    cpuCeiling: 4,
    cpuPercent: 70.0,
    ramGb: 6.9,
    ramCeiling: 8,
    ramPercent: 86.3,
    s3Tb: 0,
    s3Ceiling: 0,
    s3Percent: 0,
    dbStorageGb: 6,
    dbCeiling: 10,
    dbPercent: 60.0,
    volumeGb: 30,
    volCeiling: 50,
    volPercent: 60.0,
    lbCount: 1,
    lbCeiling: 1,
    lbPercent: 100.0,
    recordedAt: "2026-02-20T12:00:00Z",
  },
  "cust-004": {
    cpuCores: 0,
    cpuCeiling: 4,
    cpuPercent: 0,
    ramGb: 0,
    ramCeiling: 8,
    ramPercent: 0,
    s3Tb: 0,
    s3Ceiling: 0,
    s3Percent: 0,
    dbStorageGb: 0,
    dbCeiling: 10,
    dbPercent: 0,
    volumeGb: 0,
    volCeiling: 50,
    volPercent: 0,
    lbCount: 0,
    lbCeiling: 1,
    lbPercent: 0,
    recordedAt: "",
  },
  "cust-005": {
    cpuCores: 0,
    cpuCeiling: 64,
    cpuPercent: 0,
    ramGb: 0,
    ramCeiling: 128,
    ramPercent: 0,
    s3Tb: 0,
    s3Ceiling: 10,
    s3Percent: 0,
    dbStorageGb: 0,
    dbCeiling: 1000,
    dbPercent: 0,
    volumeGb: 0,
    volCeiling: 5000,
    volPercent: 0,
    lbCount: 0,
    lbCeiling: 10,
    lbPercent: 0,
    recordedAt: "",
  },
};

// ---------------------------------------------------------------------------
// Usage History (30 days for cust-001)
// ---------------------------------------------------------------------------
function generateUsageHistory(): UsageHistoryEntry[] {
  const entries: UsageHistoryEntry[] = [];
  const now = new Date();
  for (let d = 29; d >= 0; d--) {
    const date = new Date(now);
    date.setDate(date.getDate() - d);
    const cpuBase = 8 + Math.random() * 4;
    const ramBase = 18 + Math.random() * 6;
    entries.push({
      date: date.toISOString().slice(0, 10),
      cpuAvg: Math.round(cpuBase * 100) / 100,
      cpuMax: Math.round((cpuBase + 1 + Math.random() * 2) * 100) / 100,
      ramAvg: Math.round(ramBase * 100) / 100,
      ramMax: Math.round((ramBase + 1 + Math.random() * 3) * 100) / 100,
      dbStorageGb: 42,
      volumeGb: 180,
      lbCount: 2,
    });
  }
  return entries;
}

export const demoUsageHistory: UsageHistoryEntry[] = generateUsageHistory();

// ---------------------------------------------------------------------------
// Platform Usage Summary
// ---------------------------------------------------------------------------
export const demoPlatformUsageSummary: PlatformUsageSummary = {
  totalCpu: 20.0,
  totalRam: 43.2,
  totalStorage: 378.0,
  customersReporting: 3,
};
