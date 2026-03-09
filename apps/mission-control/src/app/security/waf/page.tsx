"use client";

import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { StatusBadge } from "@/components/status-badge";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { StatCardRowSkeleton, TableSkeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import type { KyvernoPolicy, WafStats } from "@/lib/api";
import { useApi } from "@/hooks/use-api";
import { ShieldAlert } from "lucide-react";

function actionBadge(action: string) {
  switch (action) {
    case "enforce":
      return <span className="rounded-full bg-red-500/15 px-2 py-0.5 text-xs font-medium text-red-400">Enforce</span>;
    case "audit":
      return <span className="rounded-full bg-amber-500/15 px-2 py-0.5 text-xs font-medium text-amber-400">Audit</span>;
    default:
      return <span className="rounded-full bg-neutral-500/10 px-2 py-0.5 text-xs font-medium text-neutral-400">{action}</span>;
  }
}

export default function WafPage() {
  const apiClient = getApi();
  const stats = useApi<WafStats>(() => apiClient.security.wafStats());
  const { data: policies, loading, error, refetch } = useApi<KyvernoPolicy[]>(
    () => apiClient.security.policies()
  );

  return (
    <Shell>
      <div className="space-y-6">
        <h1 className="text-lg font-semibold text-white">WAF & Policies</h1>

        {/* Stats */}
        {stats.loading ? (
          <StatCardRowSkeleton />
        ) : stats.data ? (
          <div className="grid grid-cols-4 gap-4">
            <StatCard label="Total Policies" value={stats.data.totalPolicies} sub="Kyverno policies" />
            <StatCard label="Enforcing" value={stats.data.enforcing} sub="blocking violations" />
            <StatCard label="Auditing" value={stats.data.auditing} sub="logging only" />
            <StatCard label="Total Violations" value={stats.data.totalViolations} sub="all time" alert={stats.data.totalViolations > 0} />
          </div>
        ) : null}

        {/* Policies Table */}
        {loading ? (
          <TableSkeleton columns={6} rows={5} />
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : !policies || policies.length === 0 ? (
          <EmptyState
            title="No policies"
            description="No Kyverno policies are configured."
            icon={ShieldAlert}
          />
        ) : (
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Name</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Kind</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Action</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Violations</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Last Updated</th>
                </tr>
              </thead>
              <tbody>
                {policies.map((policy) => (
                  <tr key={policy.name} className="border-b border-border last:border-0 transition-colors hover:bg-surface-200">
                    <td className="px-4 py-3 font-medium text-white">{policy.name}</td>
                    <td className="px-4 py-3">
                      <span className="rounded bg-surface-300 px-1.5 py-0.5 text-xs text-neutral-300">
                        {policy.kind}
                      </span>
                    </td>
                    <td className="px-4 py-3">{actionBadge(policy.action)}</td>
                    <td className="px-4 py-3">
                      <StatusBadge status={policy.ready ? "healthy" : "error"} label={policy.ready ? "Ready" : "Error"} />
                    </td>
                    <td className="px-4 py-3">
                      <span className={`text-sm ${policy.violations > 0 ? "text-red-400 font-medium" : "text-neutral-400"}`}>
                        {policy.violations}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-xs text-neutral-500">{policy.updatedAt}</td>
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
