"use client";

import { useState } from "react";
import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { TableSkeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import type { AdminUser } from "@/lib/api";
import { useApi, useMutation } from "@/hooks/use-api";
import { UserCog, Plus, X } from "lucide-react";

const roleColors: Record<string, string> = {
  owner: "bg-amber-500/15 text-amber-400",
  admin: "bg-accent-500/15 text-accent-400",
  operator: "bg-blue-500/15 text-blue-400",
  viewer: "bg-neutral-500/10 text-neutral-400",
};

const roleOptions = ["admin", "operator", "viewer"];

export default function AdminUsersPage() {
  const apiClient = getApi();
  const { data: users, loading, error, refetch } = useApi<AdminUser[]>(
    () => apiClient.adminUsers.list()
  );
  const [showInvite, setShowInvite] = useState(false);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState("operator");

  const inviteMutation = useMutation<{ email: string; role: string }, void>(
    (input) => apiClient.adminUsers.invite(input.email, input.role)
  );

  const removeMutation = useMutation<string, void>(
    (userId) => apiClient.adminUsers.remove(userId)
  );

  const changeRoleMutation = useMutation<{ userId: string; role: string }, void>(
    (input) => apiClient.adminUsers.changeRole(input.userId, input.role)
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
      await changeRoleMutation.execute({ userId, role: newRole });
      refetch();
    } catch {
      // Error handled by mutation
    }
  };

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-lg font-semibold text-white">Admin Users</h1>
          <button
            onClick={() => setShowInvite(!showInvite)}
            className="flex items-center gap-1.5 rounded-lg bg-accent-600 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-accent-500"
          >
            <Plus className="h-3.5 w-3.5" />
            Invite Admin
          </button>
        </div>

        {/* Invite Form */}
        {showInvite && (
          <div className="rounded-lg border border-accent-600/30 bg-accent-600/5 p-4">
            <div className="flex items-center justify-between mb-3">
              <h2 className="text-sm font-medium text-white">Invite Admin User</h2>
              <button
                onClick={() => setShowInvite(false)}
                className="text-neutral-500 hover:text-white transition-colors"
              >
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
          <EmptyState
            title="No admin users"
            description="No admin users have been configured."
            icon={UserCog}
          />
        ) : (
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Email</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Name</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Role</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Permissions</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Granted By</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Actions</th>
                </tr>
              </thead>
              <tbody>
                {users.map((user) => (
                  <tr key={user.id} className="border-b border-border last:border-0 transition-colors hover:bg-surface-200">
                    <td className="px-4 py-3">
                      <span className="font-medium text-white">{user.email}</span>
                      {user.mfaEnabled && (
                        <span className="ml-2 rounded bg-emerald-500/15 px-1.5 py-0.5 text-[10px] font-medium text-emerald-400">
                          MFA
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-neutral-300">{user.name}</td>
                    <td className="px-4 py-3">
                      <span className={`rounded-full px-2 py-0.5 text-xs font-medium capitalize ${roleColors[user.role] || "bg-neutral-500/10 text-neutral-400"}`}>
                        {user.role}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-neutral-400 text-xs">{user.permissionCount} permissions</td>
                    <td className="px-4 py-3 text-neutral-500 text-xs">{user.grantedBy || "—"}</td>
                    <td className="px-4 py-3">
                      {user.role !== "owner" && (
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
                      )}
                      {user.role === "owner" && (
                        <span className="text-xs text-neutral-600">Protected</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </Shell>
  );
}
