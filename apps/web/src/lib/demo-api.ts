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
  DeployApp,
  Deployment,
  EnvVar,
  Secret,
  Release,
  AppDatabase,
  AppAuthConfig,
  AppAuthUser,
  AppBucket,
  DatabaseBackup,
  UserPlanResponse,
  CustomDomain,
  APIKey,
  Session,
  MFAStatus,
  MFAEnableResponse,
  UserWebhook,
  WebhookDelivery,
  CustomRole,
  IPWhitelistEntry,
  ComplianceResponse,
  DPARecord,
  BrandingConfig,
  SSOConfig,
  PreviewDeployment,
  BillingStatus,
  InvoiceRecord,
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

// Mock deploy engine apps
const mockDeployApps: DeployApp[] = [
  {
    id: "da-1",
    user_id: "demo-user",
    name: "my-next-app",
    repo_url: "https://github.com/demo/my-next-app",
    branch: "main",
    framework: "nextjs",
    status: "running",
    subdomain: "my-next-app",
    port: 3000,
    url: "https://my-next-app.freezenith.com",
    created_at: "2026-02-10T08:00:00Z",
    updated_at: "2026-02-20T14:30:00Z",
  },
  {
    id: "da-2",
    user_id: "demo-user",
    name: "go-api",
    repo_url: "https://github.com/demo/go-api",
    branch: "main",
    framework: "go",
    status: "running",
    subdomain: "go-api",
    port: 8080,
    url: "https://go-api.freezenith.com",
    created_at: "2026-02-12T10:00:00Z",
    updated_at: "2026-02-19T09:15:00Z",
  },
  {
    id: "da-3",
    user_id: "demo-user",
    name: "flask-ml",
    repo_url: "https://github.com/demo/flask-ml",
    branch: "develop",
    framework: "flask",
    status: "building",
    subdomain: "flask-ml",
    port: 5000,
    url: "",
    created_at: "2026-02-21T12:00:00Z",
    updated_at: "2026-02-21T12:00:00Z",
  },
];

