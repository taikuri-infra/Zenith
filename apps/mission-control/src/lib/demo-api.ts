/**
 * Demo API client that returns mock data with a realistic delay.
 * Mirrors the shape of the real ApiClient so pages can use it as a drop-in.
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
  Customer,
  Plan,
  CustomerStats,
  CustomerUsage,
  UsageHistoryEntry,
  PlatformUsageSummary,
  WarRoomData,
  RevenueStats,
  GrowthStats,
  UsageAnalytics,
  CohortData,
  CrmPipeline,
  HealthScoreItem,
  CustomerNoteItem,
  ServiceHealthItem,
  ServiceDetailItem,
  DatabaseCluster,
  DatabaseStats,
  S3Bucket,
  PvcVolume,
  StorageStats,
  DnsRecord,
  Route,
  Certificate,
  GrafanaDashboard,
  LogQueryResult,
  Alert,
  AlertStats,
  AlertRule,
  Trace,
  SecurityPosture,
  SecurityOverview,
  KyvernoPolicy,
  WafStats,
  ImageScanResult,
  ImageScanStats,
  ActiveSession,
  BackupStatusData,
  BackupStats,
  VeleroSchedule,
  CnpgBackup,
  ArgoApp,
  ArgoAppItem,
  GitOpsStats,
  HarborProject,
  RegistryStats,
  RegistryRepo,
  AdminUser,
  QualityMetrics,
  QualityTicket,
  SupportTicket,
  SupportMessage,
} from "./api";

import {
  demoClusters,
  demoModules,
  demoTenants,
  demoAuditLog,
  demoPlatformUpdate,
  demoUpdateHistory,
  demoInfrastructure,
  demoPlatformState,
  demoPlatformSettings,
  demoDashboardStats,
  demoCustomers,
  demoPlans,
  demoCustomerStats,
  demoCustomerUsage,
  demoUsageHistory,
  demoPlatformUsageSummary,
} from "./demo-data";

// Simulate a short network delay so skeleton states flash briefly
const delay = (ms = 300) => new Promise<void>((r) => setTimeout(r, ms));

export const demoApi = {
  dashboard: {
    stats: async (): Promise<DashboardStats> => {
      await delay();
      return demoDashboardStats;
    },
    usage: async (): Promise<PlatformUsageSummary> => {
      await delay();
      return demoPlatformUsageSummary;
    },
  },

  clusters: {
    list: async (): Promise<Cluster[]> => {
      await delay();
      return demoClusters;
    },
    get: async (name: string): Promise<Cluster> => {
      await delay();
      const cluster = demoClusters.find((c) => c.name === name);
      if (!cluster) throw new Error(`Cluster "${name}" not found`);
      return cluster;
    },
    create: async (): Promise<Cluster> => {
      throw new Error("Not available in demo mode");
    },
    delete: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
    upgrade: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
  },

  tenants: {
    list: async (): Promise<Tenant[]> => {
      await delay();
      return demoTenants;
    },
    get: async (id: string): Promise<Tenant> => {
      await delay();
      const tenant = demoTenants.find((t) => t.name === id);
      if (!tenant) throw new Error(`Tenant "${id}" not found`);
      return tenant;
    },
    suspend: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
  },

  modules: {
    list: async (): Promise<Module[]> => {
      await delay();
      return demoModules;
    },
    install: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
    uninstall: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
    update: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
    updateAll: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
  },

  audit: {
    list: async (params?: {
      limit?: number;
      offset?: number;
      actor?: string;
      cluster?: string;
      period?: string;
    }): Promise<AuditEntry[]> => {
      await delay();
      let filtered = [...demoAuditLog];
      if (params?.actor) {
        filtered = filtered.filter((e) => e.actor === params.actor);
      }
      if (params?.cluster) {
        filtered = filtered.filter((e) => e.cluster === params.cluster);
      }
      if (params?.limit) {
        filtered = filtered.slice(0, params.limit);
      }
      return filtered;
    },
  },

  updates: {
    check: async (): Promise<PlatformUpdate> => {
      await delay();
      return demoPlatformUpdate;
    },
    apply: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
    history: async (): Promise<UpdateHistoryEntry[]> => {
      await delay();
      return demoUpdateHistory;
    },
  },

  infrastructure: {
    overview: async (): Promise<InfraOverview> => {
      await delay();
      return demoInfrastructure;
    },
  },

  state: {
    get: async (): Promise<PlatformState> => {
      await delay();
      return demoPlatformState;
    },
    export: async (): Promise<string> => {
      await delay();
      return "# Zenith Platform State Export\n# Generated: 2026-02-15\n---\napiVersion: zenith.dev/v1\nkind: PlatformState\nspec:\n  version: v1.3.2\n  clusters: 3\n  tenants: 5\n  modules: 8";
    },
  },

  settings: {
    get: async (): Promise<PlatformSettings> => {
      await delay();
      return demoPlatformSettings;
    },
    update: async (): Promise<PlatformSettings> => {
      throw new Error("Not available in demo mode");
    },
  },

  customers: {
    list: async (): Promise<Customer[]> => {
      await delay();
      return demoCustomers;
    },
    get: async (id: string): Promise<Customer> => {
      await delay();
      const customer = demoCustomers.find((c) => c.id === id);
      if (!customer) throw new Error(`Customer "${id}" not found`);
      return customer;
    },
    create: async (): Promise<Customer> => {
      throw new Error("Not available in demo mode");
    },
    update: async (): Promise<Customer> => {
      throw new Error("Not available in demo mode");
    },
    delete: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
    suspend: async (): Promise<Customer> => {
      throw new Error("Not available in demo mode");
    },
    activate: async (): Promise<Customer> => {
      throw new Error("Not available in demo mode");
    },
    stats: async (): Promise<CustomerStats> => {
      await delay();
      return demoCustomerStats;
    },
    getCluster: async (id: string): Promise<Cluster> => {
      await delay();
      const customer = demoCustomers.find((c) => c.id === id);
      if (!customer) throw new Error(`Customer "${id}" not found`);
      // Return a cluster-like object from the customer's cluster info
      return {
        name: customer.capiClusterName,
        k8sVersion: customer.clusterK8sVersion,
        nodes: customer.clusterNodes,
        region: customer.clusterRegion,
        type: "dedicated",
        cpuPercent: 45,
        ramPercent: 52,
        pods: { used: 87, total: 200 },
        pvcs: { used: 12, total: 50 },
        status: customer.clusterStatus === "running" ? "healthy" : "warning",
      } as Cluster;
    },
    scaleCluster: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
    upgradeCluster: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
    usage: async (id: string): Promise<CustomerUsage> => {
      await delay();
      const usage = demoCustomerUsage[id];
      if (!usage) throw new Error(`No usage data for "${id}"`);
      return usage;
    },
    usageHistory: async (id: string, _days?: number): Promise<UsageHistoryEntry[]> => {
      await delay();
      // Return the shared history for all customers (generated for cust-001 profile)
      if (!demoCustomerUsage[id]) throw new Error(`Customer "${id}" not found`);
      return demoUsageHistory;
    },
  },

  plans: {
    list: async (): Promise<Plan[]> => {
      await delay();
      return demoPlans;
    },
    create: async (): Promise<Plan> => {
      throw new Error("Not available in demo mode");
    },
    update: async (): Promise<Plan> => {
      throw new Error("Not available in demo mode");
    },
  },

  support: {
    list: async (): Promise<{ items: SupportTicket[]; total: number }> => {
      await delay();
      return { items: [], total: 0 };
    },
    get: async (): Promise<{ ticket: SupportTicket; messages: SupportMessage[] }> => {
      throw new Error("Not available in demo mode");
    },
    reply: async (): Promise<SupportMessage> => {
      throw new Error("Not available in demo mode");
    },
    updateStatus: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
    assign: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
  },

  // --- Mission Control v2 ---

  warRoom: {
    get: async (): Promise<WarRoomData> => {
      await delay();
      return {
        kpis: {
          mrr: 12500, mrrTrend: 8.2, activeCustomers: 42, totalCustomers: 56,
          newSignups: 7, churnRate: 2.1, avgResponseTime: "14m", healthScore: 94,
        },
        serviceHealth: [
          { name: "traefik", namespace: "zenith-platform", kind: "Deployment", status: "healthy", readyReplicas: 1, totalReplicas: 1, restarts: 0 },
          { name: "apisix", namespace: "zenith-platform", kind: "Deployment", status: "healthy", readyReplicas: 1, totalReplicas: 1, restarts: 0 },
          { name: "zenith-api", namespace: "zenith-platform", kind: "Deployment", status: "healthy", readyReplicas: 2, totalReplicas: 2, restarts: 0 },
          { name: "grafana", namespace: "monitoring", kind: "Deployment", status: "healthy", readyReplicas: 1, totalReplicas: 1, restarts: 0 },
          { name: "prometheus", namespace: "monitoring", kind: "StatefulSet", status: "healthy", readyReplicas: 1, totalReplicas: 1, restarts: 0 },
        ],
        recentAlerts: [],
        activeTickets: [
          { id: "t-1", subject: "Cannot deploy app", priority: "high", status: "open", age: "2h" },
          { id: "t-2", subject: "SSL certificate issue", priority: "medium", status: "in-progress", age: "1d" },
        ],
      };
    },
  },

  analytics: {
    revenue: async (): Promise<RevenueStats> => {
      await delay();
      return {
        mrr: 12500, arr: 150000, churnRate: 2.1, ltv: 4800,
        revenueByPlan: [
          { plan: "Free", revenue: 0, count: 20 },
          { plan: "Pro", revenue: 5800, count: 200 },
          { plan: "Team", revenue: 2970, count: 10 },
          { plan: "Business", revenue: 3730, count: 8 },
        ],
        monthlyTrend: Array.from({ length: 12 }, (_, i) => ({
          month: `2025-${String(i + 1).padStart(2, "0")}`,
          newRevenue: 800 + Math.random() * 400,
          churnedRevenue: 100 + Math.random() * 200,
          totalMrr: 8000 + i * 400,
        })),
      };
    },
    growth: async (): Promise<GrowthStats> => {
      await delay();
      return {
        totalUsers: 238, newThisMonth: 7, churnedThisMonth: 2,
        monthlyGrowth: Array.from({ length: 12 }, (_, i) => ({
          month: `2025-${String(i + 1).padStart(2, "0")}`,
          new: 5 + Math.floor(Math.random() * 8),
          churned: Math.floor(Math.random() * 3),
          total: 180 + i * 5,
        })),
        conversions: { freeToProRate: 12.5, proToTeamRate: 4.2, trialToPayRate: 28.0 },
      };
    },
    usage: async (): Promise<UsageAnalytics> => {
      await delay();
      return {
        topFeatures: [
          { feature: "Apps", usageCount: 340, userCount: 180 },
          { feature: "Databases", usageCount: 120, userCount: 85 },
          { feature: "Storage", usageCount: 95, userCount: 60 },
          { feature: "Gateways", usageCount: 45, userCount: 30 },
        ],
        avgAppsPerUser: 1.8, avgDbsPerUser: 0.6,
      };
    },
    cohorts: async (): Promise<CohortData[]> => {
      await delay();
      return [
        { cohort: "2025-01", month: "M1", retained: 48, total: 50, percentage: 96 },
        { cohort: "2025-01", month: "M2", retained: 42, total: 50, percentage: 84 },
        { cohort: "2025-01", month: "M3", retained: 38, total: 50, percentage: 76 },
      ];
    },
  },

  surveys: {
    insights: async () => {
      await delay();
      return {
        total_responses: 47,
        responses: [
          { user_id: "u-001", created_at: "2026-03-12T10:00:00Z", use_case: "saas", role: "fullstack", team_size: "small", company_name: "IndieApp", current_provider: "heroku", monthly_spend: "50_200", biggest_pain: "cost", expected_traffic: "10k_100k", timeline: "this_month", most_important: "auto_scaling", stack: ["Node.js", "React", "PostgreSQL"], discovery: "google" },
          { user_id: "u-002", created_at: "2026-03-11T15:00:00Z", use_case: "startup", role: "cto", team_size: "medium", company_name: "TechCorp", current_provider: "aws", monthly_spend: "500_2000", biggest_pain: "complexity", expected_traffic: "100k_1m", timeline: "next_quarter", most_important: "security", stack: ["Go", "React", "PostgreSQL", "Kubernetes"], discovery: "linkedin" },
          { user_id: "u-003", created_at: "2026-03-10T09:00:00Z", use_case: "migrate", role: "devops", team_size: "large", company_name: "ScaleUp GmbH", current_provider: "vercel", monthly_spend: "200_500", biggest_pain: "lock_in", expected_traffic: "10k_100k", timeline: "this_month", most_important: "cost_control", stack: ["Next.js", "Python", "PostgreSQL"], discovery: "reddit" },
        ],
        breakdowns: {
          use_case: { saas: 15, startup: 10, side_project: 8, migrate: 6, learn: 4, agency: 2, evaluate: 2 },
          role: { developer: 14, fullstack: 12, cto: 8, devops: 6, founder: 4, student: 3 },
          team_size: { solo: 12, small: 15, medium: 10, large: 6, enterprise: 4 },
          current_provider: { heroku: 10, aws: 8, vercel: 7, nowhere: 6, railway: 5, digitalocean: 4, self_hosted: 3, gcp: 2, azure: 1, other: 1 },
          monthly_spend: { "0": 8, under_50: 10, "50_200": 12, "200_500": 8, "500_2000": 5, over_2000: 4 },
          biggest_pain: { cost: 15, complexity: 10, speed: 6, scaling: 5, lock_in: 4, support: 3, compliance: 2, none: 2 },
          expected_traffic: { starting: 12, under_10k: 15, "10k_100k": 10, "100k_1m": 6, over_1m: 4 },
          timeline: { exploring: 8, this_week: 5, this_month: 15, next_quarter: 12, already_live: 7 },
          most_important: { auto_scaling: 10, managed_db: 8, cicd: 7, cost_control: 6, security: 6, monitoring: 4, custom_domains: 3, team_collab: 3 },
          stack: { "Node.js": 25, React: 22, PostgreSQL: 20, "Next.js": 15, Python: 12, Docker: 10, Go: 8, Redis: 7, Vue: 5, Kubernetes: 5, TypeScript: 4, MySQL: 3 },
          discovery: { google: 14, friend: 8, reddit: 6, twitter: 5, linkedin: 4, youtube: 3, github: 3, blog: 2, producthunt: 1, conference: 1 },
        },
      };
    },
  },

  crm: {
    pipeline: async (): Promise<CrmPipeline> => {
      await delay();
      return {
        stages: [
          { name: "trial", count: 1, customers: [{ id: "c1", name: "Acme Corp", email: "acme@test.com", healthScore: 60, plan: "Free", mrr: 0 }] },
          { name: "active", count: 1, customers: [{ id: "c3", name: "BigCo", email: "big@test.com", healthScore: 90, plan: "Business", mrr: 149 }] },
          { name: "at_risk", count: 0, customers: [] },
          { name: "churned", count: 0, customers: [] },
        ],
      };
    },
    healthScores: async (): Promise<HealthScoreItem[]> => {
      await delay();
      return [
        { userId: "u1", score: 90, usageScore: 85, supportScore: 95, loginScore: 90, riskLevel: "low" },
        { userId: "u2", score: 45, usageScore: 30, supportScore: 60, loginScore: 45, riskLevel: "high" },
      ];
    },
    getNotes: async (): Promise<CustomerNoteItem[]> => {
      await delay();
      return [];
    },
    saveNote: async (): Promise<void> => {
      await delay();
    },
    updateTags: async (): Promise<void> => {
      await delay();
    },
  },

  services: {
    list: async (): Promise<ServiceHealthItem[]> => {
      await delay();
      const svcs: ServiceHealthItem[] = [
        { name: "traefik", namespace: "zenith-platform", kind: "Deployment", status: "healthy", readyReplicas: 1, totalReplicas: 1, restarts: 0 },
        { name: "apisix", namespace: "zenith-platform", kind: "Deployment", status: "healthy", readyReplicas: 1, totalReplicas: 1, restarts: 0 },
        { name: "cert-manager", namespace: "cert-manager", kind: "Deployment", status: "healthy", readyReplicas: 1, totalReplicas: 1, restarts: 0 },
        { name: "zenith-api", namespace: "zenith-platform", kind: "Deployment", status: "healthy", readyReplicas: 2, totalReplicas: 2, restarts: 0 },
        { name: "keycloak", namespace: "zenith-staging", kind: "StatefulSet", status: "healthy", readyReplicas: 1, totalReplicas: 1, restarts: 0 },
        { name: "argocd-server", namespace: "zenith-platform", kind: "Deployment", status: "healthy", readyReplicas: 1, totalReplicas: 1, restarts: 0 },
        { name: "grafana", namespace: "monitoring", kind: "Deployment", status: "healthy", readyReplicas: 1, totalReplicas: 1, restarts: 0 },
        { name: "prometheus", namespace: "monitoring", kind: "StatefulSet", status: "healthy", readyReplicas: 1, totalReplicas: 1, restarts: 0 },
        { name: "loki", namespace: "monitoring", kind: "StatefulSet", status: "healthy", readyReplicas: 1, totalReplicas: 1, restarts: 0 },
        { name: "tempo", namespace: "monitoring", kind: "Deployment", status: "healthy", readyReplicas: 1, totalReplicas: 1, restarts: 0 },
        { name: "velero", namespace: "zenith-platform", kind: "Deployment", status: "healthy", readyReplicas: 1, totalReplicas: 1, restarts: 0 },
        { name: "kyverno", namespace: "zenith-platform", kind: "Deployment", status: "healthy", readyReplicas: 1, totalReplicas: 1, restarts: 0 },
        { name: "falco", namespace: "zenith-platform", kind: "DaemonSet", status: "healthy", readyReplicas: 1, totalReplicas: 1, restarts: 0 },
        { name: "harbor", namespace: "zenith-platform", kind: "Deployment", status: "healthy", readyReplicas: 1, totalReplicas: 1, restarts: 0 },
        { name: "nats", namespace: "zenith-platform", kind: "StatefulSet", status: "healthy", readyReplicas: 1, totalReplicas: 1, restarts: 0 },
        { name: "temporal", namespace: "zenith-platform", kind: "Deployment", status: "healthy", readyReplicas: 1, totalReplicas: 1, restarts: 0 },
        { name: "keda-operator", namespace: "zenith-platform", kind: "Deployment", status: "healthy", readyReplicas: 1, totalReplicas: 1, restarts: 0 },
        { name: "sealed-secrets", namespace: "zenith-platform", kind: "Deployment", status: "healthy", readyReplicas: 1, totalReplicas: 1, restarts: 0 },
        { name: "external-dns", namespace: "zenith-platform", kind: "Deployment", status: "healthy", readyReplicas: 1, totalReplicas: 1, restarts: 0 },
        { name: "otel-collector", namespace: "monitoring", kind: "Deployment", status: "healthy", readyReplicas: 1, totalReplicas: 1, restarts: 0 },
        { name: "zenith-postgres", namespace: "zenith-staging", kind: "Cluster", status: "healthy", readyReplicas: 1, totalReplicas: 1, restarts: 0 },
        { name: "free-pg", namespace: "zenith-shared", kind: "Cluster", status: "healthy", readyReplicas: 1, totalReplicas: 1, restarts: 0 },
      ];
      return svcs;
    },
    get: async (name: string): Promise<ServiceDetailItem> => {
      await delay();
      return {
        name, namespace: "zenith-platform", kind: "Deployment", status: "healthy",
        readyReplicas: 1, totalReplicas: 1, restarts: 0,
        pods: [{ name: `${name}-abc123`, status: "Running", restarts: 0, age: "5d" }],
        events: [],
      };
    },
    restart: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
  },

  databases: {
    list: async (): Promise<DatabaseCluster[]> => {
      await delay();
      return [
        { name: "zenith-postgres", namespace: "zenith-staging", status: "healthy", readyInstances: 1, totalInstances: 1, storage: "10Gi", walArchiving: true, lastBackup: "2026-03-09T02:00:00Z", pgVersion: "16.6" },
        { name: "free-pg", namespace: "zenith-shared", status: "healthy", readyInstances: 1, totalInstances: 1, storage: "10Gi", walArchiving: true, lastBackup: "2026-03-09T02:00:00Z", pgVersion: "16.6" },
      ];
    },
    get: async (name: string): Promise<DatabaseCluster> => {
      await delay();
      return { name, namespace: "zenith-staging", status: "healthy", readyInstances: 1, totalInstances: 1, storage: "10Gi", walArchiving: true, pgVersion: "16.6" };
    },
    stats: async (): Promise<DatabaseStats> => {
      await delay();
      return { totalClusters: 2, healthyClusters: 2, totalStorage: "20Gi", lastBackup: "2026-03-09T02:00:00Z" };
    },
  },

  adminDatabases: {
    list: async (): Promise<DatabaseCluster[]> => {
      await delay();
      return [
        { name: "zenith-postgres", namespace: "zenith-staging", status: "healthy", readyInstances: 1, totalInstances: 1, storage: "10Gi", walArchiving: true, pgVersion: "16.6" },
        { name: "free-pg", namespace: "zenith-shared", status: "healthy", readyInstances: 1, totalInstances: 1, storage: "10Gi", walArchiving: true, pgVersion: "16.6" },
      ];
    },
    get: async (name: string): Promise<DatabaseCluster> => {
      await delay();
      return { name, namespace: "zenith-staging", status: "healthy", readyInstances: 1, totalInstances: 1, storage: "10Gi", walArchiving: true, pgVersion: "16.6" };
    },
  },

  storage: {
    buckets: async (): Promise<S3Bucket[]> => {
      await delay();
      return [
        { name: "zenith-backups", region: "fsn1", size: "2.4 GB", objectCount: 1240, createdAt: "2026-01-15" },
        { name: "zenith-production", region: "fsn1", size: "128 MB", objectCount: 45, createdAt: "2026-01-15" },
      ];
    },
    volumes: async (): Promise<PvcVolume[]> => {
      await delay();
      return [
        { name: "data-zenith-postgres-1", namespace: "zenith-staging", status: "Bound", capacity: "10Gi", storageClass: "hcloud-volumes" },
        { name: "data-free-pg-1", namespace: "zenith-shared", status: "Bound", capacity: "10Gi", storageClass: "hcloud-volumes" },
      ];
    },
    stats: async (): Promise<StorageStats> => {
      await delay();
      return { totalBuckets: 2, s3Used: "2.5 GB", totalVolumes: 2, pvcUsed: "20 Gi" };
    },
  },

  adminStorage: {
    s3: async (): Promise<S3Bucket[]> => {
      await delay();
      return [
        { name: "zenith-backups", region: "fsn1", size: "2.4 GB", objectCount: 1240, createdAt: "2026-01-15" },
      ];
    },
    volumes: async (): Promise<PvcVolume[]> => {
      await delay();
      return [];
    },
  },

  networking: {
    dns: async (): Promise<DnsRecord[]> => {
      await delay();
      return [
        { name: "*.apps.stage.freezenith.com", type: "A", value: "77.42.88.149", ttl: 300, managedBy: "external-dns" },
        { name: "api.stage.freezenith.com", type: "A", value: "77.42.88.149", ttl: 300, managedBy: "terraform" },
      ];
    },
    dnsRecords: async (): Promise<DnsRecord[]> => {
      await delay();
      return [
        { name: "*.apps.stage.freezenith.com", type: "A", value: "77.42.88.149", ttl: 300, managedBy: "external-dns" },
      ];
    },
    routes: async (): Promise<Route[]> => {
      await delay();
      return [
        { name: "zenith-api", host: "api.stage.freezenith.com", service: "zenith-api", port: 8080, tls: true, namespace: "zenith-platform" },
        { name: "zenith-web", host: "app.stage.freezenith.com", service: "zenith-web", port: 3000, tls: true, namespace: "zenith-platform" },
      ];
    },
    certificates: async (): Promise<Certificate[]> => {
      await delay();
      return [
        { name: "api-tls", domains: ["api.stage.freezenith.com"], issuer: "letsencrypt-prod", ready: true, expiresAt: "2026-06-10" },
        { name: "apps-wildcard-tls", domains: ["*.apps.stage.freezenith.com"], issuer: "letsencrypt-prod", ready: true, expiresAt: "2026-06-10" },
      ];
    },
  },

  observability: {
    dashboards: async (): Promise<GrafanaDashboard[]> => {
      await delay();
      return [
        { uid: "node-health", title: "Node Health", url: "#", category: "infrastructure" },
        { uid: "pod-resources", title: "Pod Resources", url: "#", category: "infrastructure" },
        { uid: "apisix-traffic", title: "APISIX Traffic", url: "#", category: "networking" },
      ];
    },
    queryLogs: async (): Promise<LogQueryResult> => {
      await delay();
      return { lines: [{ timestamp: "2026-03-09T12:00:00Z", labels: "{app=zenith-api}", message: "GET /api/v1/health 200" }], executionTimeMs: 42 };
    },
    logLabels: async (): Promise<string[]> => {
      await delay();
      return ["app", "namespace", "pod", "container", "node"];
    },
    alerts: async (): Promise<Alert[]> => {
      await delay();
      return [];
    },
    alertRules: async (): Promise<AlertRule[]> => {
      await delay();
      return [
        { name: "HighCPU", group: "node", query: "cpu > 80%", duration: "5m", severity: "warning", state: "inactive" },
        { name: "PodCrashLooping", group: "pod", query: "restarts > 5", duration: "10m", severity: "critical", state: "inactive" },
      ];
    },
    traces: async (): Promise<Trace[]> => {
      await delay();
      return [
        { traceId: "abc123", service: "zenith-api", operationName: "GET /api/v1/health", durationMs: 12, spanCount: 3 },
      ];
    },
    getTrace: async (): Promise<unknown> => {
      await delay();
      return {};
    },
    createSilence: async (): Promise<void> => {
      await delay();
    },
  },

  dashboards: {
    list: async (): Promise<GrafanaDashboard[]> => {
      await delay();
      return [
        { uid: "node-health", title: "Node Health", url: "#", category: "infrastructure" },
        { uid: "pod-resources", title: "Pod Resources", url: "#", category: "infrastructure" },
      ];
    },
  },

  logs: {
    query: async (): Promise<LogQueryResult> => {
      await delay();
      return { lines: [{ timestamp: "2026-03-09T12:00:00Z", labels: "{app=zenith-api}", message: "Request processed" }], executionTimeMs: 35 };
    },
    labels: async (): Promise<string[]> => {
      await delay();
      return ["app", "namespace", "pod"];
    },
  },

  alerts: {
    list: async (): Promise<Alert[]> => {
      await delay();
      return [];
    },
    stats: async (): Promise<AlertStats> => {
      await delay();
      return { firing: 0, pending: 0, resolvedToday: 3, totalRules: 12 };
    },
    rules: async (): Promise<AlertRule[]> => {
      await delay();
      return [
        { name: "HighCPU", group: "node", query: "cpu > 80%", duration: "5m", severity: "warning", state: "inactive" },
      ];
    },
  },

  traces: {
    search: async (): Promise<Trace[]> => {
      await delay();
      return [
        { traceId: "abc123", service: "zenith-api", operationName: "GET /health", durationMs: 12, spanCount: 3 },
      ];
    },
    get: async (): Promise<unknown> => {
      await delay();
      return {};
    },
  },

  security: {
    posture: async (): Promise<SecurityPosture> => {
      await delay();
      return {
        overallScore: 82, mfaAdoption: 65, imageVulns: { critical: 0, high: 2, medium: 5, low: 12 },
        policyViolations: 3, falcoAlerts: 0, certWarnings: 0, failedLogins24h: 4, openIssues: 5,
      };
    },
    overview: async (): Promise<SecurityOverview> => {
      await delay();
      return {
        score: 82, mfaAdoption: 65, vulnerabilities: 19, policyViolations: 3,
        failedLogins: 4, activeSessions: 12, activeApiKeys: 8, certsExpiringSoon: 0,
      };
    },
    policies: async (): Promise<KyvernoPolicy[]> => {
      await delay();
      return [
        { name: "require-labels", kind: "ClusterPolicy", action: "enforce", ready: true, violations: 0, updatedAt: "2026-03-01" },
        { name: "disallow-privileged", kind: "ClusterPolicy", action: "enforce", ready: true, violations: 2, updatedAt: "2026-03-01" },
      ];
    },
    wafStats: async (): Promise<WafStats> => {
      await delay();
      return { totalPolicies: 8, enforcing: 6, auditing: 2, totalViolations: 3 };
    },
    falcoAlerts: async (): Promise<unknown[]> => {
      await delay();
      return [];
    },
    rateLimits: async (): Promise<unknown> => {
      await delay();
      return { global: { limit: 100, window: "60s", scope: "per-ip" } };
    },
    images: async (): Promise<ImageScanResult[]> => {
      await delay();
      return [
        { repository: "zenith-api", tag: "0.7.24", scanStatus: "scanned", critical: 0, high: 0, medium: 2, low: 4 },
        { repository: "zenith-web", tag: "0.1.0", scanStatus: "scanned", critical: 0, high: 1, medium: 1, low: 3 },
      ];
    },
    imageStats: async (): Promise<ImageScanStats> => {
      await delay();
      return { totalImages: 5, cleanImages: 3, criticalCount: 0, highCount: 1 };
    },
    triggerScan: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
    sessions: async (): Promise<ActiveSession[]> => {
      await delay();
      return [
        { id: "s1", email: "admin@freezenith.com", ipAddress: "77.42.88.1", device: "Chrome / macOS", location: "Germany", lastSeen: "2m ago", mfaEnabled: true, isAdmin: true },
      ];
    },
    terminateSession: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
  },

  backups: {
    list: async (): Promise<BackupStatusData> => {
      await delay();
      return {
        veleroSchedules: [{ name: "daily-full", schedule: "0 2 * * *", lastBackup: "2026-03-09T02:00:00Z", lastStatus: "completed", backupCount: 14 }],
        cnpgBackups: [
          { cluster: "zenith-postgres", namespace: "zenith-staging", lastBackup: "2026-03-09T02:00:00Z", status: "healthy", walArchiving: "active", retentionDays: 14 },
          { cluster: "free-pg", namespace: "zenith-shared", lastBackup: "2026-03-09T02:00:00Z", status: "healthy", walArchiving: "active", retentionDays: 14 },
        ],
      };
    },
    stats: async (): Promise<BackupStats> => {
      await delay();
      return { veleroSchedules: 1, cnpgClusters: 2, lastBackup: "2026-03-09T02:00:00Z", totalSize: "2.4 GB" };
    },
    veleroSchedules: async (): Promise<VeleroSchedule[]> => {
      await delay();
      return [{ name: "daily-full", schedule: "0 2 * * *", lastBackup: "2026-03-09T02:00:00Z", lastStatus: "completed", retention: "14d", storageLocation: "s3://zenith-backups" }];
    },
    cnpgBackups: async (): Promise<CnpgBackup[]> => {
      await delay();
      return [
        { cluster: "zenith-postgres", namespace: "zenith-staging", schedule: "0 2 * * *", lastBackup: "2026-03-09T02:00:00Z", status: "healthy", s3Destination: "s3://zenith-backups/zenith-postgres-wal/" },
        { cluster: "free-pg", namespace: "zenith-shared", schedule: "0 2 * * *", lastBackup: "2026-03-09T02:00:00Z", status: "healthy", s3Destination: "s3://zenith-backups/free-pg-wal/" },
      ];
    },
    trigger: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
  },

  gitops: {
    apps: async (): Promise<ArgoAppItem[]> => {
      await delay();
      return [
        { name: "zenith-platform", namespace: "zenith-platform", status: "synced", health: "healthy", syncStatus: "Synced", lastSync: "2026-03-09T10:00:00Z" },
        { name: "zenith-monitoring", namespace: "monitoring", status: "synced", health: "healthy", syncStatus: "Synced", lastSync: "2026-03-09T10:00:00Z" },
      ];
    },
    list: async (): Promise<ArgoApp[]> => {
      await delay();
      return [
        { name: "zenith-platform", namespace: "zenith-platform", project: "default", healthStatus: "healthy", syncStatus: "Synced", lastSynced: "2026-03-09T10:00:00Z" },
      ];
    },
    stats: async (): Promise<GitOpsStats> => {
      await delay();
      return { totalApps: 4, synced: 4, outOfSync: 0, degraded: 0 };
    },
    sync: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
    history: async (): Promise<unknown[]> => {
      await delay();
      return [];
    },
  },

  registry: {
    projects: async (): Promise<HarborProject[]> => {
      await delay();
      return [
        { name: "zenith", repoCount: 5, storageUsed: 512000000, storageQuota: 10737418240, storageUsedDisplay: "512 MB", storageQuotaDisplay: "10 GB", access: "private", public: false, createdAt: "2026-01-15" },
        { name: "customer-images", repoCount: 0, storageUsed: 0, storageQuota: 5368709120, storageUsedDisplay: "0 B", storageQuotaDisplay: "5 GB", access: "private", public: false, createdAt: "2026-02-01" },
      ];
    },
    stats: async (): Promise<RegistryStats> => {
      await delay();
      return { totalProjects: 2, totalRepos: 5, totalTags: 24, storageUsed: "512 MB", storageQuota: "15 GB" };
    },
    repos: async (): Promise<RegistryRepo[]> => {
      await delay();
      return [
        { name: "zenith-api", tagCount: 8, pullCount: 120, pushTime: "2026-03-09T08:00:00Z" },
        { name: "zenith-web", tagCount: 5, pullCount: 80, pushTime: "2026-03-08T14:00:00Z" },
      ];
    },
  },

  adminUsers: {
    list: async (): Promise<AdminUser[]> => {
      await delay();
      return [
        { id: "au-1", email: "admin@freezenith.com", name: "Babak", role: "owner", mfaEnabled: true, permissionCount: 16 },
      ];
    },
    invite: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
    updateRole: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
    changeRole: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
    remove: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
  },

  quality: {
    metrics: async (): Promise<QualityMetrics> => {
      await delay();
      return {
        avgResponseTime: "14m", avgResolutionTime: "4.2h", openTickets: 3, resolvedThisWeek: 12,
        csat: 4.6, slaCompliance: 97.5, uptime: 99.95, errorRate: 0.02, p95Latency: "145ms",
        ticketsByPriority: { critical: 0, high: 1, medium: 1, low: 1 },
        ticketsByCategory: { billing: 1, technical: 1, general: 1 },
        weeklyTrend: [
          { week: "W1", opened: 4, resolved: 5 },
          { week: "W2", opened: 3, resolved: 4 },
          { week: "W3", opened: 5, resolved: 4 },
          { week: "W4", opened: 2, resolved: 3 },
        ],
      };
    },
    tickets: async (): Promise<QualityTicket[]> => {
      await delay();
      return [
        { id: "t1", subject: "Cannot deploy app", customer: "user@test.com", priority: "high", status: "open", age: "2h" },
        { id: "t2", subject: "SSL cert renewal", customer: "admin@acme.com", priority: "medium", status: "in-progress", age: "1d" },
      ];
    },
  },
};
