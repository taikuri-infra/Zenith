/**
 * Zenith API Client
 *
 * Shared API client library for communicating with the Zenith backend.
 * Handles authentication, token refresh, and error handling.
 */

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

// Token storage keys
const ACCESS_TOKEN_KEY = "zenith_access_token";
const REFRESH_TOKEN_KEY = "zenith_refresh_token";

// Error types
export class ApiError extends Error {
  constructor(
    public status: number,
    public statusText: string,
    public body?: unknown
  ) {
    super(`API Error ${status}: ${statusText}`);
    this.name = "ApiError";
  }
}

export class UnauthorizedError extends ApiError {
  constructor() {
    super(401, "Unauthorized");
    this.name = "UnauthorizedError";
  }
}

// Token management
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

// Core fetch wrapper
async function apiFetch<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const token = getAccessToken();

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  };

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers,
  });

  if (response.status === 401) {
    // Try to refresh token
    const refreshed = await tryRefreshToken();
    if (refreshed) {
      // Retry with new token
      headers["Authorization"] = `Bearer ${getAccessToken()}`;
      const retryResponse = await fetch(`${API_BASE_URL}${path}`, {
        ...options,
        headers,
      });
      if (!retryResponse.ok) {
        throw new ApiError(
          retryResponse.status,
          retryResponse.statusText,
          await retryResponse.json().catch(() => null)
        );
      }
      return retryResponse.json();
    }
    clearTokens();
    throw new UnauthorizedError();
  }

  if (!response.ok) {
    const body = await response.json().catch(() => null);
    throw new ApiError(response.status, response.statusText, body);
  }

  // Handle 204 No Content
  if (response.status === 204) {
    return undefined as T;
  }

  return response.json();
}

async function tryRefreshToken(): Promise<boolean> {
  const refreshToken = getRefreshToken();
  if (!refreshToken) return false;

  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });

    if (!response.ok) return false;

    const data = await response.json();
    setTokens(data.access_token, data.refresh_token);
    return true;
  } catch {
    return false;
  }
}

// ---- Auth API ----

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
}

export interface RegisterRequest {
  email: string;
  password: string;
  name: string;
}

export interface RegisterResponse {
  // Tokens are present for OAuth registration (auto-verified)
  access_token?: string;
  refresh_token?: string;
  token_type?: string;
  expires_in?: number;
  // Message is present for email/password registration (verify email)
  message?: string;
}

