// Mock data for Zenith Web Platform

export interface App {
  name: string;
  status: "running" | "deploying" | "stopped" | "crashed";
  replicas: { ready: number; total: number };
  cpu: string;
  memory: string;
  domain?: string;
  source: string;
  branch: string;
  lastDeploy: string;
  port: number;
  avgResponse?: string;
  reqPerMin?: number;
}

export interface Database {
  name: string;
  engine: "postgresql" | "mysql" | "mongodb" | "redis";
  version: string;
  status: "running" | "creating" | "stopped";
  storageUsed: string;
  storageTotal: string;
  connections: { used: number; total: number };
  lastBackup?: string;
  linkedApps: string[];
}

export interface Domain {
  domain: string;
  app: string;
  ssl: boolean;
  status: "active" | "pending" | "error";
}

export interface Planet {
  name: string;
  size: string;
  cpuPercent: number;
  ramPercent: number;
  cpuCores: number;
  ramGb: number;
  status: "ready" | "joining" | "draining";
  region: string;
}

export interface StorageBucket {
  name: string;
  used: string;
  total: string;
  objects: number;
  status: "active" | "creating";
}

export interface EnvVar {
  key: string;
  value: string;
  sensitive: boolean;
}

export interface Deployment {
  id: string;
  commit: string;
  message: string;
  status: "live" | "building" | "failed" | "superseded";
  createdAt: string;
  duration: string;
}

// Auth Service (ZenAuth) data
export interface AuthRealm {
  name: string;
  users: number;
  clients: number;
  identityProviders: string[];
  sessions: number;
  status: "active" | "creating";
}

export interface AuthUser {
  id: string;
  email: string;
  name: string;
  realm: string;
  status: "active" | "suspended" | "pending";
  lastLogin: string;
  mfaEnabled: boolean;
}

export interface AuthClient {
  clientId: string;
  name: string;
  type: "public" | "confidential";
  protocol: "openid-connect" | "saml";
  realm: string;
  redirectUris: string[];
  enabled: boolean;
}

// Gateway (APISIX) data
export interface GatewayRoute {
  name: string;
  path: string;
  methods: string[];
  service: string;
  plugins: string[];
  status: "active" | "pending" | "error";
  reqPerMin: number;
  avgLatency: string;
}

export interface GatewayPlugin {
  name: string;
  scope: "global" | "route" | "service";
  appliedTo: string;
  enabled: boolean;
  config: string;
}

// IAM (Platform Access)
export interface ApiKey {
  name: string;
  prefix: string;
  created: string;
  lastUsed: string;
  scopes: string[];
}

export interface TeamMember {
  email: string;
  name: string;
  role: "Owner" | "Admin" | "Developer" | "Viewer";
  joined: string;
  lastActive: string;
}

// Registry (ECR-style)
export interface RegistryRepo {
  name: string;
  tags: { tag: string; digest: string; size: string; pushed: string; scanStatus: "passed" | "warning" | "failed" | "pending" }[];
  totalSize: string;
  lastPushed: string;
  scanEnabled: boolean;
  lifecyclePolicy: string;
}

// Monitoring
export interface GrafanaDashboard {
  name: string;
  type: "overview" | "service" | "infrastructure" | "custom";
  lastViewed: string;
  panels: number;
}

// ---------------------------------------------------------------------------
// Existing mock data
// ---------------------------------------------------------------------------