export const demoAppsDeploy = {
  list: async (): Promise<{ items: DeployApp[]; total: number }> => {
    await delay();
    return { items: mockDeployApps, total: mockDeployApps.length };
  },
  get: async (id: string): Promise<DeployApp> => {
    await delay();
    const app = mockDeployApps.find((a) => a.id === id);
    if (!app) throw new Error(`Deploy app "${id}" not found`);
    return app;
  },
  create: async (): Promise<DeployApp> => {
    throw new Error("Not available in demo mode");
  },
  delete: async (): Promise<void> => {
    throw new Error("Not available in demo mode");
  },
  listDeployments: async (): Promise<{ items: Deployment[]; total: number }> => {
    await delay();
    return { items: [], total: 0 };
  },
  getDeployment: async (): Promise<Deployment> => {
    throw new Error("Not available in demo mode");
  },
  rollback: async (): Promise<{ message: string }> => {
    throw new Error("Not available in demo mode");
  },
  getEnvVars: async (): Promise<{ items: EnvVar[]; total: number }> => {
    await delay();
    return { items: [], total: 0 };
  },
  setEnvVars: async (): Promise<{ message: string }> => {
    throw new Error("Not available in demo mode");
  },
  deleteEnvVar: async (): Promise<void> => {
    throw new Error("Not available in demo mode");
  },

  // Secrets mock
  listSecrets: async (appId: string): Promise<{ secrets: Secret[] }> => {
    await delay();
    return {
      secrets: [
        { id: "s-1", app_id: appId, key: "DATABASE_URL", created_at: "2026-02-15T10:00:00Z" },
        { id: "s-2", app_id: appId, key: "API_KEY", created_at: "2026-02-16T12:00:00Z" },
        { id: "s-3", app_id: appId, key: "JWT_SECRET", created_at: "2026-02-18T08:30:00Z" },
      ],
    };
  },
  getSecretValue: async (_appId: string, key: string): Promise<{ key: string; value: string }> => {
    await delay();
    const mockValues: Record<string, string> = {
      DATABASE_URL: "postgres://admin:s3cret@db.internal:5432/myapp",
      API_KEY: "sk_live_abc123def456ghi789",
      JWT_SECRET: "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6",
    };
    return { key, value: mockValues[key] || "mock-value" };
  },
  setSecret: async (): Promise<{ key: string; status: string }> => {
    throw new Error("Not available in demo mode");
  },
  deleteSecret: async (): Promise<{ key: string; status: string }> => {
    throw new Error("Not available in demo mode");
  },

  // Releases mock
  listReleases: async (appId: string): Promise<{ releases: Release[] }> => {
    await delay();
    return {
      releases: [
        {
          id: "rel-1",
          app_id: appId,
          image: "ghcr.io/demo/my-next-app:a1b2c3d",
          git_sha: "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0",
          branch: "main",
          message: "feat: add user dashboard",
          created_at: "2026-02-20T14:30:00Z",
        },
        {
          id: "rel-2",
          app_id: appId,
          image: "ghcr.io/demo/my-next-app:e4f5g6h",
          git_sha: "e4f5g6h7i8j9k0l1m2n3o4p5q6r7s8t9u0v1w2x3",
          branch: "main",
          message: "fix: resolve login redirect loop",
          created_at: "2026-02-19T09:15:00Z",
        },
        {
          id: "rel-3",
          app_id: appId,
          image: "ghcr.io/demo/my-next-app:b7c8d9e",
          git_sha: "b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6",
          branch: "main",
          message: "chore: upgrade dependencies",
          created_at: "2026-02-18T16:00:00Z",
        },
        {
          id: "rel-4",
          app_id: appId,
          image: "ghcr.io/demo/my-next-app:f0a1b2c",
          git_sha: "f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9",
          branch: "main",
          message: "feat: initial release",
          created_at: "2026-02-17T11:30:00Z",
        },
      ],
    };
  },
  deployRelease: async (): Promise<{ deployment_id: string; release_id: string; image: string; status: string }> => {
    throw new Error("Not available in demo mode");
  },

  // Per-app databases (Phase 3)
  listAppDatabases: async (appId: string): Promise<AppDatabase[]> => {
    await delay();
    const mockAppDbs: Record<string, AppDatabase[]> = {
      "da-1": [
        {
          id: "db-1",
          app_id: "da-1",
          name: "db-my-next",
          engine: "postgresql",
          host: "localhost",
          port: 5432,
          db_name: "z_demo_my_next",
          db_user: "u_db1user",
          connection_string: "postgresql://u_db1user:***@localhost:5432/z_demo_my_next?sslmode=disable",
          size_mb: 45,
          max_size_mb: 500,
          status: "ready",
          created_at: "2026-02-15T10:00:00Z",
        },
      ],
      "da-2": [
        {
          id: "db-2",
          app_id: "da-2",
          name: "db-go-api",
          engine: "postgresql",
          host: "localhost",
          port: 5432,
          db_name: "z_demo_go_api",
          db_user: "u_db2user",
          size_mb: 120,
          max_size_mb: 500,
          status: "ready",
          created_at: "2026-02-16T12:00:00Z",
        },
        {
          id: "db-3",
          app_id: "da-2",
          name: "cache-go-api",
          engine: "redis",
          host: "localhost",
          port: 6379,
          db_name: "0",
          db_user: "",
          size_mb: 8,
          max_size_mb: 500,
          status: "ready",
          created_at: "2026-02-17T08:00:00Z",
        },
      ],
    };
    return mockAppDbs[appId] || [];
  },
  getAppDatabase: async (_appId: string, dbId: string): Promise<AppDatabase> => {
    await delay();
    const all: AppDatabase[] = [
      {
        id: "db-1",
        app_id: "da-1",
        name: "db-my-next",
        engine: "postgresql",
        host: "localhost",
        port: 5432,
        db_name: "z_demo_my_next",
        db_user: "u_db1user",
        connection_string: "postgresql://u_db1user:s3cr3t@localhost:5432/z_demo_my_next?sslmode=disable",
        size_mb: 45,
        max_size_mb: 500,
        status: "ready",
        created_at: "2026-02-15T10:00:00Z",
      },
    ];
    const db = all.find((d) => d.id === dbId);
    if (!db) throw new Error("Database not found");
    return db;
  },
  createAppDatabase: async (): Promise<AppDatabase> => {
    throw new Error("Not available in demo mode");
  },
  deleteAppDatabase: async (): Promise<{ message: string }> => {
    throw new Error("Not available in demo mode");
  },

  // Custom Domains mock (Phase 4)
  listDomains: async (appId: string): Promise<CustomDomain[]> => {
    await delay();
    if (appId === "da-1") {
      return [
        { id: "dom-1", app_id: "da-1", domain: "myapp.example.com", status: "active", tls_ready: true, created_at: "2026-02-18T10:00:00Z" },
      ];
    }
    return [];
  },
  addDomain: async (): Promise<CustomDomain> => {
    throw new Error("Not available in demo mode");
  },
  deleteDomain: async (): Promise<{ message: string }> => {
    throw new Error("Not available in demo mode");
  },

  // Database Backups mock (Phase 3)
  listBackups: async (_appId: string, dbId: string): Promise<DatabaseBackup[]> => {
    await delay();
    if (dbId === "db-1") {
      return [
        { id: "bak-1", database_id: "db-1", type: "manual", status: "completed", size_mb: 12, created_at: "2026-02-20T02:00:00Z" },
        { id: "bak-2", database_id: "db-1", type: "scheduled", status: "completed", size_mb: 11, created_at: "2026-02-19T02:00:00Z" },
        { id: "bak-3", database_id: "db-1", type: "scheduled", status: "completed", size_mb: 10, created_at: "2026-02-18T02:00:00Z" },
      ];
    }
    return [];
  },
  createBackup: async (): Promise<DatabaseBackup> => {
    throw new Error("Not available in demo mode");
  },
  deleteBackup: async (): Promise<{ message: string }> => {
    throw new Error("Not available in demo mode");
  },
  restoreBackup: async (): Promise<{ message: string; backup_id: string; database_id: string }> => {
    throw new Error("Not available in demo mode");
  },

  // Storage mock (Phase 3)
  listAppBuckets: async (appId: string): Promise<AppBucket[]> => {
    await delay();
    if (appId === "da-1") {
      return [
        {
          id: "bkt-1",
          app_id: "da-1",
          name: "uploads",
          access: "private",
          region: "fsn1",
          size_mb: 128,
          max_size_mb: 1024,
          objects: 342,
          status: "active",
          endpoint: "https://uploads.s3.zenith.local",
          created_at: "2026-02-15T10:00:00Z",
        },
      ];
    }
    return [];
  },
  createAppBucket: async (): Promise<AppBucket> => {
    throw new Error("Not available in demo mode");
  },
  deleteAppBucket: async (): Promise<{ message: string }> => {
    throw new Error("Not available in demo mode");
  },

  // App Auth mock (Phase 3)
  getAuthStatus: async (appId: string): Promise<AppAuthConfig> => {
    await delay();
    // da-1 has auth enabled, others don't
    if (appId === "da-1") {
      return { enabled: true, user_count: 42, max_users: 1000 };
    }
    return { enabled: false, user_count: 0, max_users: 0 };
  },
  enableAuth: async (): Promise<AppAuthConfig> => {
    throw new Error("Not available in demo mode");
  },
  disableAuth: async (): Promise<{ message: string }> => {
    throw new Error("Not available in demo mode");
  },
  listAuthUsers: async (appId: string): Promise<{ users: AppAuthUser[]; total: number }> => {
    await delay();
    if (appId === "da-1") {
      return {
        users: [
          { id: "au-1", email: "alice@example.com", name: "Alice", verified: true, created_at: "2026-02-10T08:00:00Z" },
          { id: "au-2", email: "bob@example.com", name: "Bob", verified: true, created_at: "2026-02-12T10:00:00Z" },
          { id: "au-3", email: "carol@example.com", name: "Carol", verified: false, created_at: "2026-02-20T14:00:00Z" },
        ],
        total: 42,
      };
    }
    return { users: [], total: 0 };
  },
  deleteAuthUser: async (): Promise<{ message: string }> => {
    throw new Error("Not available in demo mode");
  },
};

