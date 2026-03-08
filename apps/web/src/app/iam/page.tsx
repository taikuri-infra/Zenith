"use client";

import { Shell } from "@/components/shell";
import { Modal } from "@/components/modal";
import { useState, useEffect, useCallback } from "react";
import { getApi } from "@/lib/get-api";
import type { TeamMember, APIKey } from "@/lib/api";

const api = getApi();

const statusBadge: Record<string, { color: string; label: string }> = {
  active: { color: "bg-emerald-500/10 text-emerald-400", label: "Active" },
  pending: { color: "bg-amber-500/10 text-amber-400", label: "Pending" },
  suspended: { color: "bg-red-500/10 text-red-400", label: "Suspended" },
};

const scopeColors: Record<string, string> = {
  read: "bg-emerald-500/10 text-emerald-400",
  write: "bg-blue-500/10 text-blue-400",
  deploy: "bg-accent-500/10 text-accent-400",
  "registry:push": "bg-amber-500/10 text-amber-400",
  "terraform:read": "bg-cyan-500/10 text-cyan-400",
  "terraform:write": "bg-cyan-500/10 text-cyan-400",
  "s3:read": "bg-violet-500/10 text-violet-400",
  "s3:write": "bg-violet-500/10 text-violet-400",
  "*": "bg-red-500/10 text-red-400",
};

const roleCards = [
  { name: "Owner", description: "Full platform access, billing, danger zone", color: "bg-purple-500/10 text-purple-400 border-purple-500/20" },
  { name: "Admin", description: "Manage services, deployments, team", color: "bg-blue-500/10 text-blue-400 border-blue-500/20" },
  { name: "Developer", description: "Deploy, view logs, manage apps", color: "bg-emerald-500/10 text-emerald-400 border-emerald-500/20" },
  { name: "Viewer", description: "Read-only access to all resources", color: "bg-neutral-500/10 text-neutral-400 border-neutral-500/20" },
];