export const mockApps: App[] = [
  {
    name: "frontend",
    status: "running",
    replicas: { ready: 2, total: 2 },
    cpu: "120m",
    memory: "256Mi",
    domain: "app.startup.com",
    source: "github.com/startup/frontend",
    branch: "main",
    lastDeploy: "2 hours ago",
    port: 3000,
    avgResponse: "45ms",
    reqPerMin: 2140,
  },
  {
    name: "api-gateway",
    status: "running",
    replicas: { ready: 2, total: 2 },
    cpu: "340m",
    memory: "412Mi",
    domain: "api.startup.com",
    source: "github.com/startup/gateway",
    branch: "main",
    lastDeploy: "5 hours ago",
    port: 8080,
    avgResponse: "12ms",
    reqPerMin: 4280,
  },
  {
    name: "user-service",
    status: "running",
    replicas: { ready: 3, total: 3 },
    cpu: "245m",
    memory: "380Mi",
    source: "github.com/startup/user-svc",
    branch: "main",
    lastDeploy: "1 day ago",
    port: 8080,
    avgResponse: "23ms",
    reqPerMin: 1850,
  },
  {
    name: "order-service",
    status: "running",
    replicas: { ready: 2, total: 2 },
    cpu: "180m",
    memory: "290Mi",
    source: "github.com/startup/order-svc",
    branch: "main",
    lastDeploy: "3 days ago",
    port: 8080,
    avgResponse: "34ms",
    reqPerMin: 920,
  },
  {
    name: "payment-service",
    status: "running",
    replicas: { ready: 1, total: 1 },
    cpu: "90m",
    memory: "128Mi",
    source: "github.com/startup/payment-svc",
    branch: "main",
    lastDeploy: "1 week ago",
    port: 8080,
    avgResponse: "67ms",
    reqPerMin: 340,
  },
  {
    name: "notification-svc",
    status: "stopped",
    replicas: { ready: 0, total: 1 },
    cpu: "0m",
    memory: "0Mi",
    source: "github.com/startup/notify-svc",
    branch: "main",
    lastDeploy: "2 weeks ago",
    port: 8080,
  },
];

export const mockDatabases: Database[] = [
  {
    name: "users-db",
    engine: "postgresql",
    version: "16",
    status: "running",
    storageUsed: "2.1GB",
    storageTotal: "5GB",
    connections: { used: 23, total: 100 },
    lastBackup: "3 hours ago",
    linkedApps: ["user-service", "api-gateway"],
  },
  {
    name: "orders-db",
    engine: "postgresql",
    version: "16",
    status: "running",
    storageUsed: "4.8GB",
    storageTotal: "10GB",
    connections: { used: 18, total: 100 },
    lastBackup: "3 hours ago",
    linkedApps: ["order-service"],
  },
  {
    name: "cache",
    engine: "redis",
    version: "7",
    status: "running",
    storageUsed: "0.3GB",
    storageTotal: "2GB",
    connections: { used: 45, total: 1000 },
    linkedApps: ["api-gateway", "user-service", "order-service"],
  },
];

export const mockDomains: Domain[] = [
  { domain: "app.startup.com", app: "frontend", ssl: true, status: "active" },
  { domain: "api.startup.com", app: "api-gateway", ssl: true, status: "active" },
  { domain: "www.startup.com", app: "frontend", ssl: true, status: "active" },
];

export const mockPlanets: Planet[] = [
  { name: "planet-01", size: "CX22", cpuPercent: 78, ramPercent: 52, cpuCores: 2, ramGb: 4, status: "ready", region: "fsn1" },
  { name: "planet-02", size: "CX22", cpuPercent: 61, ramPercent: 63, cpuCores: 2, ramGb: 4, status: "ready", region: "fsn1" },
  { name: "planet-03", size: "CX22", cpuPercent: 45, ramPercent: 41, cpuCores: 2, ramGb: 4, status: "ready", region: "fsn1" },
  { name: "planet-04", size: "CX22", cpuPercent: 58, ramPercent: 55, cpuCores: 2, ramGb: 4, status: "ready", region: "fsn1" },
  { name: "planet-05", size: "CX22", cpuPercent: 32, ramPercent: 28, cpuCores: 2, ramGb: 4, status: "ready", region: "fsn1" },
];

export const mockStorage: StorageBucket[] = [
  { name: "uploads", used: "4.2GB", total: "10GB", objects: 12847, status: "active" },
  { name: "backups", used: "8.1GB", total: "50GB", objects: 342, status: "active" },
];

export const mockEnvVars: EnvVar[] = [
  { key: "DATABASE_URL", value: "postgres://app:***@users-db:5432/users", sensitive: true },
  { key: "REDIS_URL", value: "redis://cache:6379", sensitive: false },
  { key: "STRIPE_KEY", value: "sk_live_***", sensitive: true },
  { key: "LOG_LEVEL", value: "info", sensitive: false },
  { key: "NODE_ENV", value: "production", sensitive: false },
];