export const demoUserDatabases = {
  list: async (): Promise<AppDatabase[]> => {
    await delay();
    return [
      {
        id: "db-1",
        app_id: "da-1",
        name: "db-my-next",
        engine: "postgresql",
        host: "localhost",
        port: 5432,
        db_name: "z_demo_my_next",
        db_user: "u_db1user",
        size_mb: 45,
        max_size_mb: 500,
        status: "ready",
        created_at: "2026-02-15T10:00:00Z",
      },
      {
        id: "db-2",
        app_id: "da-2",
        name: "db-go-api",
        engine: "postgresql",
        host: "localhost",
        port: 5432,
        db_name: "z_demo_go_api",
        db_user: "u_db2user",
        size_mb: 120,
        max_size_mb: 500,
        status: "ready",
        created_at: "2026-02-16T12:00:00Z",
      },
      {
        id: "db-3",
        app_id: "da-2",
        name: "cache-go-api",
        engine: "redis",
        host: "localhost",
        port: 6379,
        db_name: "0",
        db_user: "",
        size_mb: 8,
        max_size_mb: 500,
        status: "ready",
        created_at: "2026-02-17T08:00:00Z",
      },
    ];
  },
};

export const demoUserPlan = {
  get: async (): Promise<UserPlanResponse> => {
    await delay();
    return {
      tier: "pro",
      limits: {
        max_apps: 5, max_databases: 3, max_db_size_mb: 5120,
        max_auth_users: 10000, max_storage_mb: 10240, max_buckets: 5,
        max_cpu_millis: 2000, max_ram_mb: 2048, max_team_members: 3,
        backups_enabled: true, custom_domain: true, always_on: true, sleep_after_mins: 0,
      },
      usage: {
        apps: 3, databases: 3, storage_mb: 128, auth_users: 42, buckets: 1,
      },
    };
  },
  upgrade: async (): Promise<UserPlanResponse> => {
    throw new Error("Not available in demo mode");
  },
};

