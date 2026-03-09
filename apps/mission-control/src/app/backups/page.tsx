"use client";

import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { StatusBadge } from "@/components/status-badge";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { StatCardRowSkeleton, TableSkeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import type { VeleroSchedule, CnpgBackup, BackupStats } from "@/lib/api";
import { useApi } from "@/hooks/use-api";
import { Archive, RefreshCw } from "lucide-react";

function backupStatusBadge(status: string): "healthy" | "warning" | "error" | "idle" {
  switch (status) {
    case "completed":
    case "success":
      return "healthy";
    case "in_progress":
    case "running":
      return "warning";
    case "failed":
      return "error";
    default:
      return "idle";
  }
}

export default function BackupsPage() {
  const apiClient = getApi();
  const stats = useApi<BackupStats>(() => apiClient.backups.stats());
  const velero = useApi<VeleroSchedule[]>(() => apiClient.backups.veleroSchedules());
  const cnpg = useApi<CnpgBackup[]>(() => apiClient.backups.cnpgBackups());

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-lg font-semibold text-white">Backups</h1>
          <button
            onClick={() => { velero.refetch(); cnpg.refetch(); stats.refetch(); }}
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
            <StatCard label="Velero Schedules" value={stats.data.veleroSchedules} sub="active schedules" />
            <StatCard label="CNPG Clusters" value={stats.data.cnpgClusters} sub="with backups" />
            <StatCard label="Last Backup" value={stats.data.lastBackup || "Never"} sub="most recent" />
            <StatCard label="Total Size" value={stats.data.totalSize} sub="all backups" />
          </div>
        ) : null}

        {/* Velero Schedules */}
        <section>
          <h2 className="mb-3 text-sm font-medium text-white">Velero Schedules</h2>
          {velero.loading ? (
            <TableSkeleton columns={6} rows={3} />
          ) : velero.error ? (
            <ErrorState error={velero.error} onRetry={velero.refetch} />
          ) : !velero.data || velero.data.length === 0 ? (
            <EmptyState
              title="No Velero schedules"
              description="No Velero backup schedules are configured."
              icon={Archive}
            />
          ) : (
            <div className="overflow-hidden rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-surface-100">
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Name</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Schedule</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Last Backup</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Retention</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Storage Location</th>
                  </tr>
                </thead>
                <tbody>
                  {velero.data.map((schedule) => (
                    <tr key={schedule.name} className="border-b border-border last:border-0 transition-colors hover:bg-surface-200">
                      <td className="px-4 py-3 font-medium text-white">{schedule.name}</td>
                      <td className="px-4 py-3 font-mono text-xs text-neutral-400">{schedule.schedule}</td>
                      <td className="px-4 py-3 text-xs text-neutral-300">{schedule.lastBackup || "—"}</td>
                      <td className="px-4 py-3">
                        <StatusBadge status={backupStatusBadge(schedule.lastStatus)} label={schedule.lastStatus} />
                      </td>
                      <td className="px-4 py-3 text-neutral-400 text-xs">{schedule.retention}</td>
                      <td className="px-4 py-3 text-neutral-500 text-xs">{schedule.storageLocation}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>

        {/* CNPG Backups */}
        <section>
          <h2 className="mb-3 text-sm font-medium text-white">CNPG Database Backups</h2>
          {cnpg.loading ? (
            <TableSkeleton columns={6} rows={3} />
          ) : cnpg.error ? (
            <ErrorState error={cnpg.error} onRetry={cnpg.refetch} />
          ) : !cnpg.data || cnpg.data.length === 0 ? (
            <EmptyState
              title="No CNPG backups"
              description="No CloudNativePG backup configurations found."
              icon={Archive}
            />
          ) : (
            <div className="overflow-hidden rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-surface-100">
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Cluster</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Namespace</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Schedule</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Last Backup</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">S3 Destination</th>
                  </tr>
                </thead>
                <tbody>
                  {cnpg.data.map((backup) => (
                    <tr key={`${backup.cluster}-${backup.namespace}`} className="border-b border-border last:border-0 transition-colors hover:bg-surface-200">
                      <td className="px-4 py-3 font-medium text-white">{backup.cluster}</td>
                      <td className="px-4 py-3 text-neutral-400">{backup.namespace}</td>
                      <td className="px-4 py-3 font-mono text-xs text-neutral-400">{backup.schedule}</td>
                      <td className="px-4 py-3 text-xs text-neutral-300">{backup.lastBackup || "—"}</td>
                      <td className="px-4 py-3">
                        <StatusBadge status={backupStatusBadge(backup.status)} label={backup.status} />
                      </td>
                      <td className="px-4 py-3 font-mono text-xs text-neutral-500">{backup.s3Destination}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </div>
    </Shell>
  );
}