export const mockDeployments: Deployment[] = [
  { id: "d-1", commit: "abc1234", message: "fix: resolve auth token refresh", status: "live", createdAt: "2 hours ago", duration: "32s" },
  { id: "d-2", commit: "def5678", message: "feat: add user profile page", status: "superseded", createdAt: "5 hours ago", duration: "28s" },
  { id: "d-3", commit: "ghi9012", message: "chore: update dependencies", status: "superseded", createdAt: "1 day ago", duration: "45s" },
  { id: "d-4", commit: "jkl3456", message: "fix: memory leak in websocket handler", status: "superseded", createdAt: "2 days ago", duration: "31s" },
];

// ---------------------------------------------------------------------------
// Auth Service mock data
// ---------------------------------------------------------------------------

export const mockAuthRealms: AuthRealm[] = [
  {
    name: "production",
    users: 1247,
    clients: 8,
    identityProviders: ["Google", "GitHub"],
    sessions: 89,
    status: "active",
  },
  {
    name: "staging",
    users: 23,
    clients: 4,
    identityProviders: [],
    sessions: 3,
    status: "active",
  },
];

export const mockAuthUsers: AuthUser[] = [
  {
    id: "u-a1b2c3",
    email: "sarah.chen@startup.com",
    name: "Sarah Chen",
    realm: "production",
    status: "active",
    lastLogin: "12 minutes ago",
    mfaEnabled: true,
  },
  {
    id: "u-d4e5f6",
    email: "marcus.johnson@startup.com",
    name: "Marcus Johnson",
    realm: "production",
    status: "active",
    lastLogin: "3 hours ago",
    mfaEnabled: true,
  },
  {
    id: "u-g7h8i9",
    email: "elena.rodriguez@startup.com",
    name: "Elena Rodriguez",
    realm: "production",
    status: "active",
    lastLogin: "1 day ago",
    mfaEnabled: false,
  },
  {
    id: "u-j0k1l2",
    email: "james.wilson@startup.com",
    name: "James Wilson",
    realm: "production",
    status: "suspended",
    lastLogin: "2 weeks ago",
    mfaEnabled: false,
  },
  {
    id: "u-m3n4o5",
    email: "priya.patel@startup.com",
    name: "Priya Patel",
    realm: "production",
    status: "active",
    lastLogin: "6 hours ago",
    mfaEnabled: true,
  },
  {
    id: "u-p6q7r8",
    email: "tom.nguyen@startup.com",
    name: "Tom Nguyen",
    realm: "production",
    status: "pending",
    lastLogin: "Never",
    mfaEnabled: false,
  },
];

export const mockAuthClients: AuthClient[] = [
  {
    clientId: "web-app",
    name: "Web Application",
    type: "public",
    protocol: "openid-connect",
    realm: "production",
    redirectUris: ["https://app.startup.com/callback", "https://app.startup.com/silent-renew"],
    enabled: true,
  },
  {
    clientId: "mobile-app",
    name: "Mobile Application",
    type: "public",
    protocol: "openid-connect",
    realm: "production",
    redirectUris: ["com.startup.mobile://callback"],
    enabled: true,
  },
  {
    clientId: "admin-panel",
    name: "Admin Panel",
    type: "confidential",
    protocol: "openid-connect",
    realm: "production",
    redirectUris: ["https://admin.startup.com/callback"],
    enabled: true,
  },
  {
    clientId: "partner-api",
    name: "Partner API",
    type: "confidential",
    protocol: "openid-connect",
    realm: "production",
    redirectUris: ["https://partners.startup.com/oauth/callback"],
    enabled: true,
  },
  {
    clientId: "legacy-saml-app",
    name: "Legacy SAML App",
    type: "confidential",
    protocol: "saml",
    realm: "production",
    redirectUris: ["https://legacy.startup.com/saml/acs"],
    enabled: false,
  },
];

// ---------------------------------------------------------------------------
// Gateway (APISIX) mock data
// ---------------------------------------------------------------------------