export const demoAPIKeys = {
  list: async (): Promise<{ items: APIKey[]; total: number }> => {
    await delay();
    return {
      items: [
        { id: "ak-1", name: "CI Pipeline", key_prefix: "zk_a1b2c3", scopes: ["deploy"], user_id: "demo-user", last_used_at: "2026-02-21T10:00:00Z", created_at: "2026-02-01T10:00:00Z" },
        { id: "ak-2", name: "Read-only Monitor", key_prefix: "zk_d4e5f6", scopes: ["read"], user_id: "demo-user", created_at: "2026-02-15T08:00:00Z" },
      ],
      total: 2,
    };
  },
  create: async (): Promise<APIKey> => {
    throw new Error("Not available in demo mode");
  },
  delete: async (): Promise<{ message: string }> => {
    throw new Error("Not available in demo mode");
  },
};

export const demoSessions = {
  list: async (): Promise<{ items: Session[]; total: number }> => {
    await delay();
    return {
      items: [
        { id: "ses-1", user_id: "demo-user", ip_address: "192.168.1.42", user_agent: "Mozilla/5.0 Chrome/120", device: "Desktop", current: true, created_at: "2026-02-22T08:00:00Z", expires_at: "2026-02-23T08:00:00Z", last_seen_at: "2026-02-22T16:00:00Z" },
        { id: "ses-2", user_id: "demo-user", ip_address: "10.0.0.5", user_agent: "Mozilla/5.0 Mobile Safari", device: "Mobile", current: false, created_at: "2026-02-20T10:00:00Z", expires_at: "2026-02-21T10:00:00Z", last_seen_at: "2026-02-20T14:00:00Z" },
      ],
      total: 2,
    };
  },
  revoke: async (): Promise<{ message: string }> => {
    throw new Error("Not available in demo mode");
  },
  revokeAll: async (): Promise<{ message: string }> => {
    throw new Error("Not available in demo mode");
  },
};

// ---- MFA Demo ----

export const demoMFA = {
  getStatus: async (): Promise<MFAStatus> => {
    await delay();
    return { status: "disabled", backup_codes: 0 };
  },
  enable: async (): Promise<MFAEnableResponse> => {
    await delay();
    return {
      secret: "JBSWY3DPEHPK3PXP",
      otpauth_uri: "otpauth://totp/Zenith:demo@zenith.dev?secret=JBSWY3DPEHPK3PXP&issuer=Zenith&digits=6&period=30",
      backup_codes: ["A1B2C3D4", "E5F6G7H8", "I9J0K1L2", "M3N4O5P6", "Q7R8S9T0", "U1V2W3X4", "Y5Z6A7B8", "C9D0E1F2", "G3H4I5J6", "K7L8M9N0"],
    };
  },
  verify: async (): Promise<{ status: string; enabled_at: string }> => {
    await delay();
    return { status: "enabled", enabled_at: new Date().toISOString() };
  },
  disable: async (): Promise<{ status: string }> => {
    await delay();
    return { status: "disabled" };
  },
  regenerateBackupCodes: async (): Promise<{ backup_codes: string[] }> => {
    await delay();
    return { backup_codes: ["NEW1CODE", "NEW2CODE", "NEW3CODE", "NEW4CODE", "NEW5CODE", "NEW6CODE", "NEW7CODE", "NEW8CODE", "NEW9CODE", "NEW0CODE"] };
  },
};

