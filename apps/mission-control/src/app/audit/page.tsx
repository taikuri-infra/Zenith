"use client";

import { useState } from "react";
import { Shell } from "@/components/shell";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { TableSkeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import type { AuditEntry } from "@/lib/api";
import { useApi } from "@/hooks/use-api";
import { ScrollText } from "lucide-react";

export default function AuditPage() {
  const apiClient = getApi();

  const [actor, setActor] = useState<string>("");
  const [cluster, setCluster] = useState<string>("");
  const [period, setPeriod] = useState<string>("today");

  const {
    data: auditLog,
    loading,
    error,
    refetch,
  } = useApi<AuditEntry[]>(
    () =>
      apiClient.audit.list({
        actor: actor || undefined,
        cluster: cluster || undefined,
        period: period || undefined,
      }),
    [actor, cluster, period]
  );

  return (
    <Shell>
      <div className="space-y-6">
        <h1 className="text-lg font-semibold text-white">Audit Log</h1>

        {/* Filter dropdowns */}
        <div className="flex gap-3">
          <select
            value={actor}
            onChange={(e) => setActor(e.target.value)}
            className="rounded-lg border border-border bg-surface-100 px-3 py-1.5 text-sm text-neutral-300 outline-none"
          >
            <option value="">All Actors</option>
            <option value="admin">admin</option>
            <option value="system">system</option>
            <option value="CAPI">CAPI</option>
          </select>
          <select
            value={cluster}
            onChange={(e) => setCluster(e.target.value)}
            className="rounded-lg border border-border bg-surface-100 px-3 py-1.5 text-sm text-neutral-300 outline-none"
          >
            <option value="">All Clusters</option>
            <option value="production-eu">production-eu</option>
            <option value="staging-us">staging-us</option>
            <option value="dev-local">dev-local</option>
          </select>
          <select
            value={period}
            onChange={(e) => setPeriod(e.target.value)}
            className="rounded-lg border border-border bg-surface-100 px-3 py-1.5 text-sm text-neutral-300 outline-none"
          >
            <option value="today">Today</option>
            <option value="7d">Last 7 days</option>
            <option value="30d">Last 30 days</option>
            <option value="all">All time</option>
          </select>
        </div>

        {/* Audit log table */}
        {loading ? (
          <TableSkeleton columns={4} rows={4} />
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : !auditLog || auditLog.length === 0 ? (
          <EmptyState
            title="No audit entries"
            description="No audit events match the current filters."
            icon={ScrollText}
          />
        ) : (
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    Time
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    Actor
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    Action
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    Cluster
                  </th>
                </tr>
              </thead>
              <tbody>
                {auditLog.map((entry, i) => (
                  <tr
                    key={i}
                    className="border-b border-border last:border-0 transition-colors hover:bg-surface-200"
                  >
                    <td className="px-4 py-3 font-mono text-xs text-neutral-500">
                      {entry.time}
                    </td>
                    <td className="px-4 py-3 font-medium text-white">
                      {entry.actor}
                    </td>
                    <td className="px-4 py-3 text-neutral-300">
                      {entry.action}
                    </td>
                    <td className="px-4 py-3 text-neutral-400">
                      {entry.cluster || "--"}
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