export const mockGatewayRoutes: GatewayRoute[] = [
  {
    name: "users-route",
    path: "/api/v1/users",
    methods: ["GET", "POST", "PUT", "DELETE"],
    service: "user-service",
    plugins: ["jwt-auth", "rate-limiting", "cors"],
    status: "active",
    reqPerMin: 1850,
    avgLatency: "23ms",
  },
  {
    name: "orders-route",
    path: "/api/v1/orders",
    methods: ["GET", "POST", "PUT"],
    service: "order-service",
    plugins: ["jwt-auth", "rate-limiting", "cors"],
    status: "active",
    reqPerMin: 920,
    avgLatency: "34ms",
  },
  {
    name: "payments-route",
    path: "/api/v1/payments",
    methods: ["POST", "GET"],
    service: "payment-service",
    plugins: ["jwt-auth", "rate-limiting", "cors", "request-transformer"],
    status: "active",
    reqPerMin: 340,
    avgLatency: "67ms",
  },
  {
    name: "auth-route",
    path: "/api/v1/auth",
    methods: ["GET", "POST"],
    service: "auth-service",
    plugins: ["rate-limiting", "cors"],
    status: "active",
    reqPerMin: 2600,
    avgLatency: "18ms",
  },
  {
    name: "notifications-route",
    path: "/api/v1/notifications",
    methods: ["GET", "POST"],
    service: "notification-svc",
    plugins: ["jwt-auth", "cors"],
    status: "error",
    reqPerMin: 0,
    avgLatency: "--",
  },
  {
    name: "frontend-catchall",
    path: "/",
    methods: ["GET"],
    service: "frontend",
    plugins: ["cors"],
    status: "active",
    reqPerMin: 2140,
    avgLatency: "45ms",
  },
  {
    name: "websocket-route",
    path: "/ws",
    methods: ["GET"],
    service: "frontend",
    plugins: ["cors"],
    status: "active",
    reqPerMin: 410,
    avgLatency: "8ms",
  },
];

export const mockGatewayPlugins: GatewayPlugin[] = [
  {
    name: "jwt-auth",
    scope: "global",
    appliedTo: "All routes",
    enabled: true,
    config: "algorithm: RS256, cookie_names: [session]",
  },
  {
    name: "rate-limiting",
    scope: "global",
    appliedTo: "All routes",
    enabled: true,
    config: "limit: 1000 req/min, policy: redis",
  },
  {
    name: "cors",
    scope: "global",
    appliedTo: "All routes",
    enabled: true,
    config: "origins: [app.startup.com, admin.startup.com], credentials: true",
  },
  {
    name: "request-transformer",
    scope: "route",
    appliedTo: "/api/v1/payments",
    enabled: true,
    config: "add.headers: [X-Payment-Version:2, X-Idempotency-Check:true]",
  },
  {
    name: "ip-restriction",
    scope: "route",
    appliedTo: "/admin",
    enabled: true,
    config: "allow: [10.0.0.0/8, 172.16.0.0/12], status: 403",
  },
  {
    name: "response-ratelimiting",
    scope: "service",
    appliedTo: "payment-service",
    enabled: false,
    config: "limits.sms_notifications: minute=5",
  },
];

// ---------------------------------------------------------------------------
// IAM (Platform Access) mock data
// ---------------------------------------------------------------------------

export const mockApiKeys: ApiKey[] = [
  {
    name: "Production API",
    prefix: "zn_live_8kT3",
    created: "2025-11-02",
    lastUsed: "3 minutes ago",
    scopes: ["apps:read", "apps:write", "databases:read", "deployments:write"],
  },
  {
    name: "CI/CD Pipeline",
    prefix: "zn_live_mR7x",
    created: "2025-12-15",
    lastUsed: "1 hour ago",
    scopes: ["apps:write", "deployments:write", "registry:push"],
  },
  {
    name: "Monitoring",
    prefix: "zn_live_qP2w",
    created: "2026-01-08",
    lastUsed: "5 minutes ago",
    scopes: ["apps:read", "databases:read", "monitoring:read"],
  },
];