// ---- Webhooks Demo ----

export const demoWebhooks = {
  list: async (): Promise<{ items: UserWebhook[] }> => {
    await delay();
    return {
      items: [
        { id: "wh-1", user_id: "demo-user", url: "https://hooks.slack.com/services/T00/B00/xxx", events: ["deploy.success", "deploy.failed"], active: true, created_at: "2026-02-10T10:00:00Z", updated_at: "2026-02-10T10:00:00Z" },
        { id: "wh-2", user_id: "demo-user", url: "https://api.example.com/webhooks", events: ["app.sleeping", "limit.reached"], active: false, created_at: "2026-02-15T08:00:00Z", updated_at: "2026-02-18T12:00:00Z" },
      ],
    };
  },
  create: async (): Promise<UserWebhook> => {
    throw new Error("Not available in demo mode");
  },
  update: async (): Promise<UserWebhook> => {
    throw new Error("Not available in demo mode");
  },
  delete: async (): Promise<void> => {
    throw new Error("Not available in demo mode");
  },
  listDeliveries: async (): Promise<{ items: WebhookDelivery[] }> => {
    await delay();
    return {
      items: [
        { id: "wd-1", webhook_id: "wh-1", event: "deploy.success", payload: '{"app":"my-app","status":"running"}', status: "success", status_code: 200, attempts: 1, created_at: "2026-02-22T14:00:00Z" },
        { id: "wd-2", webhook_id: "wh-1", event: "deploy.failed", payload: '{"app":"my-app","error":"build failed"}', status: "failed", status_code: 500, error: "Internal Server Error", attempts: 3, created_at: "2026-02-21T16:00:00Z" },
      ],
    };
  },
};

// ---- Custom Roles Demo ----

export const demoRoles = {
  list: async (): Promise<{ items: CustomRole[] }> => {
    await delay();
    return {
      items: [
        { id: "role-1", user_id: "demo-user", name: "Developer", description: "Can deploy and view logs", permissions: ["deploy", "view_logs"], created_at: "2026-02-10T10:00:00Z", updated_at: "2026-02-10T10:00:00Z" },
        { id: "role-2", user_id: "demo-user", name: "DB Admin", description: "Full database management", permissions: ["manage_db", "view_logs"], created_at: "2026-02-15T08:00:00Z", updated_at: "2026-02-15T08:00:00Z" },
      ],
    };
  },
  create: async (): Promise<CustomRole> => { throw new Error("Not available in demo mode"); },
  update: async (): Promise<CustomRole> => { throw new Error("Not available in demo mode"); },
  delete: async (): Promise<void> => { throw new Error("Not available in demo mode"); },
  listPermissions: async (): Promise<{ permissions: string[] }> => {
    await delay();
    return { permissions: ["deploy", "view_logs", "manage_db", "manage_team", "manage_billing", "admin"] };
  },
};

// ---- IP Whitelist Demo ----

export const demoIPWhitelist = {
  list: async (): Promise<{ items: IPWhitelistEntry[] }> => {
    await delay();
    return {
      items: [
        { id: "ip-1", user_id: "demo-user", cidr: "10.0.0.0/8", description: "Internal network", created_at: "2026-02-10T10:00:00Z" },
        { id: "ip-2", user_id: "demo-user", cidr: "192.168.1.42/32", description: "Office IP", created_at: "2026-02-12T14:00:00Z" },
      ],
    };
  },
  add: async (): Promise<IPWhitelistEntry> => { throw new Error("Not available in demo mode"); },
  delete: async (): Promise<void> => { throw new Error("Not available in demo mode"); },
};

// ---- Compliance Demo ----

