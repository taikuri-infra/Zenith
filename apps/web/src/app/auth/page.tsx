"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { Modal } from "@/components/modal";
import { useState, useEffect, useCallback } from "react";
import { getApi } from "@/lib/get-api";
import type { AuthPool } from "@/lib/api";
import { Shield, Plus, Trash2, Users, Key, Loader2 } from "lucide-react";
import Link from "next/link";

export default function AuthPage() {
  const { authPools: api } = getApi();
  const [pools, setPools] = useState<AuthPool[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Create modal
  const [showCreate, setShowCreate] = useState(false);
  const [poolName, setPoolName] = useState("");
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);

  // Delete confirm
  const [deleteTarget, setDeleteTarget] = useState<AuthPool | null>(null);
  const [deleting, setDeleting] = useState(false);

  const fetchPools = useCallback(async () => {
    try {
      setLoading(true);
      const data = await api.list();
      setPools(Array.isArray(data) ? data : []);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load auth pools");
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    fetchPools();
  }, [fetchPools]);

  const handleCreate = async () => {
    if (!poolName.trim()) return;
    setCreating(true);
    setCreateError(null);
    try {
      await api.create(poolName.trim());
      setShowCreate(false);
      setPoolName("");
      await fetchPools();
    } catch (e) {
      setCreateError(e instanceof Error ? e.message : "Failed to create pool");
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await api.delete(deleteTarget.id);
      setDeleteTarget(null);
      await fetchPools();
    } catch {
      // ignore
    } finally {
      setDeleting(false);
    }
  };

  const statusMap: Record<string, "active" | "provisioning" | "error" | "deleting"> = {
    active: "active",
    provisioning: "provisioning",
    error: "error",
    deleting: "deleting",
  };

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Auth Pools</h1>
            <p className="text-sm text-neutral-500">
              Managed authentication and authorization for your applications
            </p>
          </div>
          <button
            onClick={() => setShowCreate(true)}
            className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors"
          >
            <Plus className="h-4 w-4" />
            Create Pool
          </button>
        </div>

        {/* Info Banner */}
        <div className="rounded-lg border border-accent-500/30 bg-accent-500/5 px-4 py-3">
          <p className="text-xs text-accent-400">
            Each pool creates an isolated authentication realm with OIDC support.
            Attach pools to your Gateway routes for automatic JWT validation.
          </p>
        </div>

        {/* Pool List */}
        {loading ? (
          <div className="flex items-center justify-center py-16">
            <Loader2 className="h-6 w-6 animate-spin text-accent-500" />
          </div>
        ) : error ? (
          <div className="rounded-lg border border-red-500/20 bg-red-500/5 p-6 text-center">
            <p className="text-sm text-red-400">{error}</p>
            <button
              onClick={fetchPools}
              className="mt-3 rounded-lg border border-border px-4 py-1.5 text-sm text-neutral-400 hover:text-white transition-colors"
            >
              Retry
            </button>
          </div>
        ) : pools.length === 0 ? (
          <div className="rounded-lg border border-dashed border-border bg-surface-100 py-16 text-center">
            <Shield className="mx-auto h-10 w-10 text-neutral-600 mb-3" />
            <h3 className="text-sm font-medium text-white mb-1">No auth pools yet</h3>
            <p className="text-xs text-neutral-500 mb-4">
              Create your first pool to add managed authentication to your apps.
            </p>
            <button
              onClick={() => setShowCreate(true)}
              className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors"
            >
              Create Pool
            </button>
          </div>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {pools.map((pool) => (
              <Link
                key={pool.id}
                href={`/auth/${pool.id}`}
                className="group rounded-lg border border-border bg-surface-100 p-5 hover:bg-surface-200 hover:border-accent-500/30 transition-colors"
              >
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-2.5">
                    <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-accent-500/10">
                      <Shield className="h-4.5 w-4.5 text-accent-400" />
                    </div>
                    <div>
                      <h3 className="text-sm font-medium text-white group-hover:text-accent-400 transition-colors">
                        {pool.name}
                      </h3>
                      <p className="text-[11px] text-neutral-500">{pool.realm_name}</p>
                    </div>
                  </div>
                  <StatusBadge
                    status={statusMap[pool.status] || "pending"}
                  />
                </div>

                <div className="mt-4 grid grid-cols-2 gap-3">
                  <div className="rounded-md bg-surface-200 px-3 py-2">
                    <div className="flex items-center gap-1.5">
                      <Users className="h-3 w-3 text-neutral-500" />
                      <span className="text-[11px] text-neutral-500">Users</span>
                    </div>
                    <p className="mt-0.5 text-sm font-medium text-white">
                      {pool.user_count}
                      <span className="text-neutral-500 font-normal text-xs"> / {pool.max_users}</span>
                    </p>
                  </div>
                  <div className="rounded-md bg-surface-200 px-3 py-2">
                    <div className="flex items-center gap-1.5">
                      <Key className="h-3 w-3 text-neutral-500" />
                      <span className="text-[11px] text-neutral-500">Protocol</span>
                    </div>
                    <p className="mt-0.5 text-sm font-medium text-white">OIDC</p>
                  </div>
                </div>

                <div className="mt-3 flex items-center justify-between">
                  <span className="font-mono text-[10px] text-neutral-600 truncate max-w-[200px]">
                    {pool.issuer_url || "Provisioning..."}
                  </span>
                  <button
                    onClick={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      setDeleteTarget(pool);
                    }}
                    className="rounded p-1 text-neutral-600 hover:text-red-400 hover:bg-red-500/10 transition-colors"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              </Link>
            ))}
          </div>
        )}
      </div>

      {/* Create Modal */}
      {showCreate && (
        <Modal title="Create Auth Pool" onClose={() => setShowCreate(false)}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              handleCreate();
            }}
            className="space-y-4"
          >
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">
                Pool Name
              </label>
              <input
                type="text"
                value={poolName}
                onChange={(e) => setPoolName(e.target.value)}
                placeholder="e.g. production, staging, my-saas"
                className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2.5 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                required
                autoFocus
              />
              <p className="mt-1.5 text-[11px] text-neutral-500">
                A Keycloak realm and OIDC client will be provisioned automatically.
              </p>
            </div>
            {createError && (
              <div className="rounded-lg bg-red-500/10 border border-red-500/20 px-3 py-2 text-xs text-red-400">
                {createError}
              </div>
            )}
            <div className="flex justify-end gap-2 pt-2">
              <button
                type="button"
                onClick={() => setShowCreate(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={creating || !poolName.trim()}
                className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50"
              >
                {creating && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
                Create Pool
              </button>
            </div>
          </form>
        </Modal>
      )}

      {/* Delete Confirm Modal */}
      {deleteTarget && (
        <Modal title="Delete Auth Pool" onClose={() => setDeleteTarget(null)}>
          <div className="space-y-4">
            <p className="text-sm text-neutral-300">
              Delete <span className="font-medium text-white">{deleteTarget.name}</span>?
              This will remove the Keycloak realm and all users in it.
            </p>
            <div className="rounded-lg bg-red-500/10 border border-red-500/20 px-3 py-2 text-xs text-red-400">
              This action cannot be undone. All users and OIDC clients will be permanently deleted.
            </div>
            <div className="flex justify-end gap-2 pt-2">
              <button
                onClick={() => setDeleteTarget(null)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleDelete}
                disabled={deleting}
                className="flex items-center gap-1.5 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-500 transition-colors disabled:opacity-50"
              >
                {deleting && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
                Delete Pool
              </button>
            </div>
          </div>
        </Modal>
      )}
    </Shell>
  );
}
