"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { Modal } from "@/components/modal";
import { useState, useEffect, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import { getApi } from "@/lib/get-api";
import type { AuthPool, AuthPoolUser } from "@/lib/api";
import {
  Shield, Users, Key, Copy, Check, ChevronLeft, Plus,
  Trash2, Loader2, UserCheck, UserX, Eye, EyeOff, LogIn, BookOpen,
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
  }, [api, poolId]);

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
  }, [api, poolId]);

  useEffect(() => {
    fetchPool();
    fetchUsers();
  }, [fetchPool, fetchUsers]);

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
      const result = await api.token(poolId, testEmail.trim(), testPassword);
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
            <CredentialRow label="Client Secret" value={pool.client_secret || "Hidden — shown only on creation"} secret />
          </div>
          <p className="mt-2 text-[11px] text-neutral-600">
            Use these credentials to configure OIDC in your application. Attach this pool to a Gateway route for automatic JWT validation.
          </p>
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
            Test authentication with a pool user&apos;s credentials. Tokens are issued by Keycloak via the OIDC password grant.
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
              <h3 className="text-xs font-semibold text-accent-400 uppercase tracking-wide mb-2">1. Create Users</h3>
              <p className="text-xs text-neutral-400">
                Add users to this pool using the form below. Each user gets an email/password credential managed by Zenith.
              </p>
            </div>
            <div>
              <h3 className="text-xs font-semibold text-accent-400 uppercase tracking-wide mb-2">2. Connect to Gateway</h3>
              <p className="text-xs text-neutral-400">
                In your <span className="text-white">Gateway</span> settings, attach this auth pool to a route.
                The gateway will automatically validate JWT tokens on every request — no code changes needed in your app.
              </p>
            </div>
            <div>
              <h3 className="text-xs font-semibold text-accent-400 uppercase tracking-wide mb-2">3. Authenticate Users</h3>
              <p className="text-xs text-neutral-400 mb-2">
                Your frontend calls the token endpoint to log users in:
              </p>
              <div className="rounded-lg bg-surface-200 p-3 font-mono text-[11px] text-neutral-300 overflow-x-auto">
                <div className="text-neutral-500">POST /api/v1/auth-pools/{pool?.id}/token</div>
                <div className="mt-1 text-neutral-400">{'{'} &quot;username&quot;: &quot;user@example.com&quot;, &quot;password&quot;: &quot;...&quot; {'}'}</div>
                <div className="mt-2 text-neutral-500">→ Returns: access_token, refresh_token, expires_in</div>
              </div>
            </div>
            <div>
              <h3 className="text-xs font-semibold text-accent-400 uppercase tracking-wide mb-2">4. Use Roles for Authorization</h3>
              <p className="text-xs text-neutral-400">
                The JWT token contains user roles from Keycloak. Use these roles in your app logic for fine-grained authorization.
                The gateway handles authentication — your app only needs to read the roles from the validated token.
              </p>
            </div>
            <div className="rounded-lg bg-accent-500/5 border border-accent-500/20 px-3 py-2">
              <p className="text-[11px] text-accent-400">
                Your app never talks to Keycloak directly. Zenith handles the entire auth infrastructure —
                token issuance, validation, and user management.
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
            <button
              onClick={() => setShowAddUser(true)}
              disabled={pool.status !== "active"}
              className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-3 py-1.5 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <Plus className="h-3.5 w-3.5" />
              Add User
            </button>
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
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Created</th>
                    <th className="px-4 py-2.5 text-right text-xs font-medium text-neutral-500">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {users.map((user) => (
                    <tr key={user.id} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
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
                      <td className="px-4 py-3 text-xs text-neutral-500">
                        {user.createdTimestamp
                          ? new Date(user.createdTimestamp).toLocaleDateString()
                          : "—"}
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center justify-end gap-1">
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