export const demoCompliance = {
  getStatus: async (): Promise<ComplianceResponse> => {
    await delay();
    return {
      checks: [
        { category: "Authentication", item: "Multi-Factor Authentication (MFA)", status: "fail", description: "Two-factor authentication is enabled for your account" },
        { category: "Encryption", item: "Encryption at Rest", status: "pass", description: "All data is encrypted at rest using AES-256-GCM" },
        { category: "Encryption", item: "Encryption in Transit", status: "pass", description: "All API and dashboard traffic uses TLS 1.3" },
        { category: "Audit", item: "Audit Logging", status: "pass", description: "All administrative actions are logged with actor and timestamp" },
        { category: "Access Control", item: "IP Whitelisting", status: "na", description: "Dashboard and API access restricted to allowed IP ranges" },
        { category: "GDPR", item: "Right to Deletion", status: "pass", description: "Users can delete their account and all associated data" },
        { category: "GDPR", item: "Data Processing Agreement", status: "na", description: "DPA available for Team and Enterprise plans" },
        { category: "Authentication", item: "Single Sign-On (SSO)", status: "na", description: "SAML 2.0 and OIDC SSO available for Team plans and above" },
      ],
      summary: { total: 8, pass: 4, fail: 1, partial: 0, na: 3 },
    };
  },
};

// ---- DPA Demo ----

export const demoDPA = {
  get: async (): Promise<DPARecord> => {
    await delay();
    return { user_id: "demo-user", status: "unsigned" };
  },
  sign: async (): Promise<DPARecord> => {
    throw new Error("Not available in demo mode");
  },
};

// ---- Branding Demo ----

export const demoBranding = {
  get: async (): Promise<BrandingConfig> => {
    await delay();
    return { user_id: "demo-user", company_name: "", logo_url: "", primary_color: "", domain_verified: false, hide_branding: false, updated_at: "" };
  },
  update: async (): Promise<BrandingConfig> => {
    throw new Error("Not available in demo mode");
  },
  setDomain: async (): Promise<BrandingConfig> => {
    throw new Error("Not available in demo mode");
  },
};

// ---- SSO Demo ----

export const demoSSO = {
  list: async (): Promise<{ items: SSOConfig[] }> => {
    await delay();
    return {
      items: [
        { id: "sso-1", user_id: "demo-user", provider: "saml", entity_id: "https://idp.example.com/metadata", sso_url: "https://idp.example.com/sso", certificate: "MIIDpDCCAo...", enabled: true, created_at: "2026-02-10T10:00:00Z", updated_at: "2026-02-10T10:00:00Z" },
      ],
    };
  },
  configureSAML: async (): Promise<SSOConfig> => { throw new Error("Not available in demo mode"); },
  configureOIDC: async (): Promise<SSOConfig> => { throw new Error("Not available in demo mode"); },
  delete: async (): Promise<void> => { throw new Error("Not available in demo mode"); },
};

// ---- Preview Deployments Demo ----

export const demoPreviews = {
  list: async (): Promise<{ items: PreviewDeployment[] }> => {
    await delay();
    return {
      items: [
        { id: "prev-1", app_id: "da-1", pr_number: 42, branch: "feat/dark-mode", url: "https://pr-42--my-next-app.freezenith.com", status: "running", git_sha: "a1b2c3d4", created_at: "2026-02-21T14:00:00Z", updated_at: "2026-02-21T14:05:00Z" },
        { id: "prev-2", app_id: "da-1", pr_number: 38, branch: "fix/login-bug", url: "https://pr-38--my-next-app.freezenith.com", status: "stopped", git_sha: "e5f6g7h8", created_at: "2026-02-19T10:00:00Z", updated_at: "2026-02-20T08:00:00Z" },
      ],
    };
  },
  create: async (): Promise<PreviewDeployment> => { throw new Error("Not available in demo mode"); },
  delete: async (): Promise<void> => { throw new Error("Not available in demo mode"); },
};

// ---- Autoscaler Demo ----

