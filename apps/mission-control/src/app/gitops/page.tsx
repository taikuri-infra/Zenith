"use client";

import { useState } from "react";
import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { StatusBadge } from "@/components/status-badge";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { StatCardRowSkeleton, TableSkeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import type { ArgoApp, GitOpsStats } from "@/lib/api";
import { useApi, useMutation } from "@/hooks/use-api";
import { GitBranch, RefreshCw } from "lucide-react";

function syncStatusBadge(status: string): "healthy" | "warning" | "error" | "idle" {
  switch (status) {
    case "Synced":
      return "healthy";
    case "OutOfSync":
      return "warning";
    case "Unknown":
      return "idle";
    default:
      return "idle";
  }
}

function healthStatusBadge(health: string): "healthy" | "warning" | "error" | "idle" {
  switch (health) {
    case "Healthy":
      return "healthy";
    case "Degraded":
      return "warning";
    case "Progressing":
      return "warning";
    case "Suspended":
      return "idle";
    case "Missing":
    case "Unknown":
      return "error";
    default:
      return "idle";
  }
}

export default function GitOpsPage() {
  const apiClient = getApi();
  const stats = useApi<GitOpsStats>(() => apiClient.gitops.stats());
  const { data: apps, loading, error, refetch } = useApi<ArgoApp[]>(
    () => apiClient.gitops.list()
  );
  const [syncingApp, setSyncingApp] = useState<string | null>(null);

  const syncMutation = useMutation<string, void>(
    (appName) => apiClient.gitops.sync(appName)
  );

  const handleSync = async (appName: string) => {
    setSyncingApp(appName);
    try {
      await syncMutation.execute(appName);
      refetch();
    } finally {
      setSyncingApp(null);
    }
  };

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-lg font-semibold text-white">GitOps</h1>
          <button
            onClick={refetch}
            className="flex items-center gap-1.5 rounded-lg border border-border bg-surface-100 px-3 py-1.5 text-sm text-neutral-400 hover:bg-surface-200 hover:text-white transition-colors"
          >
            <RefreshCw className="h-3.5 w-3.5" />
            Refresh
          </button>
        </div>

        {/* Stats */}
        {stats.loading ? (
          <StatCardRowSkeleton />
        ) : stats.data ? (
          <div className="grid grid-cols-4 gap-4">
            <StatCard label="Applications" value={stats.data.totalApps} sub="ArgoCD managed" />
            <StatCard label="Synced" value={stats.data.synced} sub="in sync" />
            <StatCard label="Out of Sync" value={stats.data.outOfSync} sub="need sync" alert={stats.data.outOfSync > 0} />
            <StatCard label="Degraded" value={stats.data.degraded} sub="unhealthy" alert={stats.data.degraded > 0} />
          </div>
        ) : null}

        {/* ArgoCD Apps Table */}
        {loading ? (
          <TableSkeleton columns={7} rows={5} />
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : !apps || apps.length === 0 ? (
          <EmptyState
            title="No ArgoCD applications"
            description="No applications are managed by ArgoCD."
            icon={GitBranch}
          />
        ) : (
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Name</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Namespace</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Health</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Sync Status</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Revision</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Last Synced</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Actions</th>
                </tr>
              </thead>
              <tbody>
                {apps.map((app) => (
                  <tr key={app.name} className="border-b border-border last:border-0 transition-colors hover:bg-surface-200">
                    <td className="px-4 py-3">
                      <span className="font-medium text-white">{app.name}</span>
                      {app.project && (
                        <span className="ml-2 text-xs text-neutral-500">{app.project}</span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-neutral-400">{app.namespace}</td>
                    <td className="px-4 py-3">
                      <StatusBadge status={healthStatusBadge(app.healthStatus)} label={app.healthStatus} />
                    </td>
                    <td className="px-4 py-3">
                      <StatusBadge status={syncStatusBadge(app.syncStatus)} label={app.syncStatus} />
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-neutral-400">
                      {app.revision ? app.revision.slice(0, 7) : "—"}
                    </td>
                    <td className="px-4 py-3 text-xs text-neutral-500">{app.lastSynced || "—"}</td>
                    <td className="px-4 py-3">
                      <button
                        onClick={() => handleSync(app.name)}
                        disabled={syncingApp === app.name}
                        className="flex items-center gap-1 rounded-md border border-border bg-surface-100 px-2 py-1 text-xs text-neutral-400 hover:bg-surface-200 hover:text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        {syncingApp === app.name ? (
                          <div className="h-3 w-3 animate-spin rounded-full border border-neutral-400/30 border-t-neutral-400" />
                        ) : (
                          <RefreshCw className="h-3 w-3" />
                        )}
                        Sync
                      </button>
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
