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

export const auth = {
  async login(data: LoginRequest): Promise<LoginResponse> {
    const response = await apiFetch<LoginResponse>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify(data),
    });
    setTokens(response.access_token, response.refresh_token);
    return response;
  },

  async register(data: RegisterRequest): Promise<LoginResponse> {
    const response = await apiFetch<LoginResponse>("/api/v1/auth/register", {
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
  display_name: string;
  owner: string;
  plan: string;
  region: string;
  status: string;
  created_at: string;
}

export interface CreateProjectRequest {
  name: string;
  display_name: string;
  plan: string;
  region: string;
}

export const projects = {
  list: () => apiFetch<{ items: Project[] }>("/api/v1/projects"),
  get: (id: string) => apiFetch<Project>(`/api/v1/projects/${id}`),
  create: (data: CreateProjectRequest) =>
    apiFetch<Project>("/api/v1/projects", {
      method: "POST",
      body: JSON.stringify(data),
    }),
  update: (id: string, data: Partial<CreateProjectRequest>) =>
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
