"use client";

import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { StatCardRowSkeleton, TableSkeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import type { Alert, AlertStats } from "@/lib/api";
import { useApi } from "@/hooks/use-api";
import { useState } from "react";
import { Bell, RefreshCw } from "lucide-react";

function severityBadge(severity: string) {
  switch (severity) {
    case "critical":
      return <span className="rounded-full bg-red-500/15 px-2 py-0.5 text-xs font-medium text-red-400">Critical</span>;
    case "warning":
      return <span className="rounded-full bg-amber-500/15 px-2 py-0.5 text-xs font-medium text-amber-400">Warning</span>;
    case "info":
      return <span className="rounded-full bg-blue-500/15 px-2 py-0.5 text-xs font-medium text-blue-400">Info</span>;
    default:
      return <span className="rounded-full bg-neutral-500/10 px-2 py-0.5 text-xs font-medium text-neutral-400">{severity}</span>;
  }
}

function stateBadge(state: string) {
  switch (state) {
    case "firing":
      return <span className="rounded-full bg-red-500/15 px-2 py-0.5 text-xs font-medium text-red-400">Firing</span>;
    case "pending":
      return <span className="rounded-full bg-amber-500/15 px-2 py-0.5 text-xs font-medium text-amber-400">Pending</span>;
    case "resolved":
      return <span className="rounded-full bg-emerald-500/15 px-2 py-0.5 text-xs font-medium text-emerald-400">Resolved</span>;
    default:
      return <span className="rounded-full bg-neutral-500/10 px-2 py-0.5 text-xs font-medium text-neutral-400">{state}</span>;
  }
}

const stateFilters = [
  { value: "", label: "All" },
  { value: "firing", label: "Firing" },
  { value: "pending", label: "Pending" },
  { value: "resolved", label: "Resolved" },
];

export default function AlertsPage() {
  const apiClient = getApi();
  const stats = useApi<AlertStats>(() => apiClient.alerts.stats());
  const { data: alerts, loading, error, refetch } = useApi<Alert[]>(
    () => apiClient.alerts.list()
  );
  const [stateFilter, setStateFilter] = useState("");

  const filtered = alerts?.filter((a) => !stateFilter || a.state === stateFilter);

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-lg font-semibold text-white">Alerts</h1>
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
            <StatCard label="Firing" value={stats.data.firing} sub="active alerts" alert={stats.data.firing > 0} />
            <StatCard label="Pending" value={stats.data.pending} sub="pending alerts" />
            <StatCard label="Resolved (24h)" value={stats.data.resolvedToday} sub="last 24 hours" />
            <StatCard label="Total Rules" value={stats.data.totalRules} sub="configured" />
          </div>
        ) : null}

        {/* Filter Tabs */}
        <div className="flex gap-2">
          {stateFilters.map((tab) => (
            <button
              key={tab.value}
              onClick={() => setStateFilter(tab.value)}
              className={`rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${
                stateFilter === tab.value
                  ? "bg-accent-600/15 text-accent-400"
                  : "text-neutral-400 hover:bg-surface-300 hover:text-white"
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>

        {/* Alerts Table */}
        {loading ? (
          <TableSkeleton columns={5} rows={5} />
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : !filtered || filtered.length === 0 ? (
          <EmptyState
            title="No alerts"
            description={stateFilter ? `No ${stateFilter} alerts.` : "All systems nominal. No active alerts."}
            icon={Bell}
          />
        ) : (
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Name</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">State</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Severity</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Summary</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Active Since</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((alert, i) => (
                  <tr key={`${alert.name}-${i}`} className="border-b border-border last:border-0 transition-colors hover:bg-surface-200">
                    <td className="px-4 py-3 font-medium text-white">{alert.name}</td>
                    <td className="px-4 py-3">{stateBadge(alert.state)}</td>
                    <td className="px-4 py-3">{severityBadge(alert.severity)}</td>
                    <td className="px-4 py-3 text-neutral-300 text-xs max-w-xs truncate">{alert.summary}</td>
                    <td className="px-4 py-3 text-xs text-neutral-500">{alert.activeSince}</td>
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
