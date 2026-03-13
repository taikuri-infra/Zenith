"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { Modal } from "@/components/modal";
import React, { useState, useEffect, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import { getApi } from "@/lib/get-api";
import type { AuthPool, AuthPoolUser, AuthPoolRole, AuthPoolSocialProvider, AuthPoolSession, AuthPoolCredential } from "@/lib/api";
import {
  Shield, Users, Key, Copy, Check, ChevronLeft, Plus,
  Trash2, Loader2, UserCheck, UserX, Eye, EyeOff, LogIn, BookOpen, Tag, X,
  Globe, Monitor, Mail, ShieldCheck, Fingerprint,
} from "lucide-react";
import Link from "next/link";

function CopyButton({ value }: { value: string }) {
  const [copied, setCopied] = useState(false);
  const copy = () => {
    navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };
  return (
    <button onClick={copy} className="ml-2 rounded p-1 text-neutral-500 hover:text-white transition-colors">
      {copied ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5" />}
    </button>
  );
}

function CredentialRow({ label, value, secret }: { label: string; value: string; secret?: boolean }) {
  const [visible, setVisible] = useState(false);
  const display = secret && !visible ? "••••••••••••••••" : value;
  return (
    <div className="flex items-center justify-between rounded-lg bg-surface-200 px-4 py-3">
      <div className="min-w-0 flex-1">
        <p className="text-[11px] font-medium text-neutral-500 uppercase tracking-wide">{label}</p>
        <p className="mt-0.5 font-mono text-xs text-neutral-300 truncate">{display}</p>
      </div>
      <div className="flex items-center">
        {secret && (
          <button onClick={() => setVisible(!visible)} className="rounded p-1 text-neutral-500 hover:text-white transition-colors">
            {visible ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
          </button>
        )}
        <CopyButton value={value} />
      </div>
    </div>
  );
}

export default function PoolDetailPage() {
  const params = useParams();
  const router = useRouter();
  const poolId = params.poolId as string;
  const { authPools: api } = getApi();

  const [pool, setPool] = useState<AuthPool | null>(null);
  const [users, setUsers] = useState<AuthPoolUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [usersLoading, setUsersLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Add user modal
  const [showAddUser, setShowAddUser] = useState(false);
  const [newEmail, setNewEmail] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [newFirstName, setNewFirstName] = useState("");
  const [newLastName, setNewLastName] = useState("");
  const [addingUser, setAddingUser] = useState(false);
  const [addUserError, setAddUserError] = useState<string | null>(null);

  // Delete user confirm
  const [deleteUserTarget, setDeleteUserTarget] = useState<AuthPoolUser | null>(null);
  const [deletingUser, setDeletingUser] = useState(false);

  // Test login
  const [testEmail, setTestEmail] = useState("");
  const [testPassword, setTestPassword] = useState("");
  const [testingLogin, setTestingLogin] = useState(false);
  const [tokenResult, setTokenResult] = useState<{ access_token: string; refresh_token: string; expires_in: number; token_type: string } | null>(null);
  const [tokenError, setTokenError] = useState<string | null>(null);

  // Secret reveal
  const [revealedSecret, setRevealedSecret] = useState<string | null>(null);

  // Roles
  const [roles, setRoles] = useState<AuthPoolRole[]>([]);
  const [rolesLoading, setRolesLoading] = useState(true);
  const [newRoleName, setNewRoleName] = useState("");
  const [newRoleDesc, setNewRoleDesc] = useState("");
  const [creatingRole, setCreatingRole] = useState(false);
  // User roles: userId → role names
  const [userRolesMap, setUserRolesMap] = useState<Record<string, AuthPoolRole[]>>({});
  const [assigningRole, setAssigningRole] = useState<string | null>(null);

  // Social providers
  const [providers, setProviders] = useState<AuthPoolSocialProvider[]>([]);
  const [providersLoading, setProvidersLoading] = useState(true);
  const [showAddProvider, setShowAddProvider] = useState(false);
  const [newProviderType, setNewProviderType] = useState("google");
  const [newProviderClientId, setNewProviderClientId] = useState("");
  const [newProviderClientSecret, setNewProviderClientSecret] = useState("");
  const [addingProvider, setAddingProvider] = useState(false);

  // Sessions (per user expandable)
  const [expandedUserSessions, setExpandedUserSessions] = useState<string | null>(null);
  const [userSessions, setUserSessions] = useState<AuthPoolSession[]>([]);
  const [sessionsLoading, setSessionsLoading] = useState(false);
  const [userCredentials, setUserCredentials] = useState<AuthPoolCredential[]>([]);

  const fetchPool = useCallback(async () => {
    try {
      setLoading(true);
      const data = await api.get(poolId);
      setPool(data);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load pool");
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [poolId]);

  const fetchUsers = useCallback(async () => {
    try {
      setUsersLoading(true);
      const data = await api.listUsers(poolId);
      setUsers(Array.isArray(data) ? data : []);
    } catch {
      setUsers([]);
    } finally {
      setUsersLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [poolId]);

  const fetchRoles = useCallback(async () => {
    try {
      setRolesLoading(true);
      const data = await api.listRoles(poolId);
      setRoles(Array.isArray(data) ? data : []);
    } catch {
      setRoles([]);
    } finally {
      setRolesLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [poolId]);

  const fetchProviders = useCallback(async () => {
    try {
      setProvidersLoading(true);
      const data = await api.listProviders(poolId);
      setProviders(Array.isArray(data) ? data : []);
    } catch {
      setProviders([]);
    } finally {
      setProvidersLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [poolId]);

  const fetchUserRoles = useCallback(async (userId: string) => {
    try {
      const data = await api.getUserRoles(poolId, userId);
      setUserRolesMap(prev => ({ ...prev, [userId]: Array.isArray(data) ? data : [] }));
    } catch {
      setUserRolesMap(prev => ({ ...prev, [userId]: [] }));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [poolId]);

  useEffect(() => {
    fetchPool();
    fetchUsers();
    fetchRoles();
    fetchProviders();
  }, [fetchPool, fetchUsers, fetchRoles, fetchProviders]);

  // Fetch roles for each user when users load
  useEffect(() => {
    users.forEach(u => fetchUserRoles(u.id));
  }, [users, fetchUserRoles]);

  const handleCreateRole = async () => {
    if (!newRoleName.trim()) return;
    setCreatingRole(true);
    try {
      await api.createRole(poolId, newRoleName.trim(), newRoleDesc.trim());
      setNewRoleName("");
      setNewRoleDesc("");
      await fetchRoles();
    } catch {
      // ignore
    } finally {
      setCreatingRole(false);
    }
  };

  const handleDeleteRole = async (roleName: string) => {
    try {
      await api.deleteRole(poolId, roleName);
      await fetchRoles();
      // Refresh user roles
      users.forEach(u => fetchUserRoles(u.id));
    } catch {
      // ignore
    }
  };

  const handleAssignRole = async (userId: string, roleName: string) => {
    setAssigningRole(userId + roleName);
    try {
      await api.assignRole(poolId, userId, roleName);
      await fetchUserRoles(userId);
    } catch {
      // ignore
    } finally {
      setAssigningRole(null);
    }
  };

  const handleRemoveRole = async (userId: string, roleName: string) => {
    try {
      await api.removeRole(poolId, userId, roleName);
      await fetchUserRoles(userId);
    } catch {
      // ignore
    }
  };

  const handleAddProvider = async () => {
    if (!newProviderClientId.trim() || !newProviderClientSecret.trim()) return;
    setAddingProvider(true);
    try {
      await api.createProvider(poolId, newProviderType, newProviderClientId.trim(), newProviderClientSecret.trim());
      setShowAddProvider(false);
      setNewProviderClientId("");
      setNewProviderClientSecret("");
      await fetchProviders();
    } catch {
      // ignore
    } finally {
      setAddingProvider(false);
    }
  };

  const handleDeleteProvider = async (alias: string) => {
    try {
      await api.deleteProvider(poolId, alias);
      await fetchProviders();
    } catch {
      // ignore
    }
  };

  const handleExpandSessions = async (userId: string) => {
    if (expandedUserSessions === userId) {
      setExpandedUserSessions(null);
      return;
    }
    setExpandedUserSessions(userId);
    setSessionsLoading(true);
    try {
      const [sessions, creds] = await Promise.all([
        api.getUserSessions(poolId, userId),
        api.getUserCredentials(poolId, userId),
      ]);
      setUserSessions(Array.isArray(sessions) ? sessions : []);
      setUserCredentials(Array.isArray(creds) ? creds : []);
    } catch {
      setUserSessions([]);
      setUserCredentials([]);
    } finally {
      setSessionsLoading(false);
    }
  };

  const handleRevokeSession = async (userId: string, sessionId: string) => {
    try {
      await api.revokeUserSession(poolId, userId, sessionId);
      handleExpandSessions(userId);
    } catch {
      // ignore
    }
  };

  const handleRevokeAllSessions = async (userId: string) => {
    try {
      await api.revokeAllUserSessions(poolId, userId);
      handleExpandSessions(userId);
    } catch {
      // ignore
    }
  };

  const handleDeleteCredential = async (userId: string, credentialId: string) => {
    try {
      await api.deleteUserCredential(poolId, userId, credentialId);
      handleExpandSessions(userId);
    } catch {
      // ignore
    }
  };

  const handleSendVerifyEmail = async (userId: string) => {
    try {
      await api.sendVerifyEmail(poolId, userId);
    } catch {
      // ignore
    }
  };

  const handleAddUser = async () => {
    if (!newEmail.trim() || !newPassword.trim()) return;
    setAddingUser(true);
    setAddUserError(null);
    try {
      await api.createUser(poolId, newEmail.trim(), newPassword, newFirstName.trim(), newLastName.trim());
      setShowAddUser(false);
      setNewEmail("");
      setNewPassword("");
      setNewFirstName("");
      setNewLastName("");
      await fetchUsers();
      await fetchPool(); // refresh user count
    } catch (e) {
      setAddUserError(e instanceof Error ? e.message : "Failed to add user");
    } finally {
      setAddingUser(false);
    }
  };

  const handleDeleteUser = async () => {
    if (!deleteUserTarget) return;
    setDeletingUser(true);
    try {
      await api.deleteUser(poolId, deleteUserTarget.id);
      setDeleteUserTarget(null);
      await fetchUsers();
      await fetchPool();
    } catch {
      // ignore
    } finally {
      setDeletingUser(false);
    }
  };

  const handleToggleUser = async (user: AuthPoolUser) => {
    try {
      if (user.enabled) {
        await api.disableUser(poolId, user.id);
      } else {
        await api.enableUser(poolId, user.id);
      }
      await fetchUsers();
    } catch {
      // ignore
    }
  };

  const handleTestLogin = async () => {
    if (!testEmail.trim() || !testPassword.trim()) return;
    setTestingLogin(true);
    setTokenError(null);
    setTokenResult(null);
    try {
      const result = await api.login(poolId, testEmail.trim(), testPassword);
      setTokenResult(result);
    } catch (e) {
      setTokenError(e instanceof Error ? e.message : "Authentication failed");
    } finally {
      setTestingLogin(false);
    }
  };

  if (loading) {
    return (
      <Shell>
        <div className="flex items-center justify-center py-24">
          <Loader2 className="h-6 w-6 animate-spin text-accent-500" />
        </div>
      </Shell>
    );
  }

  if (error || !pool) {
    return (
      <Shell>
        <div className="py-16 text-center">
          <p className="text-sm text-red-400">{error || "Pool not found"}</p>
          <Link href="/auth" className="mt-3 inline-block text-sm text-accent-400 hover:text-accent-300">
            Back to Auth Pools
          </Link>
        </div>
      </Shell>
    );
  }

  const statusMap: Record<string, "active" | "provisioning" | "error" | "deleting"> = {
    active: "active",
    provisioning: "provisioning",
    error: "error",
    deleting: "deleting",
  };

  return (
    <Shell>
      <div className="space-y-6">
        {/* Header */}
        <div>
          <Link
            href="/auth"
            className="mb-3 inline-flex items-center gap-1 text-xs text-neutral-500 hover:text-white transition-colors"
          >
            <ChevronLeft className="h-3 w-3" />
            Auth Pools
          </Link>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-accent-500/10">
                <Shield className="h-5 w-5 text-accent-400" />
              </div>
              <div>
                <h1 className="text-lg font-semibold text-white">{pool.name}</h1>
                <p className="text-xs text-neutral-500">{pool.realm_name}</p>
              </div>
            </div>
            <StatusBadge
              status={statusMap[pool.status] || "pending"}
            />
          </div>
        </div>

        {/* OIDC Credentials */}
        <section>
          <h2 className="mb-3 flex items-center gap-2 text-sm font-medium text-white">
            <Key className="h-4 w-4 text-neutral-500" />
            OIDC Credentials
          </h2>
          <div className="space-y-2">
            <CredentialRow label="Issuer URL" value={pool.issuer_url || "Provisioning..."} />
            <CredentialRow label="Client ID" value={pool.client_id} />
            <CredentialRow label="Client Secret" value={revealedSecret || pool.client_secret || "Click reveal to show"} secret />
            {!revealedSecret && !pool.client_secret && (
              <button
                onClick={async () => {
                  try {
                    const data = await api.revealSecret(poolId);
                    setRevealedSecret(data.client_secret);
                  } catch { /* ignore */ }
                }}
                className="mt-1 text-[11px] text-accent-400 hover:text-accent-300 transition-colors"
              >
                Reveal Secret
              </button>
            )}
          </div>
          <p className="mt-2 text-[11px] text-neutral-600">
            Use these credentials to configure OIDC in your application. Attach this pool to a Gateway route for automatic JWT validation.
          </p>
        </section>

        {/* Roles */}
        <section>
          <h2 className="mb-3 flex items-center gap-2 text-sm font-medium text-white">
            <Tag className="h-4 w-4 text-neutral-500" />
            Roles
            <span className="rounded-full bg-surface-300 px-2 py-0.5 text-[11px] text-neutral-400">
              {roles.length}
            </span>
          </h2>

          {/* Create Role */}
          <div className="mb-3 flex items-end gap-2">
            <div className="flex-1">
              <label className="mb-1 block text-[11px] font-medium text-neutral-500">Role Name</label>
              <input
                type="text"
                value={newRoleName}
                onChange={(e) => setNewRoleName(e.target.value)}
                placeholder="e.g. admin, editor, viewer"
                className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
            </div>
            <div className="flex-1">
              <label className="mb-1 block text-[11px] font-medium text-neutral-500">Description</label>
              <input
                type="text"
                value={newRoleDesc}
                onChange={(e) => setNewRoleDesc(e.target.value)}
                placeholder="Optional description"
                className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
            </div>
            <button
              onClick={handleCreateRole}
              disabled={creatingRole || !newRoleName.trim()}
              className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50"
            >
              {creatingRole ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Plus className="h-3.5 w-3.5" />}
              Add
            </button>
          </div>

          {/* Role List */}
          {rolesLoading ? (
            <div className="flex items-center justify-center py-4">
              <Loader2 className="h-4 w-4 animate-spin text-accent-500" />
            </div>
          ) : roles.length === 0 ? (
            <div className="rounded-lg border border-dashed border-border bg-surface-100 py-6 text-center">
              <Tag className="mx-auto h-6 w-6 text-neutral-600 mb-2" />
              <p className="text-xs text-neutral-400">No roles yet — create roles to control access in your app</p>
            </div>
          ) : (
            <div className="flex flex-wrap gap-2">
              {roles.map((role) => (
                <div
                  key={role.name}
                  className="group flex items-center gap-2 rounded-lg border border-border bg-surface-100 px-3 py-2"
                >
                  <Tag className="h-3 w-3 text-accent-400" />
                  <span className="text-sm font-medium text-white">{role.name}</span>
                  {role.description && (
                    <span className="text-[11px] text-neutral-500">{role.description}</span>
                  )}
                  <button
                    onClick={() => handleDeleteRole(role.name)}
                    className="ml-1 rounded p-0.5 text-neutral-600 hover:text-red-400 transition-colors opacity-0 group-hover:opacity-100"
                  >
                    <Trash2 className="h-3 w-3" />
                  </button>
                </div>
              ))}
            </div>
          )}
        </section>

        {/* Social Login Providers */}
        <section>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="flex items-center gap-2 text-sm font-medium text-white">
              <Globe className="h-4 w-4 text-neutral-500" />
              Social Login Providers
              <span className="rounded-full bg-surface-300 px-2 py-0.5 text-[11px] text-neutral-400">
                {providers.length}
              </span>
            </h2>
            <button
              onClick={() => setShowAddProvider(true)}
              className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-3 py-1.5 text-sm font-medium text-white hover:bg-accent-600 transition-colors"
            >
              <Plus className="h-3.5 w-3.5" />
              Add Provider
            </button>
          </div>

          {providersLoading ? (
            <div className="flex items-center justify-center py-4">
              <Loader2 className="h-4 w-4 animate-spin text-accent-500" />
            </div>
          ) : providers.length === 0 ? (
            <div className="rounded-lg border border-dashed border-border bg-surface-100 py-6 text-center">
              <Globe className="mx-auto h-6 w-6 text-neutral-600 mb-2" />
              <p className="text-xs text-neutral-400">No social providers configured — add Google, GitHub, or Apple to enable social login</p>
            </div>
          ) : (
            <div className="space-y-2">
              {providers.map((p) => (
                <div key={p.alias} className="group flex items-center justify-between rounded-lg border border-border bg-surface-100 px-4 py-3">
                  <div className="flex items-center gap-3">
                    <Globe className="h-4 w-4 text-accent-400" />
                    <div>
                      <p className="text-sm font-medium text-white">{p.display_name || p.alias}</p>
                      <p className="text-[11px] text-neutral-500">
                        {p.provider_id} · Client ID: {p.client_id}
                        {p.enabled ? "" : " · Disabled"}
                      </p>
                    </div>
                  </div>
                  <button
                    onClick={() => handleDeleteProvider(p.alias)}
                    className="rounded p-1.5 text-neutral-600 hover:text-red-400 transition-colors opacity-0 group-hover:opacity-100"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              ))}
            </div>
          )}
        </section>

        {/* Test Login */}
        <section>
          <h2 className="mb-3 flex items-center gap-2 text-sm font-medium text-white">
            <LogIn className="h-4 w-4 text-neutral-500" />
            Test Login
          </h2>
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <form
              onSubmit={(e) => {
                e.preventDefault();
                handleTestLogin();
              }}
              className="flex items-end gap-3"
            >
              <div className="flex-1">
                <label className="mb-1 block text-[11px] font-medium text-neutral-500">Email</label>
                <input
                  type="email"
                  value={testEmail}
                  onChange={(e) => setTestEmail(e.target.value)}
                  placeholder="user@example.com"
                  className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                  required
                />
              </div>
              <div className="flex-1">
                <label className="mb-1 block text-[11px] font-medium text-neutral-500">Password</label>
                <input
                  type="password"
                  value={testPassword}
                  onChange={(e) => setTestPassword(e.target.value)}
                  placeholder="Password"
                  className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                  required
                />
              </div>
              <button
                type="submit"
                disabled={testingLogin || !testEmail.trim() || !testPassword.trim()}
                className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50"
              >
                {testingLogin ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <LogIn className="h-3.5 w-3.5" />}
                Login
              </button>
            </form>
            {tokenError && (
              <div className="mt-3 rounded-lg bg-red-500/10 border border-red-500/20 px-3 py-2 text-xs text-red-400">
                {tokenError}
              </div>
            )}
            {tokenResult && (
              <div className="mt-3 space-y-2">
                <div className="flex items-center gap-2">
                  <Check className="h-4 w-4 text-emerald-400" />
                  <span className="text-sm font-medium text-emerald-400">Authentication successful</span>
                  <span className="text-xs text-neutral-500">Expires in {tokenResult.expires_in}s</span>
                </div>
                <div className="rounded-lg bg-surface-200 p-3">
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-[11px] font-medium text-neutral-500 uppercase tracking-wide">Access Token</span>
                    <CopyButton value={tokenResult.access_token} />
                  </div>
                  <p className="font-mono text-[11px] text-neutral-400 break-all line-clamp-3">{tokenResult.access_token}</p>
                </div>
                <div className="rounded-lg bg-surface-200 p-3">
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-[11px] font-medium text-neutral-500 uppercase tracking-wide">Refresh Token</span>
                    <CopyButton value={tokenResult.refresh_token} />
                  </div>
                  <p className="font-mono text-[11px] text-neutral-400 break-all line-clamp-2">{tokenResult.refresh_token}</p>
                </div>
              </div>
            )}
          </div>
          <p className="mt-2 text-[11px] text-neutral-600">
            Test authentication with a pool user&apos;s credentials. Tokens are issued via the OIDC password grant.
          </p>
        </section>

        {/* How to Use */}
        <section>
          <h2 className="mb-3 flex items-center gap-2 text-sm font-medium text-white">
            <BookOpen className="h-4 w-4 text-neutral-500" />
            How to Use
          </h2>
          <div className="rounded-lg border border-border bg-surface-100 p-5 space-y-4">
            <div>
              <h3 className="text-xs font-semibold text-accent-400 uppercase tracking-wide mb-2">API Endpoints</h3>
              <p className="text-xs text-neutral-400 mb-2">
                Use these endpoints from your frontend — no server SDK needed:
              </p>
              <div className="rounded-lg bg-surface-200 p-3 font-mono text-[11px] text-neutral-300 overflow-x-auto space-y-1">
                <div><span className="text-emerald-400">POST</span> <span className="text-neutral-500">/signup</span> — register + auto-login (returns tokens)</div>
                <div><span className="text-emerald-400">POST</span> <span className="text-neutral-500">/login</span> — authenticate (email + password → tokens)</div>
                <div><span className="text-emerald-400">POST</span> <span className="text-neutral-500">/refresh</span> — exchange refresh token for new tokens</div>
                <div><span className="text-emerald-400">POST</span> <span className="text-neutral-500">/logout</span> — revoke refresh token</div>
                <div><span className="text-blue-400">POST</span> <span className="text-neutral-500">/forgot-password</span> — send password reset email</div>
                <div><span className="text-blue-400">POST</span> <span className="text-neutral-500">/reset-password</span> — set new password</div>
                <div><span className="text-amber-400">GET</span>&nbsp; <span className="text-neutral-500">/user</span> — get current user profile (Bearer token)</div>
                <div><span className="text-amber-400">PUT</span>&nbsp; <span className="text-neutral-500">/user</span> — update profile (first/last name)</div>
                <div><span className="text-amber-400">POST</span> <span className="text-neutral-500">/user/password</span> — change password</div>
                <div><span className="text-amber-400">GET</span>&nbsp; <span className="text-neutral-500">/user/metadata</span> — get custom user metadata</div>
                <div><span className="text-amber-400">PUT</span>&nbsp; <span className="text-neutral-500">/user/metadata</span> — set custom user metadata</div>
                <div className="border-t border-border/30 pt-1 mt-1"></div>
                <div><span className="text-purple-400">POST</span> <span className="text-neutral-500">/anonymous</span> — anonymous sign-in (temp user + tokens)</div>
                <div><span className="text-purple-400">POST</span> <span className="text-neutral-500">/magic-link</span> — send passwordless login link</div>
                <div><span className="text-purple-400">POST</span> <span className="text-neutral-500">/magic-link/verify</span> — verify link → tokens</div>
                <div><span className="text-purple-400">GET</span>&nbsp; <span className="text-neutral-500">/authorize</span> — PKCE authorization URL</div>
                <div><span className="text-purple-400">POST</span> <span className="text-neutral-500">/token/code</span> — exchange auth code → tokens</div>
              </div>
              <p className="mt-2 text-[11px] text-neutral-500">
                Base URL: <code className="text-neutral-400">/api/v1/auth-pools/{pool?.id}</code>
              </p>
            </div>
            <div>
              <h3 className="text-xs font-semibold text-accent-400 uppercase tracking-wide mb-2">Gateway Integration</h3>
              <p className="text-xs text-neutral-400">
                Attach this pool to a <span className="text-white">Gateway</span> route — the gateway validates JWTs automatically.
                Your app receives only authenticated requests.
              </p>
            </div>
            <div>
              <h3 className="text-xs font-semibold text-accent-400 uppercase tracking-wide mb-2">Social Login</h3>
              <p className="text-xs text-neutral-400">
                Add Google, GitHub, or Apple providers above. Users can sign in with their social accounts
                alongside email/password.
              </p>
            </div>
            <div>
              <h3 className="text-xs font-semibold text-accent-400 uppercase tracking-wide mb-2">Roles &amp; Authorization</h3>
              <p className="text-xs text-neutral-400">
                Create roles and assign them to users. The JWT includes roles in the{" "}
                <code className="rounded bg-surface-300 px-1 py-0.5 text-[11px]">realm_access.roles</code> claim.
              </p>
            </div>
            <div>
              <h3 className="text-xs font-semibold text-accent-400 uppercase tracking-wide mb-2">User Metadata</h3>
              <p className="text-xs text-neutral-400">
                Store custom key-value metadata on users via the <code className="rounded bg-surface-300 px-1 py-0.5 text-[11px]">/user/metadata</code> endpoint.
                Max 20 keys, 256 chars per value.
              </p>
            </div>
            <div>
              <h3 className="text-xs font-semibold text-accent-400 uppercase tracking-wide mb-2">Passwordless (Magic Link)</h3>
              <p className="text-xs text-neutral-400">
                Send a magic link to a user&apos;s email with <code className="rounded bg-surface-300 px-1 py-0.5 text-[11px]">POST /magic-link</code>.
                Verify the token at <code className="rounded bg-surface-300 px-1 py-0.5 text-[11px]">POST /magic-link/verify</code> to get auth tokens — no password needed.
              </p>
            </div>
            <div>
              <h3 className="text-xs font-semibold text-accent-400 uppercase tracking-wide mb-2">Anonymous Sign-In</h3>
              <p className="text-xs text-neutral-400">
                Call <code className="rounded bg-surface-300 px-1 py-0.5 text-[11px]">POST /anonymous</code> to create a temporary user and get tokens instantly.
                Great for guest access or try-before-signup flows.
              </p>
            </div>
            <div>
              <h3 className="text-xs font-semibold text-accent-400 uppercase tracking-wide mb-2">PKCE / Authorization Code Flow</h3>
              <p className="text-xs text-neutral-400">
                For SPAs and mobile apps: get the authorization URL with <code className="rounded bg-surface-300 px-1 py-0.5 text-[11px]">GET /authorize</code>,
                then exchange the code at <code className="rounded bg-surface-300 px-1 py-0.5 text-[11px]">POST /token/code</code> with your code_verifier.
              </p>
            </div>
            <div>
              <h3 className="text-xs font-semibold text-accent-400 uppercase tracking-wide mb-2">Sessions &amp; MFA</h3>
              <p className="text-xs text-neutral-400">
                View and revoke user sessions from the user table. Email verification is enabled by default.
                Users can set up TOTP — manage their MFA factors from the admin panel.
              </p>
            </div>
            <div>
              <h3 className="text-xs font-semibold text-accent-400 uppercase tracking-wide mb-2">Invite Users</h3>
              <p className="text-xs text-neutral-400">
                Send invitation emails from the admin panel. Invited users receive a link to verify their email and set a password.
              </p>
            </div>
            <div>
              <h3 className="text-xs font-semibold text-accent-400 uppercase tracking-wide mb-2">Quick Start (JavaScript)</h3>
              <div className="rounded-lg bg-surface-200 p-3 font-mono text-[11px] text-neutral-300 overflow-x-auto space-y-1">
                <div className="text-neutral-500">{"// Signup"}</div>
                <div>{"const res = await fetch(`${BASE}/signup`, {"}</div>
                <div>{"  method: 'POST',"}</div>
                <div>{"  headers: { 'Content-Type': 'application/json' },"}</div>
                <div>{"  body: JSON.stringify({ email, password })"}</div>
                <div>{"}); // → { user, access_token, refresh_token }"}</div>
                <div className="text-neutral-500 mt-2">{"// Authenticated request"}</div>
                <div>{"const data = await fetch('/api/data', {"}</div>
                <div>{"  headers: { Authorization: `Bearer ${access_token}` }"}</div>
                <div>{"});"}</div>
              </div>
              <p className="mt-2 text-[11px] text-neutral-500">
                Base URL: <code className="text-neutral-400">{`${typeof window !== 'undefined' ? window.location.origin : ''}/api/v1/auth-pools/${pool?.id}`}</code>
              </p>
            </div>
            <div className="rounded-lg bg-accent-500/5 border border-accent-500/20 px-3 py-2">
              <p className="text-[11px] text-accent-400">
                Your app never talks to the identity provider directly. Zenith handles everything —
                signup, login, magic links, anonymous sessions, social login, PKCE, password resets, sessions, and user management.
              </p>
            </div>
          </div>
        </section>

        {/* Users */}
        <section>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="flex items-center gap-2 text-sm font-medium text-white">
              <Users className="h-4 w-4 text-neutral-500" />
              Users
              <span className="rounded-full bg-surface-300 px-2 py-0.5 text-[11px] text-neutral-400">
                {pool.user_count} / {pool.max_users}
              </span>
            </h2>
            <div className="flex items-center gap-2">
              <button
                onClick={async () => {
                  const email = prompt("Enter email to invite:");
                  if (email) {
                    try {
                      await api.inviteUser(poolId, email);
                      fetchUsers();
                      fetchPool();
                    } catch { /* ignore */ }
                  }
                }}
                disabled={pool.status !== "active"}
                className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm font-medium text-neutral-300 hover:text-white hover:border-accent-500 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <Mail className="h-3.5 w-3.5" />
                Invite
              </button>
              <button
                onClick={() => setShowAddUser(true)}
                disabled={pool.status !== "active"}
                className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-3 py-1.5 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <Plus className="h-3.5 w-3.5" />
                Add User
              </button>
            </div>
          </div>

          {usersLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-5 w-5 animate-spin text-accent-500" />
            </div>
          ) : users.length === 0 ? (
            <div className="rounded-lg border border-dashed border-border bg-surface-100 py-10 text-center">
              <Users className="mx-auto h-8 w-8 text-neutral-600 mb-2" />
              <p className="text-sm text-neutral-400">No users in this pool yet</p>
              <p className="text-xs text-neutral-600 mt-1">Add users to enable authentication for your app</p>
            </div>
          ) : (
            <div className="overflow-hidden rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-surface-100">
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Email</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Name</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Roles</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Created</th>
                    <th className="px-4 py-2.5 text-right text-xs font-medium text-neutral-500">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {users.map((user) => (
                    <React.Fragment key={user.id}>
                    <tr className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                      <td className="px-4 py-3 font-medium text-white">{user.email}</td>
                      <td className="px-4 py-3 text-neutral-300">
                        {[user.firstName, user.lastName].filter(Boolean).join(" ") || "—"}
                      </td>
                      <td className="px-4 py-3">
                        {user.enabled ? (
                          <span className="inline-flex items-center gap-1.5 text-xs text-emerald-400">
                            <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
                            Active
                          </span>
                        ) : (
                          <span className="inline-flex items-center gap-1.5 text-xs text-neutral-500">
                            <span className="h-1.5 w-1.5 rounded-full bg-neutral-500" />
                            Disabled
                          </span>
                        )}
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex flex-wrap items-center gap-1">
                          {(userRolesMap[user.id] || []).map((r) => (
                            <span
                              key={r.name}
                              className="group/role inline-flex items-center gap-1 rounded-full bg-accent-500/10 px-2 py-0.5 text-[11px] text-accent-400"
                            >
                              {r.name}
                              <button
                                onClick={() => handleRemoveRole(user.id, r.name)}
                                className="opacity-0 group-hover/role:opacity-100 transition-opacity"
                              >
                                <X className="h-2.5 w-2.5" />
                              </button>
                            </span>
                          ))}
                          {roles.length > 0 && (
                            <select
                              className="rounded bg-surface-300 px-1.5 py-0.5 text-[11px] text-neutral-400 border-0 focus:outline-none cursor-pointer"
                              value=""
                              onChange={(e) => {
                                if (e.target.value) handleAssignRole(user.id, e.target.value);
                              }}
                              disabled={assigningRole === user.id}
                            >
                              <option value="">+ role</option>
                              {roles
                                .filter((r) => !(userRolesMap[user.id] || []).some((ur) => ur.name === r.name))
                                .map((r) => (
                                  <option key={r.name} value={r.name}>{r.name}</option>
                                ))}
                            </select>
                          )}
                        </div>
                      </td>
                      <td className="px-4 py-3 text-xs text-neutral-500">
                        {user.createdTimestamp
                          ? new Date(user.createdTimestamp).toLocaleDateString()
                          : "—"}
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center justify-end gap-1">
                          <button
                            onClick={() => handleSendVerifyEmail(user.id)}
                            title="Send verification email"
                            className="rounded p-1.5 text-neutral-500 hover:text-blue-400 hover:bg-blue-500/10 transition-colors"
                          >
                            <Mail className="h-3.5 w-3.5" />
                          </button>
                          <button
                            onClick={() => handleExpandSessions(user.id)}
                            title="Sessions & MFA"
                            className={`rounded p-1.5 transition-colors ${expandedUserSessions === user.id ? "text-accent-400 bg-accent-500/10" : "text-neutral-500 hover:text-white hover:bg-surface-300"}`}
                          >
                            <Monitor className="h-3.5 w-3.5" />
                          </button>
                          <button
                            onClick={() => handleToggleUser(user)}
                            title={user.enabled ? "Disable user" : "Enable user"}
                            className="rounded p-1.5 text-neutral-500 hover:text-white hover:bg-surface-300 transition-colors"
                          >
                            {user.enabled ? <UserX className="h-3.5 w-3.5" /> : <UserCheck className="h-3.5 w-3.5" />}
                          </button>
                          <button
                            onClick={() => setDeleteUserTarget(user)}
                            title="Delete user"
                            className="rounded p-1.5 text-neutral-500 hover:text-red-400 hover:bg-red-500/10 transition-colors"
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </button>
                        </div>
                      </td>
                    </tr>
                    {/* Sessions & MFA expansion */}
                    {expandedUserSessions === user.id && (
                      <tr className="bg-surface-200/50">
                        <td colSpan={6} className="px-4 py-3">
                          {sessionsLoading ? (
                            <div className="flex items-center justify-center py-2">
                              <Loader2 className="h-4 w-4 animate-spin text-accent-500" />
                            </div>
                          ) : (
                            <div className="space-y-3">
                              {/* Active Sessions */}
                              <div>
                                <div className="flex items-center justify-between mb-2">
                                  <h4 className="text-xs font-medium text-neutral-400 flex items-center gap-1.5">
                                    <Monitor className="h-3 w-3" /> Active Sessions ({userSessions.length})
                                  </h4>
                                  {userSessions.length > 0 && (
                                    <button
                                      onClick={() => handleRevokeAllSessions(user.id)}
                                      className="text-[11px] text-red-400 hover:text-red-300 transition-colors"
                                    >
                                      Revoke All
                                    </button>
                                  )}
                                </div>
                                {userSessions.length === 0 ? (
                                  <p className="text-[11px] text-neutral-500">No active sessions</p>
                                ) : (
                                  <div className="space-y-1">
                                    {userSessions.map((s) => (
                                      <div key={s.id} className="flex items-center justify-between rounded bg-surface-300 px-3 py-2 text-[11px]">
                                        <div className="flex items-center gap-3">
                                          <span className="text-neutral-300">{s.ip_address}</span>
                                          <span className="text-neutral-500">
                                            Started {new Date(s.start * 1000).toLocaleString()}
                                          </span>
                                          <span className="text-neutral-500">
                                            Last: {new Date(s.last_access * 1000).toLocaleString()}
                                          </span>
                                        </div>
                                        <button
                                          onClick={() => handleRevokeSession(user.id, s.id)}
                                          className="text-red-400 hover:text-red-300"
                                        >
                                          Revoke
                                        </button>
                                      </div>
                                    ))}
                                  </div>
                                )}
                              </div>
                              {/* MFA / Credentials */}
                              <div>
                                <h4 className="text-xs font-medium text-neutral-400 flex items-center gap-1.5 mb-2">
                                  <Fingerprint className="h-3 w-3" /> Credentials & MFA
                                </h4>
                                {userCredentials.length === 0 ? (
                                  <p className="text-[11px] text-neutral-500">No credentials found</p>
                                ) : (
                                  <div className="space-y-1">
                                    {userCredentials.map((c) => (
                                      <div key={c.id} className="flex items-center justify-between rounded bg-surface-300 px-3 py-2 text-[11px]">
                                        <div className="flex items-center gap-3">
                                          <span className={`rounded px-1.5 py-0.5 font-medium ${c.type === "otp" || c.type === "totp" ? "bg-amber-500/10 text-amber-400" : "bg-neutral-500/10 text-neutral-400"}`}>
                                            {c.type.toUpperCase()}
                                          </span>
                                          <span className="text-neutral-300">{c.user_label || "—"}</span>
                                          {c.created_at > 0 && (
                                            <span className="text-neutral-500">
                                              Added {new Date(c.created_at).toLocaleDateString()}
                                            </span>
                                          )}
                                        </div>
                                        {c.type !== "password" && (
                                          <button
                                            onClick={() => handleDeleteCredential(user.id, c.id)}
                                            className="text-red-400 hover:text-red-300"
                                          >
                                            Remove
                                          </button>
                                        )}
                                      </div>
                                    ))}
                                  </div>
                                )}
                              </div>
                            </div>
                          )}
                        </td>
                      </tr>
                    )}
                    </React.Fragment>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </div>

      {/* Add User Modal */}
      {showAddUser && (
        <Modal title="Add User" onClose={() => setShowAddUser(false)}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              handleAddUser();
            }}
            className="space-y-3"
          >
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="mb-1 block text-xs font-medium text-neutral-400">First Name</label>
                <input
                  type="text"
                  value={newFirstName}
                  onChange={(e) => setNewFirstName(e.target.value)}
                  placeholder="John"
                  className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-neutral-400">Last Name</label>
                <input
                  type="text"
                  value={newLastName}
                  onChange={(e) => setNewLastName(e.target.value)}
                  placeholder="Doe"
                  className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                />
              </div>
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Email</label>
              <input
                type="email"
                value={newEmail}
                onChange={(e) => setNewEmail(e.target.value)}
                placeholder="user@example.com"
                className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                required
                autoFocus
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Password</label>
              <input
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                placeholder="Minimum 8 characters"
                className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                required
                minLength={8}
              />
            </div>
            {addUserError && (
              <div className="rounded-lg bg-red-500/10 border border-red-500/20 px-3 py-2 text-xs text-red-400">
                {addUserError}
              </div>
            )}
            <div className="flex justify-end gap-2 pt-2">
              <button
                type="button"
                onClick={() => setShowAddUser(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={addingUser || !newEmail.trim() || !newPassword.trim()}
                className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50"
              >
                {addingUser && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
                Add User
              </button>
            </div>
          </form>
        </Modal>
      )}

      {/* Add Social Provider Modal */}
      {showAddProvider && (
        <Modal title="Add Social Login Provider" onClose={() => setShowAddProvider(false)}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              handleAddProvider();
            }}
            className="space-y-3"
          >
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Provider</label>
              <select
                value={newProviderType}
                onChange={(e) => setNewProviderType(e.target.value)}
                className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
              >
                <option value="google">Google</option>
                <option value="github">GitHub</option>
                <option value="apple">Apple</option>
                <option value="microsoft">Microsoft</option>
                <option value="facebook">Facebook</option>
                <option value="twitter">Twitter / X</option>
                <option value="discord">Discord</option>
                <option value="gitlab">GitLab</option>
              </select>
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Client ID</label>
              <input
                type="text"
                value={newProviderClientId}
                onChange={(e) => setNewProviderClientId(e.target.value)}
                placeholder="OAuth client ID from the provider"
                className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                required
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Client Secret</label>
              <input
                type="password"
                value={newProviderClientSecret}
                onChange={(e) => setNewProviderClientSecret(e.target.value)}
                placeholder="OAuth client secret"
                className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                required
              />
            </div>
            <div className="flex justify-end gap-2 pt-2">
              <button
                type="button"
                onClick={() => setShowAddProvider(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={addingProvider || !newProviderClientId.trim() || !newProviderClientSecret.trim()}
                className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50"
              >
                {addingProvider && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
                Add Provider
              </button>
            </div>
          </form>
        </Modal>
      )}

      {/* Delete User Confirm */}
      {deleteUserTarget && (
        <Modal title="Delete User" onClose={() => setDeleteUserTarget(null)}>
          <div className="space-y-4">
            <p className="text-sm text-neutral-300">
              Delete <span className="font-medium text-white">{deleteUserTarget.email}</span> from this pool?
            </p>
            <div className="flex justify-end gap-2 pt-2">
              <button
                onClick={() => setDeleteUserTarget(null)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleDeleteUser}
                disabled={deletingUser}
                className="flex items-center gap-1.5 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-500 transition-colors disabled:opacity-50"
              >
                {deletingUser && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
                Delete
              </button>
            </div>
          </div>
        </Modal>
      )}
    </Shell>
  );
}