export default function IAMPage() {
  const [apiKeys, setApiKeys] = useState<APIKey[]>([]);
  const [teamMembers, setTeamMembers] = useState<TeamMember[]>([]);
  const [loading, setLoading] = useState(true);
  const [planLimit, setPlanLimit] = useState<number>(1);

  // API Key modal
  const [showCreateKey, setShowCreateKey] = useState(false);
  const [keyName, setKeyName] = useState("");
  const [keyScopes, setKeyScopes] = useState<string[]>(["read"]);
  const [keyType, setKeyType] = useState("personal");
  const [generatedKey, setGeneratedKey] = useState<string | null>(null);

  // Invite modal
  const [showInvite, setShowInvite] = useState(false);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState("developer");
  const [inviteError, setInviteError] = useState("");

  const loadData = useCallback(async () => {
    try {
      const [keysRes, membersRes, planRes] = await Promise.all([
        api.apiKeys.list(),
        api.team.list(),
        api.userPlan.get(),
      ]);
      setApiKeys(keysRes.items || []);
      setTeamMembers(membersRes.items || []);
      if (planRes?.limits?.max_team_members) {
        setPlanLimit(planRes.limits.max_team_members);
      }
    } catch {
      // silently handle — page will show empty states
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleCreateKey = async () => {
    if (!keyName.trim()) return;
    try {
      const key = await api.apiKeys.create(keyName.trim(), keyScopes, keyType);
      if (key.key) {
        setGeneratedKey(key.key);
      }
      setApiKeys((prev) => [...prev, key]);
    } catch {
      // handled silently
    }
  };

  const handleDeleteKey = async (id: string) => {
    try {
      await api.apiKeys.delete(id);
      setApiKeys((prev) => prev.filter((k) => k.id !== id));
    } catch {
      // handled silently
    }
  };

  const handleCloseKeyModal = () => {
    setShowCreateKey(false);
    setKeyName("");
    setKeyScopes(["read"]);
    setKeyType("personal");
    setGeneratedKey(null);
  };

  const handleInvite = async () => {
    if (!inviteEmail.trim()) return;
    setInviteError("");
    try {
      const member = await api.team.invite(inviteEmail.trim(), inviteRole);
      setTeamMembers((prev) => [...prev, member]);
      setShowInvite(false);
      setInviteEmail("");
      setInviteRole("developer");
    } catch (err) {
      setInviteError(err instanceof Error ? err.message : "Failed to send invite");
    }
  };

  const handleRemoveMember = async (id: string) => {
    try {
      await api.team.remove(id);
      setTeamMembers((prev) => prev.filter((m) => m.id !== id));
    } catch {
      // handled silently
    }
  };

  const handleUpdateRole = async (id: string, role: string) => {
    try {
      await api.team.updateRole(id, role);
      setTeamMembers((prev) => prev.map((m) => (m.id === id ? { ...m, role } : m)));
    } catch {
      // handled silently
    }
  };

  const toggleScope = (scope: string) => {
    setKeyScopes((prev) =>
      prev.includes(scope) ? prev.filter((s) => s !== scope) : [...prev, scope]
    );
  };

  const memberCount = teamMembers.length + 1; // +1 for owner

  if (loading) {
    return (
      <Shell>
        <div className="flex items-center justify-center py-20">
          <div className="h-6 w-6 animate-spin rounded-full border-2 border-accent-500 border-t-transparent" />
        </div>
      </Shell>
    );
  }

  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">IAM</h1>
          <p className="text-sm text-neutral-500">Platform identity &amp; access management</p>
        </div>

        {/* Team Members */}
        <section>
          <div className="mb-3 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <h2 className="text-sm font-medium text-white">Team Members</h2>
              <span className="text-xs text-neutral-500">{memberCount}/{planLimit}</span>
            </div>
            <button
              onClick={() => setShowInvite(true)}
              disabled={memberCount >= planLimit}
              className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              + Invite Member
            </button>
          </div>
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Email</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Role</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Joined</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500"></th>
                </tr>
              </thead>
              <tbody>
                {teamMembers.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="px-4 py-8 text-center text-sm text-neutral-500">
                      No team members yet. Invite someone to get started.
                    </td>
                  </tr>
                ) : (
                  teamMembers.map((member) => (
                    <tr key={member.id} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                      <td className="px-4 py-3 text-neutral-300">{member.email}</td>
                      <td className="px-4 py-3">
                        <select
                          value={member.role}
                          onChange={(e) => handleUpdateRole(member.id, e.target.value)}
                          className="rounded-md border border-border bg-surface-200 px-2 py-0.5 text-xs text-white focus:border-accent-500 focus:outline-none"
                        >
                          <option value="admin">Admin</option>
                          <option value="developer">Developer</option>
                          <option value="viewer">Viewer</option>
                        </select>
                      </td>
                      <td className="px-4 py-3">
                        <span className={`inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium ${statusBadge[member.status]?.color || "bg-neutral-500/10 text-neutral-400"}`}>
                          {statusBadge[member.status]?.label || member.status}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-xs text-neutral-500">
                        {new Date(member.created_at).toLocaleDateString()}
                      </td>
                      <td className="px-4 py-3 text-right">
                        <button
                          onClick={() => handleRemoveMember(member.id)}
                          className="text-xs text-red-400 hover:text-red-300 transition-colors"
                        >
                          Remove
                        </button>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </section>

        {/* API Keys */}
        <section>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-sm font-medium text-white">API Keys</h2>
            <button
              onClick={() => setShowCreateKey(true)}
              className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors"
            >
              + Create API Key
            </button>
          </div>
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Name</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Key Prefix</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Type</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Scopes</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Created</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500"></th>
                </tr>
              </thead>
              <tbody>
                {apiKeys.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="px-4 py-8 text-center text-sm text-neutral-500">
                      No API keys yet. Create one for CI/CD or programmatic access.
                    </td>
                  </tr>
                ) : (
                  apiKeys.map((key) => (
                    <tr key={key.id} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                      <td className="px-4 py-3 font-medium text-white">{key.name}</td>
                      <td className="px-4 py-3 font-mono text-xs text-neutral-400">{key.key_prefix}...</td>
                      <td className="px-4 py-3">
                        <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${key.type === "service" ? "bg-cyan-500/10 text-cyan-400" : "bg-neutral-500/10 text-neutral-400"}`}>
                          {key.type || "personal"}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex flex-wrap gap-1">
                          {key.scopes.map((scope) => (
                            <span
                              key={scope}
                              className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${scopeColors[scope] ?? "bg-surface-300 text-neutral-400"}`}
                            >
                              {scope}
                            </span>
                          ))}
                        </div>
                      </td>
                      <td className="px-4 py-3 text-xs text-neutral-500">
                        {new Date(key.created_at).toLocaleDateString()}
                      </td>
                      <td className="px-4 py-3 text-right">
                        <button
                          onClick={() => handleDeleteKey(key.id)}
                          className="text-xs text-red-400 hover:text-red-300 transition-colors"
                        >
                          Revoke
                        </button>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </section>

        {/* Roles */}
        <section>
          <div className="mb-3">
            <h2 className="text-sm font-medium text-white">Roles</h2>
          </div>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {roleCards.map((role) => (
              <div key={role.name} className={`rounded-lg border p-4 ${role.color}`}>
                <p className="text-sm font-medium">{role.name}</p>
                <p className="mt-1 text-xs opacity-70">{role.description}</p>
              </div>
            ))}
          </div>
        </section>
      </div>

      {showCreateKey && (
        <Modal title="Create API Key" onClose={handleCloseKeyModal}>
          {generatedKey ? (
            <div className="space-y-3">
              <div className="rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-4 py-3">
                <p className="text-xs text-emerald-400 mb-2">API key created successfully. Copy it now — you will not be able to see it again.</p>
                <code className="block rounded bg-surface-200 p-2 font-mono text-sm text-white break-all">{generatedKey}</code>
              </div>
              <div className="flex justify-end pt-4">
                <button onClick={handleCloseKeyModal} className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors">Done</button>
              </div>
            </div>
          ) : (
            <form onSubmit={(e) => { e.preventDefault(); handleCreateKey(); }} className="space-y-3">
              <div>
                <label className="mb-1 block text-xs font-medium text-neutral-400">Key Name</label>
                <input
                  type="text"
                  value={keyName}
                  onChange={(e) => setKeyName(e.target.value)}
                  placeholder="My API Key"
                  className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                  required
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-neutral-400">Type</label>
                <select
                  value={keyType}
                  onChange={(e) => setKeyType(e.target.value)}
                  className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
                >
                  <option value="personal">Personal</option>
                  <option value="service">Service Token</option>
                </select>
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-neutral-400">Scopes</label>
                <div className="flex flex-wrap gap-2 mt-1">
                  {["read", "write", "deploy", "registry:push", "terraform:read", "terraform:write", "s3:read", "s3:write"].map((scope) => (
                    <button
                      key={scope}
                      type="button"
                      onClick={() => toggleScope(scope)}
                      className={`rounded-full px-2.5 py-1 text-xs font-medium border transition-colors ${
                        keyScopes.includes(scope)
                          ? "border-accent-500 bg-accent-500/20 text-accent-400"
                          : "border-border bg-surface-200 text-neutral-500 hover:text-neutral-300"
                      }`}
                    >
                      {scope}
                    </button>
                  ))}
                </div>
              </div>
              <div className="flex justify-end gap-2 pt-4">
                <button type="button" onClick={handleCloseKeyModal} className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors">Cancel</button>
                <button type="submit" className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors">Create Key</button>
              </div>
            </form>
          )}
        </Modal>
      )}

      {showInvite && (
        <Modal title="Invite Member" onClose={() => { setShowInvite(false); setInviteError(""); }}>
          <form onSubmit={(e) => { e.preventDefault(); handleInvite(); }} className="space-y-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Email</label>
              <input
                type="email"
                value={inviteEmail}
                onChange={(e) => setInviteEmail(e.target.value)}
                placeholder="teammate@company.com"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                required
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Role</label>
              <select
                value={inviteRole}
                onChange={(e) => setInviteRole(e.target.value)}
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
              >
                <option value="admin">Admin</option>
                <option value="developer">Developer</option>
                <option value="viewer">Viewer</option>
              </select>
            </div>
            {inviteError && (
              <p className="text-xs text-red-400">{inviteError}</p>
            )}
            <div className="flex justify-end gap-2 pt-4">
              <button type="button" onClick={() => { setShowInvite(false); setInviteError(""); }} className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors">Cancel</button>
              <button type="submit" className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors">Send Invite</button>
            </div>
          </form>
        </Modal>
      )}
    </Shell>
  );
}