export const mockTeamMembers: TeamMember[] = [
  {
    email: "sarah.chen@startup.com",
    name: "Sarah Chen",
    role: "Owner",
    joined: "2025-09-14",
    lastActive: "12 minutes ago",
  },
  {
    email: "marcus.johnson@startup.com",
    name: "Marcus Johnson",
    role: "Admin",
    joined: "2025-10-01",
    lastActive: "3 hours ago",
  },
  {
    email: "elena.rodriguez@startup.com",
    name: "Elena Rodriguez",
    role: "Developer",
    joined: "2025-11-20",
    lastActive: "1 day ago",
  },
  {
    email: "tom.nguyen@startup.com",
    name: "Tom Nguyen",
    role: "Viewer",
    joined: "2026-01-05",
    lastActive: "2 days ago",
  },
];

// ---------------------------------------------------------------------------
// Registry (ECR-style) mock data
// ---------------------------------------------------------------------------

export const mockRegistryRepos: RegistryRepo[] = [
  {
    name: "frontend",
    tags: [
      { tag: "latest", digest: "sha256:a3f8d2c1e9b4...", size: "142MB", pushed: "2 hours ago", scanStatus: "passed" },
      { tag: "v2.4.1", digest: "sha256:a3f8d2c1e9b4...", size: "142MB", pushed: "2 hours ago", scanStatus: "passed" },
      { tag: "v2.4.0", digest: "sha256:7e1b9f3c0a52...", size: "140MB", pushed: "3 days ago", scanStatus: "passed" },
      { tag: "v2.3.9", digest: "sha256:d4c8a2e6f1b3...", size: "139MB", pushed: "1 week ago", scanStatus: "warning" },
    ],
    totalSize: "421MB",
    lastPushed: "2 hours ago",
    scanEnabled: true,
    lifecyclePolicy: "Keep last 10 tags",
  },
  {
    name: "user-service",
    tags: [
      { tag: "latest", digest: "sha256:b5d1e7f9c2a8...", size: "98MB", pushed: "1 day ago", scanStatus: "passed" },
      { tag: "v1.12.0", digest: "sha256:b5d1e7f9c2a8...", size: "98MB", pushed: "1 day ago", scanStatus: "passed" },
      { tag: "v1.11.3", digest: "sha256:f2a9c7d4e8b1...", size: "97MB", pushed: "5 days ago", scanStatus: "passed" },
    ],
    totalSize: "293MB",
    lastPushed: "1 day ago",
    scanEnabled: true,
    lifecyclePolicy: "Keep last 10 tags",
  },
  {
    name: "order-service",
    tags: [
      { tag: "latest", digest: "sha256:c9e3f1a7d2b6...", size: "105MB", pushed: "3 days ago", scanStatus: "passed" },
      { tag: "v1.8.2", digest: "sha256:c9e3f1a7d2b6...", size: "105MB", pushed: "3 days ago", scanStatus: "passed" },
      { tag: "v1.8.1", digest: "sha256:e6b4d8a1f3c9...", size: "104MB", pushed: "1 week ago", scanStatus: "failed" },
      { tag: "v1.8.0", digest: "sha256:a1c3e5d7f9b2...", size: "103MB", pushed: "2 weeks ago", scanStatus: "passed" },
    ],
    totalSize: "417MB",
    lastPushed: "3 days ago",
    scanEnabled: true,
    lifecyclePolicy: "Keep last 5 tags, expire untagged after 7 days",
  },
];

// ---------------------------------------------------------------------------
// Monitoring (Grafana) mock data
// ---------------------------------------------------------------------------

export const mockGrafanaDashboards: GrafanaDashboard[] = [
  {
    name: "Platform Overview",
    type: "overview",
    lastViewed: "10 minutes ago",
    panels: 12,
  },
  {
    name: "Service Health",
    type: "service",
    lastViewed: "1 hour ago",
    panels: 8,
  },
  {
    name: "Node Metrics",
    type: "infrastructure",
    lastViewed: "4 hours ago",
    panels: 10,
  },
  {
    name: "Custom: API Latency",
    type: "custom",
    lastViewed: "2 days ago",
    panels: 6,
  },
];

// ---------------------------------------------------------------------------
// Project-level exports
// ---------------------------------------------------------------------------

export const projectName = "my-startup";
export const projectPlan = "Starter";