export const auth = {
  async login(data: LoginRequest): Promise<LoginResponse> {
    const response = await apiFetch<LoginResponse>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify(data),
    });
    setTokens(response.access_token, response.refresh_token);
    return response;
  },

  async register(data: RegisterRequest): Promise<RegisterResponse> {
    return apiFetch<RegisterResponse>("/api/v1/auth/register", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  async verifyEmail(data: { token: string }): Promise<LoginResponse> {
    const response = await apiFetch<LoginResponse>("/api/v1/auth/verify-email", {
      method: "POST",
      body: JSON.stringify(data),
    });
    setTokens(response.access_token, response.refresh_token);
    return response;
  },

  async resendVerification(data: { email: string }): Promise<{ message: string }> {
    return apiFetch<{ message: string }>("/api/v1/auth/resend-verification", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  async exchangeOAuthCode(data: { code: string }): Promise<LoginResponse> {
    const response = await apiFetch<LoginResponse>("/api/v1/auth/exchange", {
      method: "POST",
      body: JSON.stringify(data),
    });
    setTokens(response.access_token, response.refresh_token);
    return response;
  },

  logout(): void {
    clearTokens();
    if (typeof window !== "undefined") {
      window.location.href = "/login";
    }
  },

  getOAuthUrl(provider: "google" | "github"): string {
    return `${API_BASE_URL}/api/v1/auth/oauth/${provider}`;
  },
};

// ---- Projects API ----

export interface Project {
  id: string;
  name: string;
  slug: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface CreateProjectRequest {
  name: string;
  description?: string;
}

export const projects = {
  list: () => apiFetch<{ items: Project[]; total: number }>("/api/v1/projects"),
  get: (id: string) => apiFetch<Project>(`/api/v1/projects/${id}`),
  create: (data: CreateProjectRequest) =>
    apiFetch<Project>("/api/v1/projects", {
      method: "POST",
      body: JSON.stringify(data),
    }),
  update: (id: string, data: { name?: string; description?: string }) =>
    apiFetch<Project>(`/api/v1/projects/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),
  delete: (id: string) =>
    apiFetch<void>(`/api/v1/projects/${id}`, { method: "DELETE" }),
};

// ---- Apps API ----

export interface App {
  name: string;
  image: string;
  replicas: number;
  port: number;
  status: string;
  cpu: string;
  memory: string;
  domain?: string;
  env?: Record<string, string>;
  created_at: string;
}

export interface CreateAppRequest {
  name: string;
  image: string;
  replicas?: number;
  port: number;
  env?: Record<string, string>;
}

export const apps = {
  list: (projectId: string) =>
    apiFetch<{ items: App[] }>(`/api/v1/projects/${projectId}/apps`),
  get: (projectId: string, name: string) =>
    apiFetch<App>(`/api/v1/projects/${projectId}/apps/${name}`),
  create: (projectId: string, data: CreateAppRequest) =>
    apiFetch<App>(`/api/v1/projects/${projectId}/apps`, {
      method: "POST",
      body: JSON.stringify(data),
    }),
  update: (projectId: string, name: string, data: Partial<CreateAppRequest>) =>
    apiFetch<App>(`/api/v1/projects/${projectId}/apps/${name}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),
  delete: (projectId: string, name: string) =>
    apiFetch<void>(`/api/v1/projects/${projectId}/apps/${name}`, {
      method: "DELETE",
    }),
  redeploy: (projectId: string, name: string) =>
    apiFetch<void>(`/api/v1/projects/${projectId}/apps/${name}/redeploy`, {
      method: "POST",
    }),
};

// ---- Databases API ----

export interface Database {
  name: string;
  engine: string;
  version: string;
  storage: string;
  status: string;
  connection_string: string;
  port: number;
  created_at: string;
}

export interface CreateDatabaseRequest {
  name: string;
  engine: string;
  version: string;
  storage: string;
}

export const databases = {
  list: (projectId: string) =>
    apiFetch<{ items: Database[] }>(`/api/v1/projects/${projectId}/databases`),
  get: (projectId: string, name: string) =>
    apiFetch<Database>(`/api/v1/projects/${projectId}/databases/${name}`),
  create: (projectId: string, data: CreateDatabaseRequest) =>
    apiFetch<Database>(`/api/v1/projects/${projectId}/databases`, {
      method: "POST",
      body: JSON.stringify(data),
    }),
  delete: (projectId: string, name: string) =>
    apiFetch<void>(`/api/v1/projects/${projectId}/databases/${name}`, {
      method: "DELETE",
    }),
};

// ---- Storage API ----

export interface StorageBucket {
  name: string;
  access: string;
  region: string;
  size: string;
  objects: number;
  status: string;
  created_at: string;
}

export interface CreateStorageRequest {
  name: string;
  access: string;
  region: string;
}

export const storage = {
  list: (projectId: string) =>
    apiFetch<{ items: StorageBucket[] }>(
      `/api/v1/projects/${projectId}/storage`
    ),
  create: (projectId: string, data: CreateStorageRequest) =>
    apiFetch<StorageBucket>(`/api/v1/projects/${projectId}/storage`, {
      method: "POST",
      body: JSON.stringify(data),
    }),
  delete: (projectId: string, name: string) =>
    apiFetch<void>(`/api/v1/projects/${projectId}/storage/${name}`, {
      method: "DELETE",
    }),
};

// ---- Deploy Engine API (Phase 2) ----

export type AppType = "web" | "worker" | "cron";

export interface HealthCheckConfig {
  path: string;
  interval_seconds: number;
  timeout_seconds: number;
}

export interface HealthCheckStatus {
  status: "healthy" | "unhealthy" | "unknown";
  uptime_percent: number;
  last_check: string;
  response_time_ms: number;
}

export interface DeployApp {
  id: string;
  project_id: string;
  user_id: string;
  name: string;
  repo_url: string;
  branch: string;
  framework: string;
  status: string;
  subdomain: string;
  port: number;
  url: string;
  app_type?: AppType;
  command?: string;
  cron_schedule?: string;
  health_check?: HealthCheckConfig;
  health_status?: HealthCheckStatus;
  created_at: string;
  updated_at: string;
}

export interface CreateDeployAppRequest {
  project_id?: string;
  name: string;
  deploy_source: "git" | "image";
  port?: number;
  app_type?: AppType;
  command?: string;
  cron_schedule?: string;
  // Git deploy fields
  repo_url?: string;
  branch?: string;
  // Image deploy fields
  image_url?: string;
  registry_username?: string;
  registry_password?: string;
}

export interface Deployment {
  id: string;
  app_id: string;
  git_sha: string;
  status: string;
  build_log: string;
  deploy_log: string;
  created_at: string;
  updated_at: string;
}

export interface EnvVar {
  key: string;
  value: string;
}

export interface Secret {
  id: string;
  app_id: string;
  key: string;
  created_at: string;
}

export interface Release {
  id: string;
  app_id: string;
  image: string;
  git_sha: string;
  branch: string;
  message: string;
  created_at: string;
}

// ---- Per-App Database types (Phase 3) ----

export interface AppDatabase {
  id: string;
  app_id?: string;
  name: string;
  engine: string;
  host: string;
  port: number;
  db_name: string;
  db_user: string;
  db_password?: string;
  connection_string?: string;
  size_mb: number;
  max_size_mb: number;
  status: string;
  created_at: string;
}

export interface CreateAppDatabaseRequest {
  engine?: string;
  name?: string;
}

// ---- User-level databases (all databases across apps) ----

export const userDatabases = {
  list: (projectId?: string) =>
    apiFetch<AppDatabase[]>(
      projectId ? `/api/v1/databases?project_id=${projectId}` : "/api/v1/databases"
    ),
};

// ---- Standalone databases (not tied to an app) ----

export interface CreateStandaloneDatabaseRequest {
  name: string;
  engine: string;
}

export const standaloneDatabases = {
  list: (projectId?: string) =>
    apiFetch<AppDatabase[]>(
      projectId ? `/api/v1/databases?project_id=${projectId}` : "/api/v1/databases"
    ),
  get: (id: string) => apiFetch<AppDatabase>(`/api/v1/databases/${id}`),
  create: (data: CreateStandaloneDatabaseRequest) =>
    apiFetch<AppDatabase>("/api/v1/databases", {
      method: "POST",
      body: JSON.stringify(data),
    }),
  delete: (id: string) =>
    apiFetch<void>(`/api/v1/databases/${id}`, { method: "DELETE" }),
  resetPassword: (id: string) =>
    apiFetch<{ db_password: string; connection_string: string }>(
      `/api/v1/databases/${id}/reset-password`,
      { method: "POST" }
    ),
  startExplorer: (id: string, readonly = false) =>
    apiFetch<{ url: string; status: string; readonly: boolean }>(
      `/api/v1/databases/${id}/explorer`,
      { method: "POST", body: JSON.stringify({ readonly }) }
    ),
  explorerStatus: (id: string) =>
    apiFetch<{ active: boolean; url?: string; status?: string; readonly?: boolean }>(
      `/api/v1/databases/${id}/explorer`
    ),
  stopExplorer: (id: string) =>
    apiFetch<{ message: string }>(`/api/v1/databases/${id}/explorer`, {
      method: "DELETE",
    }),
};

// ---- Standalone Storage Buckets API ----

export interface StorageBucketV2 {
  id: string;
  app_id: string;
  name: string;
  access: string;
  region: string;
  size_mb: number;
  max_size_mb: number;
  objects: number;
  status: string;
  endpoint: string;
  created_at: string;
}

export interface StorageObject {
  key: string;
  size: number;
  last_modified: string;
  etag: string;
  is_folder: boolean;
}

export interface ListObjectsResponse {
  objects: StorageObject[];
  common_prefixes: string[];
  prefix: string;
  is_truncated: boolean;
}

export interface PresignedURLResponse {
  url: string;
  method: string;
  expires_in: number;
}

export const storageBuckets = {
  list: (projectId?: string) =>
    apiFetch<StorageBucketV2[]>(
      projectId ? `/api/v1/storage-buckets?project_id=${projectId}` : "/api/v1/storage-buckets"
    ),
  get: (id: string) =>
    apiFetch<StorageBucketV2>(`/api/v1/storage-buckets/${id}`),
  create: (data: { name: string; access?: string }) =>
    apiFetch<StorageBucketV2>("/api/v1/storage-buckets", {
      method: "POST",
      body: JSON.stringify(data),
    }),
  update: (id: string, data: { access: string }) =>
    apiFetch<StorageBucketV2>(`/api/v1/storage-buckets/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),
  delete: (id: string) =>
    apiFetch<{ message: string }>(`/api/v1/storage-buckets/${id}`, {
      method: "DELETE",
    }),
  listObjects: (bucketId: string, prefix = "", delimiter = "/") =>
    apiFetch<ListObjectsResponse>(
      `/api/v1/storage-buckets/${bucketId}/objects?prefix=${encodeURIComponent(prefix)}&delimiter=${encodeURIComponent(delimiter)}`
    ),
  getUploadURL: (bucketId: string, key: string, contentType?: string) =>
    apiFetch<PresignedURLResponse>(
      `/api/v1/storage-buckets/${bucketId}/objects/upload`,
      {
        method: "POST",
        body: JSON.stringify({ key, content_type: contentType }),
      }
    ),
  getDownloadURL: (bucketId: string, key: string) =>
    apiFetch<PresignedURLResponse>(
      `/api/v1/storage-buckets/${bucketId}/objects/download?key=${encodeURIComponent(key)}`
    ),
  deleteObject: (bucketId: string, key: string) =>
    apiFetch<{ message: string }>(
      `/api/v1/storage-buckets/${bucketId}/objects?key=${encodeURIComponent(key)}`,
      { method: "DELETE" }
    ),
  createFolder: (bucketId: string, prefix: string) =>
    apiFetch<{ message: string; prefix: string }>(
      `/api/v1/storage-buckets/${bucketId}/objects/folder`,
      {
        method: "POST",
        body: JSON.stringify({ prefix }),
      }
    ),
  uploadObject: (
    bucketId: string,
    key: string,
    file: File,
    onProgress?: (loaded: number, total: number) => void
  ): Promise<{ message: string }> => {
    return new Promise((resolve, reject) => {
      const token = getAccessToken();
      const xhr = new XMLHttpRequest();
      xhr.open(
        "PUT",
        `${API_BASE_URL}/api/v1/storage-buckets/${bucketId}/objects/content?key=${encodeURIComponent(key)}`
      );
      if (token) xhr.setRequestHeader("Authorization", `Bearer ${token}`);
      xhr.setRequestHeader(
        "Content-Type",
        file.type || "application/octet-stream"
      );
      xhr.upload.onprogress = (e) => {
        if (e.lengthComputable && onProgress) {
          onProgress(e.loaded, e.total);
        }
      };
      xhr.onload = () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          resolve(JSON.parse(xhr.responseText));
        } else {
          reject(new ApiError(xhr.status, xhr.statusText));
        }
      };
      xhr.onerror = () => reject(new Error("Upload failed"));
      xhr.send(file);
    });
  },
  downloadObject: async (bucketId: string, key: string): Promise<void> => {
    const token = getAccessToken();
    const response = await fetch(
      `${API_BASE_URL}/api/v1/storage-buckets/${bucketId}/objects/content?key=${encodeURIComponent(key)}`,
      {
        headers: {
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
      }
    );
    if (!response.ok) {
      throw new ApiError(response.status, response.statusText);
    }
    const blob = await response.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = key.split("/").pop() || "download";
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  },
};

// ---- Notifications ----

export interface Notification {
  id: string;
  type: "deploy_started" | "deploy_success" | "deploy_failed" | "app_crashed" | "plan_warning";
  title: string;
  description: string;
  read: boolean;
  created_at: string;
}

export const notifications = {
  list: () => apiFetch<Notification[]>("/api/v1/notifications"),
  markAllRead: () =>
    apiFetch<void>("/api/v1/notifications/read", { method: "POST" }),
};

// ---- Activity Log ----

export interface ActivityEvent {
  id: string;
  type: "deploy" | "db_create" | "app_create" | "plan_change" | "domain_add";
  title: string;
  description: string;
  created_at: string;
}

export const activity = {
  list: () => apiFetch<ActivityEvent[]>("/api/v1/activity"),
};

// ---- Database Backup types (Phase 3) ----

export interface DatabaseBackup {
  id: string;
  database_id: string;
  type: string;
  status: string;
  size_mb: number;
  error?: string;
  created_at: string;
}

// ---- Per-App Storage types (Phase 3) ----

export interface AppBucket {
  id: string;
  app_id: string;
  name: string;
  access: string;
  region: string;
  size_mb: number;
  max_size_mb: number;
  objects: number;
  status: string;
  endpoint: string;
  created_at: string;
}

export interface CreateAppBucketRequest {
  name: string;
  access?: string;
}

// ---- App Auth types (Phase 3) ----

export interface AppAuthConfig {
  enabled: boolean;
  user_count: number;
  max_users: number;
}

export interface AppAuthUser {
  id: string;
  email: string;
  name: string;
  verified: boolean;
  created_at: string;
}

// ---- Plan types (Phase 4) ----

export interface PlanLimits {
  max_apps: number;
  max_databases: number;
  max_db_size_mb: number;
  max_auth_users: number;
  max_storage_mb: number;
  max_buckets: number;
  max_cpu_millis: number;
  max_ram_mb: number;
  max_team_members: number;
  backups_enabled: boolean;
  custom_domain: boolean;
  always_on: boolean;
  sleep_after_mins: number;
}

export interface PlanUsage {
  apps: number;
  databases: number;
  storage_mb: number;
  auth_users: number;
  buckets: number;
}

export interface UserPlanResponse {
  tier: string;
  limits: PlanLimits;
  usage: PlanUsage;
}

export const userPlan = {
  get: () => apiFetch<UserPlanResponse>("/api/v1/plan"),
  upgrade: (tier: string) =>
    apiFetch<UserPlanResponse>("/api/v1/plan/upgrade", {
      method: "POST",
      body: JSON.stringify({ tier }),
    }),
};

// ---- Custom Domain types (Phase 4) ----

export interface CustomDomain {
  id: string;
  app_id: string;
  domain: string;
  status: string;
  tls_ready: boolean;
  created_at: string;
}

// ---- MFA types (Phase 6.5) ----

export interface MFAStatus {
  status: string;
  enabled_at?: string;
  backup_codes: number;
}

export interface MFAEnableResponse {
  secret: string;
  otpauth_uri: string;
  backup_codes: string[];
}

export const mfa = {
  getStatus: () => apiFetch<MFAStatus>("/api/v1/auth/mfa"),
  enable: () =>
    apiFetch<MFAEnableResponse>("/api/v1/auth/mfa/enable", { method: "POST" }),
  verify: (code: string) =>
    apiFetch<{ status: string; enabled_at: string }>("/api/v1/auth/mfa/verify", {
      method: "POST",
      body: JSON.stringify({ code }),
    }),
  disable: (code: string) =>
    apiFetch<{ status: string }>("/api/v1/auth/mfa/disable", {
      method: "POST",
      body: JSON.stringify({ code }),
    }),
  regenerateBackupCodes: () =>
    apiFetch<{ backup_codes: string[] }>("/api/v1/auth/mfa/backup-codes", {
      method: "POST",
    }),
};

// ---- Webhook types (Phase 6.5) ----

export interface UserWebhook {
  id: string;
  user_id: string;
  url: string;
  events: string[];
  secret?: string;
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface WebhookDelivery {
  id: string;
  webhook_id: string;
  event: string;
  payload: string;
  status: string;
  status_code?: number;
  error?: string;
  attempts: number;
  created_at: string;
}

export const webhooks = {
  list: () => apiFetch<{ items: UserWebhook[] }>("/api/v1/webhooks"),
  create: (url: string, events: string[]) =>
    apiFetch<UserWebhook>("/api/v1/webhooks", {
      method: "POST",
      body: JSON.stringify({ url, events }),
    }),
  update: (id: string, data: { url?: string; events?: string[]; active?: boolean }) =>
    apiFetch<UserWebhook>(`/api/v1/webhooks/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),
  delete: (id: string) =>
    apiFetch<void>(`/api/v1/webhooks/${id}`, { method: "DELETE" }),
  listDeliveries: (id: string) =>
    apiFetch<{ items: WebhookDelivery[] }>(`/api/v1/webhooks/${id}/deliveries`),
};

// ---- SSO types (Phase 6.5) ----

export interface SSOConfig {
  id: string;
  user_id: string;
  provider: string;
  entity_id?: string;
  sso_url?: string;
  certificate?: string;
  client_id?: string;
  discovery_url?: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export const sso = {
  list: () => apiFetch<{ items: SSOConfig[] }>("/api/v1/settings/sso"),
  configureSAML: (entityId: string, ssoUrl: string, certificate: string) =>
    apiFetch<SSOConfig>("/api/v1/settings/sso/saml", {
      method: "POST",
      body: JSON.stringify({ entity_id: entityId, sso_url: ssoUrl, certificate }),
    }),
  configureOIDC: (clientId: string, clientSecret: string, discoveryUrl: string) =>
    apiFetch<SSOConfig>("/api/v1/settings/sso/oidc", {
      method: "POST",
      body: JSON.stringify({ client_id: clientId, client_secret: clientSecret, discovery_url: discoveryUrl }),
    }),
  delete: (id: string) =>
    apiFetch<void>(`/api/v1/settings/sso/${id}`, { method: "DELETE" }),
};

// ---- Preview Deployment types (Phase 6.5) ----

export interface PreviewDeployment {
  id: string;
  app_id: string;
  pr_number: number;
  branch: string;
  url: string;
  status: string;
  git_sha: string;
  created_at: string;
  updated_at: string;
}

export const previews = {
  list: (appId: string) =>
    apiFetch<{ items: PreviewDeployment[] }>(`/api/v1/apps/${appId}/previews`),
  create: (appId: string, prNumber: number, branch: string, gitSha: string) =>
    apiFetch<PreviewDeployment>(`/api/v1/apps/${appId}/previews`, {
      method: "POST",
      body: JSON.stringify({ pr_number: prNumber, branch, git_sha: gitSha }),
    }),
  delete: (appId: string, previewId: string) =>
    apiFetch<void>(`/api/v1/apps/${appId}/previews/${previewId}`, { method: "DELETE" }),
};

// ---- DPA types (Phase 6.5) ----

export interface DPARecord {
  user_id: string;
  status: string;
  signed_by?: string;
  signed_at?: string;
  ip_address?: string;
}

export const dpa = {
  get: () => apiFetch<DPARecord>("/api/v1/settings/dpa"),
  sign: (signedBy: string) =>
    apiFetch<DPARecord>("/api/v1/settings/dpa/sign", {
      method: "POST",
      body: JSON.stringify({ signed_by: signedBy }),
    }),
};

// ---- Branding types (Phase 6.5) ----

export interface BrandingConfig {
  user_id: string;
  company_name: string;
  logo_url: string;
  primary_color: string;
  dashboard_domain?: string;
  domain_verified: boolean;
  hide_branding: boolean;
  updated_at: string;
}

export const branding = {
  get: () => apiFetch<BrandingConfig>("/api/v1/settings/branding"),
  update: (data: { company_name?: string; logo_url?: string; primary_color?: string; hide_branding?: boolean }) =>
    apiFetch<BrandingConfig>("/api/v1/settings/branding", {
      method: "PUT",
      body: JSON.stringify(data),
    }),
  setDomain: (domain: string) =>
    apiFetch<BrandingConfig>("/api/v1/settings/domain", {
      method: "POST",
      body: JSON.stringify({ domain }),
    }),
};

// ---- Custom Roles types (Phase 6.5) ----

export interface CustomRole {
  id: string;
  user_id: string;
  name: string;
  description: string;
  permissions: string[];
  created_at: string;
  updated_at: string;
}

export interface RoleAssignment {
  id: string;
  role_id: string;
  member_id: string;
  assigned_by: string;
  created_at: string;
}

export const roles = {
  list: () => apiFetch<{ items: CustomRole[] }>("/api/v1/roles"),
  create: (name: string, description: string, permissions: string[]) =>
    apiFetch<CustomRole>("/api/v1/roles", {
      method: "POST",
      body: JSON.stringify({ name, description, permissions }),
    }),
  update: (id: string, data: { name?: string; description?: string; permissions?: string[] }) =>
    apiFetch<CustomRole>(`/api/v1/roles/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),
  delete: (id: string) =>
    apiFetch<void>(`/api/v1/roles/${id}`, { method: "DELETE" }),
  listPermissions: () =>
    apiFetch<{ permissions: string[] }>("/api/v1/roles/permissions"),
};

// ---- IP Whitelist types (Phase 6.5) ----

export interface IPWhitelistEntry {
  id: string;
  user_id: string;
  cidr: string;
  description: string;
  created_at: string;
}

export const ipWhitelist = {
  list: () => apiFetch<{ items: IPWhitelistEntry[] }>("/api/v1/settings/ip-whitelist"),
  add: (cidr: string, description: string) =>
    apiFetch<IPWhitelistEntry>("/api/v1/settings/ip-whitelist", {
      method: "POST",
      body: JSON.stringify({ cidr, description }),
    }),
  delete: (id: string) =>
    apiFetch<void>(`/api/v1/settings/ip-whitelist/${id}`, { method: "DELETE" }),
};

// ---- Compliance types (Phase 6.5) ----

export interface ComplianceCheck {
  category: string;
  item: string;
  status: string;
  description: string;
}

export interface ComplianceResponse {
  checks: ComplianceCheck[];
  summary: {
    total: number;
    pass: number;
    fail: number;
    partial: number;
    na: number;
  };
}

export const compliance = {
  getStatus: () => apiFetch<ComplianceResponse>("/api/v1/compliance"),
};

// ---- API Key types (Phase 6.5) ----

export interface APIKey {
  id: string;
  name: string;
  key_prefix: string;
  key?: string; // Only returned on creation
  scopes: string[];
  user_id: string;
  last_used_at?: string;
  created_at: string;
}

export const apiKeys = {
  list: () => apiFetch<{ items: APIKey[]; total: number }>("/api/v1/api-keys"),
  create: (name: string, scopes: string[]) =>
    apiFetch<APIKey>("/api/v1/api-keys", {
      method: "POST",
      body: JSON.stringify({ name, scopes }),
    }),
  delete: (id: string) =>
    apiFetch<{ message: string }>(`/api/v1/api-keys/${id}`, { method: "DELETE" }),
};

// ---- Session types (Phase 6.5) ----

export interface Session {
  id: string;
  user_id: string;
  ip_address: string;
  user_agent: string;
  device: string;
  current: boolean;
  created_at: string;
  expires_at: string;
  last_seen_at: string;
}

export const sessions = {
  list: () => apiFetch<{ items: Session[]; total: number }>("/api/v1/auth/sessions"),
  revoke: (id: string) =>
    apiFetch<{ message: string }>(`/api/v1/auth/sessions/${id}`, { method: "DELETE" }),
  revokeAll: () =>
    apiFetch<{ message: string }>("/api/v1/auth/sessions", { method: "DELETE" }),
};

export const appsDeploy = {
  list: (projectId?: string) =>
    apiFetch<{ items: DeployApp[]; total: number }>(
      projectId ? `/api/v1/apps?project_id=${projectId}` : "/api/v1/apps"
    ),
  get: (id: string) =>
    apiFetch<DeployApp>(`/api/v1/apps/${id}`),
  create: (data: CreateDeployAppRequest) =>
    apiFetch<DeployApp>("/api/v1/apps", {
      method: "POST",
      body: JSON.stringify(data),
    }),
  delete: (id: string) =>
    apiFetch<void>(`/api/v1/apps/${id}`, { method: "DELETE" }),

  // Deployments
  listDeployments: (appId: string, limit = 20) =>
    apiFetch<{ items: Deployment[]; total: number }>(
      `/api/v1/apps/${appId}/deployments?limit=${limit}`
    ),
  getDeployment: (appId: string, deployId: string) =>
    apiFetch<Deployment>(`/api/v1/apps/${appId}/deployments/${deployId}`),
  rollback: (appId: string, deploymentId: string) =>
    apiFetch<{ message: string }>(`/api/v1/apps/${appId}/rollback`, {
      method: "POST",
      body: JSON.stringify({ deployment_id: deploymentId }),
    }),

  // Env vars
  getEnvVars: (appId: string) =>
    apiFetch<{ items: EnvVar[]; total: number }>(`/api/v1/apps/${appId}/env`),
  setEnvVars: (appId: string, vars: Record<string, string>) =>
    apiFetch<{ message: string }>(`/api/v1/apps/${appId}/env`, {
      method: "PUT",
      body: JSON.stringify({ vars }),
    }),
  deleteEnvVar: (appId: string, key: string) =>
    apiFetch<void>(`/api/v1/apps/${appId}/env/${key}`, { method: "DELETE" }),

  // Secrets
  listSecrets: (appId: string) =>
    apiFetch<{ secrets: Secret[] }>(`/api/v1/apps/${appId}/secrets`),
  getSecretValue: (appId: string, key: string) =>
    apiFetch<{ key: string; value: string }>(
      `/api/v1/apps/${appId}/secrets/${key}/value`
    ),
  setSecret: (appId: string, key: string, value: string) =>
    apiFetch<{ key: string; status: string }>(`/api/v1/apps/${appId}/secrets`, {
      method: "POST",
      body: JSON.stringify({ key, value }),
    }),
  deleteSecret: (appId: string, key: string) =>
    apiFetch<{ key: string; status: string }>(
      `/api/v1/apps/${appId}/secrets/${key}`,
      { method: "DELETE" }
    ),

  // Releases
  listReleases: (appId: string) =>
    apiFetch<{ releases: Release[] }>(`/api/v1/apps/${appId}/releases`),
  deployRelease: (appId: string, releaseId: string) =>
    apiFetch<{ deployment_id: string; release_id: string; image: string; status: string }>(
      `/api/v1/apps/${appId}/releases/${releaseId}/deploy`,
      { method: "POST" }
    ),

  // Databases (per-app, Phase 3)
  listAppDatabases: (appId: string) =>
    apiFetch<AppDatabase[]>(`/api/v1/apps/${appId}/databases`),
  getAppDatabase: (appId: string, dbId: string) =>
    apiFetch<AppDatabase>(`/api/v1/apps/${appId}/databases/${dbId}`),
  createAppDatabase: (appId: string, data: CreateAppDatabaseRequest) =>
    apiFetch<AppDatabase>(`/api/v1/apps/${appId}/databases`, {
      method: "POST",
      body: JSON.stringify(data),
    }),
  deleteAppDatabase: (appId: string, dbId: string) =>
    apiFetch<{ message: string }>(`/api/v1/apps/${appId}/databases/${dbId}`, {
      method: "DELETE",
    }),

  // Custom Domains (Phase 4)
  listDomains: (appId: string) =>
    apiFetch<CustomDomain[]>(`/api/v1/apps/${appId}/domains`),
  addDomain: (appId: string, domain: string) =>
    apiFetch<CustomDomain>(`/api/v1/apps/${appId}/domains`, {
      method: "POST",
      body: JSON.stringify({ domain }),
    }),
  deleteDomain: (appId: string, domainId: string) =>
    apiFetch<{ message: string }>(`/api/v1/apps/${appId}/domains/${domainId}`, {
      method: "DELETE",
    }),

  // Database Backups (per-database, Phase 3)
  listBackups: (appId: string, dbId: string) =>
    apiFetch<DatabaseBackup[]>(`/api/v1/apps/${appId}/databases/${dbId}/backups`),
  createBackup: (appId: string, dbId: string) =>
    apiFetch<DatabaseBackup>(`/api/v1/apps/${appId}/databases/${dbId}/backups`, {
      method: "POST",
      body: JSON.stringify({ type: "manual" }),
    }),
  deleteBackup: (appId: string, dbId: string, backupId: string) =>
    apiFetch<{ message: string }>(`/api/v1/apps/${appId}/databases/${dbId}/backups/${backupId}`, {
      method: "DELETE",
    }),
  restoreBackup: (appId: string, dbId: string, backupId: string) =>
    apiFetch<{ message: string; backup_id: string; database_id: string }>(
      `/api/v1/apps/${appId}/databases/${dbId}/backups/${backupId}/restore`,
      { method: "POST" }
    ),

  // Storage (per-app S3 buckets, Phase 3)
  listAppBuckets: (appId: string) =>
    apiFetch<AppBucket[]>(`/api/v1/apps/${appId}/storage`),
  createAppBucket: (appId: string, data: CreateAppBucketRequest) =>
    apiFetch<AppBucket>(`/api/v1/apps/${appId}/storage`, {
      method: "POST",
      body: JSON.stringify(data),
    }),
  deleteAppBucket: (appId: string, bucketId: string) =>
    apiFetch<{ message: string }>(`/api/v1/apps/${appId}/storage/${bucketId}`, {
      method: "DELETE",
    }),

  // App Auth (per-app built-in auth, Phase 3)
  getAuthStatus: (appId: string) =>
    apiFetch<AppAuthConfig>(`/api/v1/apps/${appId}/auth`),
  enableAuth: (appId: string) =>
    apiFetch<AppAuthConfig>(`/api/v1/apps/${appId}/auth/enable`, {
      method: "POST",
    }),
  disableAuth: (appId: string) =>
    apiFetch<{ message: string }>(`/api/v1/apps/${appId}/auth/disable`, {
      method: "POST",
    }),
  listAuthUsers: (appId: string) =>
    apiFetch<{ users: AppAuthUser[]; total: number }>(
      `/api/v1/apps/${appId}/auth/users`
    ),
  deleteAuthUser: (appId: string, userId: string) =>
    apiFetch<{ message: string }>(
      `/api/v1/apps/${appId}/auth/users/${userId}`,
      { method: "DELETE" }
    ),
};

// ---- Billing types (Phase 6) ----

export interface BillingStatus {
  tier: string;
  billing_status: string;
  price_cents: number;
  currency: string;
  period_end?: string;
  cancel_at_period_end: boolean;
  limits: PlanLimits;
  usage: PlanUsage;
  stripe_enabled: boolean;
}

export interface CheckoutResponse {
  checkout_url: string;
  session_id: string;
}

export interface PortalResponse {
  portal_url: string;
}

export interface InvoiceRecord {
  id: string;
  amount_cents: number;
  currency: string;
  status: string;
  invoice_url?: string;
  invoice_pdf?: string;
  period_start: string;
  period_end: string;
  created_at: string;
}

export const billing = {
  getStatus: () => apiFetch<BillingStatus>("/api/v1/billing"),
  createCheckout: (tier: string) =>
    apiFetch<CheckoutResponse>("/api/v1/billing/checkout", {
      method: "POST",
      body: JSON.stringify({ tier }),
    }),
  createPortal: () =>
    apiFetch<PortalResponse>("/api/v1/billing/portal", { method: "POST" }),
  cancel: (immediate = false) =>
    apiFetch<{ status: string; cancel_at_period_end: boolean }>(
      "/api/v1/billing/cancel",
      {
        method: "POST",
        body: JSON.stringify({ immediate }),
      }
    ),
  listInvoices: () =>
    apiFetch<{ items: InvoiceRecord[]; total: number }>("/api/v1/billing/invoices"),
};

// ---- Autoscaler (Phase 5 — Admin) ----

export interface AutoscalerStatus {
  enabled: boolean;
  node_count: number;
  min_nodes: number;
  max_nodes: number;
  cpu_percent: number;
  ram_percent: number;
  budget_cap_eur: number;
  budget_used_eur: number;
  last_scale_up: string;
  last_scale_down: string;
  last_check_at: string;
}

export interface HetznerNode {
  server_id: number;
  name: string;
  ip: string;
  status: string;
  server_type: string;
  cpu_cores: number;
  ram_mb: number;
  monthly_cost: number;
  created_at: string;
}

export interface AutoscaleEvent {
  id: string;
  timestamp: string;
  action: "scale_up" | "scale_down";
  old_count: number;
  new_count: number;
  reason: string;
  server_name: string;
}

export const autoscaler = {
  getStatus: () =>
    apiFetch<AutoscalerStatus>("/api/v1/admin/autoscaler/status"),
  listNodes: () =>
    apiFetch<{ items: HetznerNode[]; total: number }>("/api/v1/admin/autoscaler/nodes"),
  listEvents: (limit = 50) =>
    apiFetch<{ items: AutoscaleEvent[]; total: number }>(`/api/v1/admin/autoscaler/events?limit=${limit}`),
};

// ---- Registry API ----

export interface RegistryImage {
  name: string;
  tags: string[];
  size: string;
  lastPushed: string;
}

export const registry = {
  listImages: () =>
    apiFetch<{ items: RegistryImage[] }>("/api/v1/registry/images"),
};

// ---- API Gateways ----

export interface ApiGateway {
  id: string;
  user_id: string;
  name: string;
  slug: string;
  status: "provisioning" | "active" | "error" | "deleting";
  endpoint: string;
  route_count: number;
  created_at: string;
  updated_at: string;
}

export interface GatewayRoutePlugin {
  name: string;
  enable: boolean;
  config: Record<string, unknown>;
}

export interface GatewayRouteInfo {
  id: string;
  gateway_id: string;
  name: string;
  path: string;
  methods: string[];
  app_id: string;
  app_subdomain: string;
  strip_prefix: boolean;
  auth: "none" | "jwt" | "key-auth";
  plugins: GatewayRoutePlugin[];
  priority: number;
  status: "active" | "stopped";
  created_at: string;
  updated_at: string;
}

export interface CreateRouteInput {
  name: string;
  path: string;
  methods: string[];
  app_id: string;
  strip_prefix?: boolean;
  auth?: string;
  plugins?: GatewayRoutePlugin[];
  priority?: number;
}

export interface UpdateRouteInput {
  name?: string;
  path?: string;
  methods?: string[];
  app_id?: string;
  strip_prefix?: boolean;
  auth?: string;
  plugins?: GatewayRoutePlugin[];
  priority?: number;
  status?: string;
}

export const gateways = {
  list: (projectId?: string) =>
    apiFetch<ApiGateway[]>(
      projectId ? `/api/v1/gateways?project_id=${projectId}` : "/api/v1/gateways"
    ),
  get: (id: string) =>
    apiFetch<{ gateway: ApiGateway; routes: GatewayRouteInfo[] }>(
      `/api/v1/gateways/${id}`
    ),
  create: (name: string) =>
    apiFetch<ApiGateway>("/api/v1/gateways", {
      method: "POST",
      body: JSON.stringify({ name }),
    }),
  update: (id: string, name: string) =>
    apiFetch<ApiGateway>(`/api/v1/gateways/${id}`, {
      method: "PUT",
      body: JSON.stringify({ name }),
    }),
  delete: (id: string) =>
    apiFetch<void>(`/api/v1/gateways/${id}`, { method: "DELETE" }),
  sync: (id: string) =>
    apiFetch<void>(`/api/v1/gateways/${id}/sync`, { method: "POST" }),
  listRoutes: (gwId: string) =>
    apiFetch<GatewayRouteInfo[]>(`/api/v1/gateways/${gwId}/routes`),
  createRoute: (gwId: string, data: CreateRouteInput) =>
    apiFetch<GatewayRouteInfo>(`/api/v1/gateways/${gwId}/routes`, {
      method: "POST",
      body: JSON.stringify(data),
    }),
  updateRoute: (gwId: string, routeId: string, data: UpdateRouteInput) =>
    apiFetch<GatewayRouteInfo>(`/api/v1/gateways/${gwId}/routes/${routeId}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),
  deleteRoute: (gwId: string, routeId: string) =>
    apiFetch<void>(`/api/v1/gateways/${gwId}/routes/${routeId}`, {
      method: "DELETE",
    }),
};

// ---- WebSocket for real-time updates ----

export type WebSocketEvent =
  | { type: "deployment_progress"; data: { app: string; status: string; progress: number } }
  | { type: "log"; data: { app: string; level: string; message: string; timestamp: string } }
  | { type: "status_change"; data: { resource: string; name: string; status: string } };

export function connectWebSocket(
  projectId: string,
  onMessage: (event: WebSocketEvent) => void,
  onError?: (error: Event) => void
): WebSocket | null {
  if (typeof window === "undefined") return null;

  const token = getAccessToken();
  const wsUrl = API_BASE_URL.replace(/^http/, "ws");
  const ws = new WebSocket(
    `${wsUrl}/api/v1/projects/${projectId}/ws?token=${token}`
  );

  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data) as WebSocketEvent;
      onMessage(data);
    } catch {
      // Ignore non-JSON messages
    }
  };

  ws.onerror = (error) => {
    if (onError) onError(error);
  };

  return ws;
}
