// Mock data for Mission Control - will be replaced with real API calls

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

export const mockClusters: Cluster[] = [
  {
    name: "zenith-shared",
    k8sVersion: "v1.30.2",
    nodes: 8,
    region: "fsn1",
    type: "shared",
    cpuPercent: 62,
    ramPercent: 58,
    pods: { used: 234, total: 500 },
    pvcs: { used: 89, total: 200 },
    status: "healthy",
  },
  {
    name: "pro-startup-a",
    k8sVersion: "v1.30.2",
    nodes: 4,
    region: "fsn1",
    type: "dedicated",
    tenant: "startup-a",
    cpuPercent: 45,
    ramPercent: 52,
    pods: { used: 67, total: 200 },
    pvcs: { used: 12, total: 50 },
    status: "healthy",
  },
  {
    name: "pro-enterprise",
    k8sVersion: "v1.28.6",
    nodes: 12,
    region: "nbg1",
    type: "dedicated",
    tenant: "acme-corp",
    cpuPercent: 71,
    ramPercent: 68,
    pods: { used: 312, total: 500 },
    pvcs: { used: 45, total: 100 },
    status: "warning",
    upgradeAvailable: "v1.30.2",
  },
];

export const mockModules: Module[] = [
  { name: "Zenith Operator", installed: "v1.2.1", latest: "v1.3.0", status: "update_available", description: "Core platform operator" },
  { name: "CloudNativePG", installed: "v1.22.1", latest: "v1.23.0", status: "update_available", description: "PostgreSQL operator" },
  { name: "Redis Operator", installed: "v7.2.0", latest: "v7.2.0", status: "up_to_date", description: "Redis operator" },
  { name: "cert-manager", installed: "v1.14.2", latest: "v1.14.2", status: "up_to_date", description: "SSL certificate management" },
  { name: "Traefik", installed: "v2.11.0", latest: "v2.11.0", status: "up_to_date", description: "Ingress controller" },
  { name: "Harbor", installed: "v2.10.0", latest: "v2.10.1", status: "update_available", description: "Container registry" },
  { name: "Keycloak Operator", installed: "v24.0.0", latest: "v24.0.0", status: "up_to_date", description: "Identity & access management" },
  { name: "Prometheus Stack", installed: "v56.2.0", latest: "v56.2.0", status: "up_to_date", description: "Monitoring & alerting" },
  { name: "Loki", installed: "v3.0.1", latest: "v3.0.1", status: "up_to_date", description: "Log aggregation" },
  { name: "NATS", installed: "v2.10.0", latest: "v2.10.0", status: "up_to_date", description: "Message queue & KV store" },
  { name: "Linkerd", installed: "v2.14.0", latest: "v2.14.1", status: "update_available", description: "Service mesh" },
];

export const mockTenants: Tenant[] = [
  { name: "my-startup", plan: "starter", apps: 12, databases: 3, cpuUsed: "2.4", cpuLimit: "4", ramUsed: "3.1", ramLimit: "4", status: "active" },
  { name: "acme-corp", plan: "pro", apps: 45, databases: 8, cpuUsed: "8.2", cpuLimit: "16", ramUsed: "12", ramLimit: "16", status: "active" },
  { name: "dev-agency", plan: "starter", apps: 3, databases: 1, cpuUsed: "0.5", cpuLimit: "4", ramUsed: "0.8", ramLimit: "4", status: "active" },
  { name: "test-project", plan: "starter", apps: 1, databases: 0, cpuUsed: "0.1", cpuLimit: "4", ramUsed: "0.2", ramLimit: "4", status: "idle" },
  { name: "enterprise-x", plan: "pro", apps: 87, databases: 12, cpuUsed: "22", cpuLimit: "32", ramUsed: "28", ramLimit: "32", status: "active" },
];

export const mockAuditLog: AuditEntry[] = [
  { time: "14:23", actor: "admin", action: "Upgraded CloudNativePG v1.21 → v1.22", cluster: "zenith-shared" },
  { time: "12:01", actor: "CAPI", action: "Scaled nodes 7 → 8", cluster: "zenith-shared" },
  { time: "09:45", actor: "system", action: "Tenant created: startup-x", cluster: "zenith-shared" },
  { time: "08:12", actor: "system", action: "Backup completed: all databases (47 tenants)" },
];

export const mockPlatformUpdate: PlatformUpdate = {
  version: "v1.3.0",
  current: "v1.2.1",
  releasedAt: "February 10, 2026",
  features: [
    "MongoDB support",
    "Cloud Connections (AWS/GCP/Azure VPN)",
    "GitOps mode (zen export/apply)",
    "Auto-generated documentation",
  ],
  breakingChanges: false,
};