export const demoAutoscaler = {
  getStatus: async (): Promise<import("./api").AutoscalerStatus> => {
    await delay();
    return {
      enabled: true,
      node_count: 3,
      min_nodes: 2,
      max_nodes: 10,
      cpu_percent: 62,
      ram_percent: 54,
      budget_cap_eur: 450,
      budget_used_eur: 46.77,
      last_scale_up: "2026-02-22T10:30:00Z",
      last_scale_down: "2026-02-21T03:15:00Z",
      last_check_at: "2026-02-22T19:59:00Z",
    };
  },
  listNodes: async (): Promise<{ items: import("./api").HetznerNode[]; total: number }> => {
    await delay();
    return {
      items: [
        { server_id: 42001, name: "zenith-worker-1", ip: "116.203.42.1", status: "running", server_type: "cpx31", cpu_cores: 4, ram_mb: 8192, monthly_cost: 15.59, created_at: "2026-02-10T08:00:00Z" },
        { server_id: 42002, name: "zenith-worker-2", ip: "116.203.42.2", status: "running", server_type: "cpx31", cpu_cores: 4, ram_mb: 8192, monthly_cost: 15.59, created_at: "2026-02-15T14:20:00Z" },
        { server_id: 42003, name: "zenith-worker-3", ip: "116.203.42.3", status: "running", server_type: "cpx31", cpu_cores: 4, ram_mb: 8192, monthly_cost: 15.59, created_at: "2026-02-22T10:30:00Z" },
      ],
      total: 3,
    };
  },
  listEvents: async (): Promise<{ items: import("./api").AutoscaleEvent[]; total: number }> => {
    await delay();
    return {
      items: [
        { id: "evt-1", timestamp: "2026-02-22T10:30:00Z", action: "scale_up", old_count: 2, new_count: 3, reason: "CPU=85% RAM=72% (thresholds: CPU>80% or RAM>80%)", server_name: "zenith-worker-3" },
        { id: "evt-2", timestamp: "2026-02-21T03:15:00Z", action: "scale_down", old_count: 4, new_count: 3, reason: "CPU=28% RAM=31% (thresholds: CPU<40% and RAM<40%)", server_name: "zenith-worker-4" },
        { id: "evt-3", timestamp: "2026-02-20T16:45:00Z", action: "scale_up", old_count: 3, new_count: 4, reason: "CPU=82% RAM=78% (thresholds: CPU>80% or RAM>80%)", server_name: "zenith-worker-4" },
      ],
      total: 3,
    };
  },
};

// ---- Billing Demo (Phase 6) ----

export const demoBilling = {
  getStatus: async (): Promise<BillingStatus> => {
    await delay();
    return {
      tier: "pro",
      billing_status: "active",
      price_cents: 2900,
      currency: "eur",
      period_end: "2026-03-22T00:00:00Z",
      cancel_at_period_end: false,
      limits: {
        max_apps: 5, max_databases: 3, max_db_size_mb: 5120,
        max_auth_users: 10000, max_storage_mb: 10240, max_buckets: 5,
        max_cpu_millis: 2000, max_ram_mb: 2048, max_team_members: 3,
        backups_enabled: true, custom_domain: true, always_on: true, sleep_after_mins: 0,
      },
      usage: {
        apps: 3, databases: 3, storage_mb: 128, auth_users: 42, buckets: 1,
      },
      stripe_enabled: true,
    };
  },
  createCheckout: async (): Promise<{ checkout_url: string; session_id: string }> => {
    throw new Error("Not available in demo mode");
  },
  createPortal: async (): Promise<{ portal_url: string }> => {
    throw new Error("Not available in demo mode");
  },
  cancel: async (): Promise<{ status: string; cancel_at_period_end: boolean }> => {
    throw new Error("Not available in demo mode");
  },
  listInvoices: async (): Promise<{ items: InvoiceRecord[]; total: number }> => {
    await delay();
    return {
      items: [
        {
          id: "inv-1",
          amount_cents: 2900,
          currency: "eur",
          status: "paid",
          invoice_url: "#",
          invoice_pdf: "#",
          period_start: "2026-01-22T00:00:00Z",
          period_end: "2026-02-22T00:00:00Z",
          created_at: "2026-02-22T00:00:00Z",
        },
        {
          id: "inv-2",
          amount_cents: 2900,
          currency: "eur",
          status: "paid",
          invoice_url: "#",
          invoice_pdf: "#",
          period_start: "2025-12-22T00:00:00Z",
          period_end: "2026-01-22T00:00:00Z",
          created_at: "2026-01-22T00:00:00Z",
        },
      ],
      total: 2,
    };
  },
};

// Re-export as a unified object matching the real API import pattern
export const demoApi = {
  auth: demoAuth,
  projects: demoProjects,
  apps: demoApps,
  databases: demoDatabases,
  storage: demoStorage,
  appsDeploy: demoAppsDeploy,
  userDatabases: demoUserDatabases,
  userPlan: demoUserPlan,
  apiKeys: demoAPIKeys,
  sessions: demoSessions,
  mfa: demoMFA,
  webhooks: demoWebhooks,
  roles: demoRoles,
  ipWhitelist: demoIPWhitelist,
  compliance: demoCompliance,
  dpa: demoDPA,
  branding: demoBranding,
  sso: demoSSO,
  previews: demoPreviews,
  autoscaler: demoAutoscaler,
  billing: demoBilling,
};
