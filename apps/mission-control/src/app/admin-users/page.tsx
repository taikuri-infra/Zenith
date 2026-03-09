"use client";

import { useState } from "react";
import { Shell } from "@/components/shell";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { TableSkeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import { demoApi } from "@/lib/demo-api";
import type { AdminUser } from "@/lib/api";
import { useApiWithFallback, useMutation } from "@/hooks/use-api";
import { UserCog, Plus, X, Shield, ShieldCheck, Eye, Settings } from "lucide-react";

const roleConfig: Record<string, { color: string; icon: typeof Shield; description: string }> = {
  owner: { color: "bg-amber-500/15 text-amber-400 border-amber-500/20", icon: Settings, description: "Full access, user management, destructive actions" },
  admin: { color: "bg-accent-500/15 text-accent-400 border-accent-500/20", icon: ShieldCheck, description: "Everything except admin management & platform settings" },
  operator: { color: "bg-blue-500/15 text-blue-400 border-blue-500/20", icon: Shield, description: "Customers, support, CRM, infrastructure (read-only)" },
  viewer: { color: "bg-neutral-500/10 text-neutral-400 border-neutral-500/20", icon: Eye, description: "Read-only access to all dashboards" },
};

const roleOptions = ["admin", "operator", "viewer"];

const permissionGroups = [
  { group: "War Room", owner: "RW", admin: "RW", operator: "R", viewer: "R" },
  { group: "Analytics", owner: "RW", admin: "RW", operator: "R", viewer: "R" },
  { group: "Customers", owner: "RW", admin: "RW", operator: "RW", viewer: "R" },
  { group: "CRM", owner: "RW", admin: "RW", operator: "RW", viewer: "R" },
  { group: "Support", owner: "RW", admin: "RW", operator: "RW", viewer: "R" },
  { group: "Infrastructure", owner: "RW", admin: "RW", operator: "R", viewer: "R" },
  { group: "Observability", owner: "RW", admin: "RW", operator: "—", viewer: "R" },
  { group: "Security", owner: "RW", admin: "RW", operator: "—", viewer: "R" },
  { group: "Modules", owner: "RW", admin: "RW", operator: "—", viewer: "—" },
  { group: "Backups", owner: "RW", admin: "RW", operator: "—", viewer: "R" },
  { group: "Admin Settings", owner: "RW", admin: "—", operator: "—", viewer: "—" },
];

export default function AdminUsersPage() {
  const apiClient = getApi();
  const { data: users, loading, error, refetch, isDemo } = useApiWithFallback<AdminUser[]>(
    () => apiClient.adminUsers.list(),
    () => demoApi.adminUsers.list()
  );
  const [showInvite, setShowInvite] = useState(false);
  const [showPermissions, setShowPermissions] = useState(false);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState("operator");

  const inviteMutation = useMutation<{ email: string; role: string }, void>(
    (input) => apiClient.adminUsers.invite(input.email, input.role)
  );

  const removeMutation = useMutation<string, void>(
    (userId) => apiClient.adminUsers.remove(userId)
  );

  const handleInvite = async () => {
    if (!inviteEmail.trim()) return;
    try {
      await inviteMutation.execute({ email: inviteEmail, role: inviteRole });
      setInviteEmail("");
      setShowInvite(false);
      refetch();
    } catch {
      // Error is displayed via inviteMutation.error
    }
  };

  const handleRemove = async (userId: string, email: string) => {
    if (!confirm(`Remove admin access for ${email}?`)) return;
    try {
      await removeMutation.execute(userId);
      refetch();
    } catch {
      // Error handled by mutation
    }
  };

  const handleRoleChange = async (userId: string, newRole: string) => {
    try {
      await apiClient.adminUsers.changeRole(userId, newRole);
      refetch();
    } catch {
      // Error handled
    }
  };

  return (
    <Shell>
      <div className="space-y-8">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Admin Users</h1>
            <p className="mt-1 text-sm text-neutral-500">
              Manage who can access Mission Control
            </p>
            {isDemo && (
              <p className="mt-1 text-xs text-amber-400/70">Showing sample data</p>
            )}
          </div>
          <div className="flex gap-2">
            <button
              onClick={() => setShowPermissions(!showPermissions)}
              className="flex items-center gap-1.5 rounded-lg border border-border bg-surface-100 px-3 py-1.5 text-sm text-neutral-400 hover:bg-surface-200 hover:text-white transition-colors"
            >
              <Shield className="h-3.5 w-3.5" />
              Permissions
            </button>
            <button
              onClick={() => setShowInvite(!showInvite)}
              className="flex items-center gap-1.5 rounded-lg bg-accent-600 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-accent-500"
            >
              <Plus className="h-3.5 w-3.5" />
              Invite Admin
            </button>
          </div>
        </div>

        {/* Role Cards */}
        <div className="grid grid-cols-4 gap-4">
          {Object.entries(roleConfig).map(([role, config]) => (
            <div
              key={role}
              className={`rounded-xl border p-4 ${config.color}`}
            >
              <config.icon className="h-5 w-5 mb-2" />
              <h3 className="text-sm font-semibold capitalize text-white">{role}</h3>
              <p className="mt-1 text-xs text-neutral-500">{config.description}</p>
              {users && (
                <p className="mt-2 text-xs font-mono text-neutral-400">
                  {users.filter((u) => u.role === role).length} user{users.filter((u) => u.role === role).length !== 1 ? "s" : ""}
                </p>
              )}
            </div>
          ))}
        </div>

        {/* Permission Matrix */}
        {showPermissions && (
          <section className="rounded-xl border border-border bg-surface-100 p-5">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-sm font-medium text-white">Permission Matrix</h2>
              <button onClick={() => setShowPermissions(false)} className="text-neutral-500 hover:text-white transition-colors">
                <X className="h-4 w-4" />
              </button>
            </div>
            <div className="overflow-hidden rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-surface-200">
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Group</th>
                    <th className="px-4 py-2.5 text-center text-xs font-medium text-amber-400">Owner</th>
                    <th className="px-4 py-2.5 text-center text-xs font-medium text-accent-400">Admin</th>
                    <th className="px-4 py-2.5 text-center text-xs font-medium text-blue-400">Operator</th>
                    <th className="px-4 py-2.5 text-center text-xs font-medium text-neutral-400">Viewer</th>
                  </tr>
                </thead>
                <tbody>
                  {permissionGroups.map((pg) => (
                    <tr key={pg.group} className="border-b border-border last:border-0">
                      <td className="px-4 py-2 text-white">{pg.group}</td>
                      {["owner", "admin", "operator", "viewer"].map((role) => {
                        const val = pg[role as keyof typeof pg];
                        return (
                          <td key={role} className="px-4 py-2 text-center">
                            <span className={`text-xs font-mono ${val === "RW" ? "text-emerald-400" : val === "R" ? "text-neutral-400" : "text-neutral-600"}`}>
                              {val}
                            </span>
                          </td>
                        );
                      })}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        )}

        {/* Invite Form */}
        {showInvite && (
          <div className="rounded-xl border border-accent-600/30 bg-accent-600/5 p-5">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-sm font-medium text-white">Invite Admin User</h2>
              <button onClick={() => setShowInvite(false)} className="text-neutral-500 hover:text-white transition-colors">
                <X className="h-4 w-4" />
              </button>
            </div>
            <div className="flex gap-3">
              <input
                type="email"
                value={inviteEmail}
                onChange={(e) => setInviteEmail(e.target.value)}
                placeholder="admin@example.com"
                className="flex-1 rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
              <select
                value={inviteRole}
                onChange={(e) => setInviteRole(e.target.value)}
                className="rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
              >
                {roleOptions.map((role) => (
                  <option key={role} value={role}>
                    {role.charAt(0).toUpperCase() + role.slice(1)}
                  </option>
                ))}
              </select>
              <button
                onClick={handleInvite}
                disabled={inviteMutation.loading || !inviteEmail.trim()}
                className="rounded-lg bg-accent-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-accent-500 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {inviteMutation.loading ? "Sending..." : "Send Invite"}
              </button>
            </div>
            {inviteMutation.error && (
              <p className="mt-2 text-xs text-red-400">{inviteMutation.error.message}</p>
            )}
          </div>
        )}

        {/* Users Table */}
        {loading ? (
          <TableSkeleton columns={6} rows={4} />
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : !users || users.length === 0 ? (
          <EmptyState title="No admin users" description="No admin users have been configured." icon={UserCog} />
        ) : (
          <div className="overflow-hidden rounded-xl border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">User</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Role</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Permissions</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Security</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Granted By</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Actions</th>
                </tr>
              </thead>
              <tbody>
                {users.map((user) => {
                  const config = roleConfig[user.role] || roleConfig.viewer;
                  return (
                    <tr key={user.id} className="border-b border-border last:border-0 transition-colors hover:bg-surface-200">
                      <td className="px-4 py-3">
                        <div>
                          <span className="font-medium text-white">{user.email}</span>
                          {user.name && <span className="ml-2 text-xs text-neutral-500">{user.name}</span>}
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <span className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium capitalize ${config.color}`}>
                          <config.icon className="h-3 w-3" />
                          {user.role}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-neutral-400 text-xs">{user.permissionCount} groups</td>
                      <td className="px-4 py-3">
                        {user.mfaEnabled ? (
                          <span className="rounded bg-emerald-500/15 px-1.5 py-0.5 text-[10px] font-medium text-emerald-400">MFA</span>
                        ) : (
                          <span className="rounded bg-red-500/10 px-1.5 py-0.5 text-[10px] font-medium text-red-400">No MFA</span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-neutral-500 text-xs">{user.grantedBy || "System"}</td>
                      <td className="px-4 py-3">
                        {user.role !== "owner" ? (
                          <div className="flex items-center gap-2">
                            <select
                              defaultValue={user.role}
                              onChange={(e) => handleRoleChange(user.id, e.target.value)}
                              className="rounded border border-border bg-surface-200 px-2 py-1 text-xs text-neutral-300 focus:border-accent-500 focus:outline-none"
                            >
                              {roleOptions.map((role) => (
                                <option key={role} value={role}>
                                  {role.charAt(0).toUpperCase() + role.slice(1)}
                                </option>
                              ))}
                            </select>
                            <button
                              onClick={() => handleRemove(user.id, user.email)}
                              className="rounded-md border border-red-500/30 bg-red-500/10 p-1 text-red-400 hover:bg-red-500/20 transition-colors"
                            >
                              <X className="h-3 w-3" />
                            </button>
                          </div>
                        ) : (
                          <span className="text-xs text-neutral-600 italic">Protected</span>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </Shell>
  );
}
