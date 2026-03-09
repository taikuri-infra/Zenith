"use client";

import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { StatusBadge } from "@/components/status-badge";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { StatCardRowSkeleton, TableSkeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import { demoApi } from "@/lib/demo-api";
import type { DatabaseCluster, DatabaseStats } from "@/lib/api";
import { useApi, useApiWithFallback } from "@/hooks/use-api";
import { Database } from "lucide-react";

function dbStatusBadge(status: string): "healthy" | "warning" | "error" | "idle" {
  switch (status) {
    case "healthy":
    case "running":
      return "healthy";
    case "creating":
    case "upgrading":
      return "warning";
    case "failed":
      return "error";
    default:
      return "idle";
  }
}

export default function DatabasesPage() {
  const apiClient = getApi();
  const stats = useApiWithFallback<DatabaseStats>(
    () => apiClient.databases.stats(),
    () => demoApi.databases.stats(),
    (data) => !data || data.totalStorage === "0 Gi"
  );
  const { data: clusters, loading, error, refetch, isDemo } = useApiWithFallback<DatabaseCluster[]>(
    () => apiClient.databases.list(),
    () => demoApi.databases.list()
  );

  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">Databases</h1>
          {isDemo && (
            <p className="mt-1 text-xs text-amber-400/70">Showing sample data</p>
          )}
        </div>

        {/* Stats */}
        {stats.loading ? (
          <StatCardRowSkeleton />
        ) : stats.data ? (
          <div className="grid grid-cols-4 gap-4">
            <StatCard label="Total Clusters" value={stats.data.totalClusters} sub="CNPG managed" />
            <StatCard label="Healthy" value={stats.data.healthyClusters} sub="operational" />
            <StatCard label="Total Storage" value={stats.data.totalStorage} sub="allocated" />
            <StatCard label="Last Backup" value={stats.data.lastBackup ? new Date(stats.data.lastBackup).toLocaleDateString() : "Never"} sub="most recent" />
          </div>
        ) : null}

        {/* Database Clusters Table */}
        {loading ? (
          <TableSkeleton columns={7} rows={4} />
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : !clusters || clusters.length === 0 ? (
          <EmptyState
            title="No databases"
            description="No CNPG database clusters have been provisioned."
            icon={Database}
          />
        ) : (
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Name</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Namespace</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Instances</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Storage</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">WAL Archiving</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Last Backup</th>
                </tr>
              </thead>
              <tbody>
                {clusters.map((db) => (
                  <tr key={`${db.namespace}-${db.name}`} className="border-b border-border last:border-0 transition-colors hover:bg-surface-200">
                    <td className="px-4 py-3">
                      <span className="font-medium text-white">{db.name}</span>
                      {db.pgVersion && (
                        <span className="ml-2 font-mono text-xs text-neutral-500">PG {db.pgVersion}</span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-neutral-400">{db.namespace}</td>
                    <td className="px-4 py-3">
                      <StatusBadge status={dbStatusBadge(db.status)} label={db.status} />
                    </td>
                    <td className="px-4 py-3 text-neutral-300">
                      {db.readyInstances}/{db.totalInstances}
                    </td>
                    <td className="px-4 py-3 text-neutral-300">{db.storage || "—"}</td>
                    <td className="px-4 py-3">
                      {db.walArchiving ? (
                        <span className="text-emerald-400 text-xs">Active</span>
                      ) : (
                        <span className="text-red-400 text-xs">Disabled</span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-xs text-neutral-500">
                      {db.lastBackup ? new Date(db.lastBackup).toLocaleDateString() : "—"}
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
