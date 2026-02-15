"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import {
  StatCardRowSkeleton,
  TableSkeleton,
  SectionSkeleton,
} from "@/components/loading-skeleton";
import { api } from "@/lib/api";
import type { PlatformState, Cluster, Module } from "@/lib/api";
import { useApi } from "@/hooks/use-api";

export default function StatePage() {
  const state = useApi<PlatformState>(() => api.state.get());
  const clusters = useApi<Cluster[]>(() => api.clusters.list());
  const modules = useApi<Module[]>(() => api.modules.list());

  return (
    <Shell>
      <div className="space-y-6">
        <h1 className="text-lg font-semibold text-white">Platform State</h1>

        {/* Platform overview */}
        {state.loading ? (
          <StatCardRowSkeleton />
        ) : state.error ? (
          <ErrorState error={state.error} onRetry={state.refetch} />
        ) : state.data ? (
          <div className="grid grid-cols-4 gap-4">
            <div className="rounded-lg border border-border bg-surface-100 p-4">
              <p className="text-xs font-medium text-neutral-500">
                Platform Version
              </p>
              <p className="mt-1 font-mono text-lg font-semibold text-white">
                {state.data.platformVersion}
              </p>
              {state.data.updateAvailable && (
                <p className="mt-0.5 text-xs text-amber-400">
                  Update available: {state.data.updateAvailable}
                </p>
              )}
            </div>
            <div className="rounded-lg border border-border bg-surface-100 p-4">
              <p className="text-xs font-medium text-neutral-500">
                Installed Date
              </p>
              <p className="mt-1 text-lg font-semibold text-white">
                {state.data.installedDate}
              </p>
              <p className="mt-0.5 text-xs text-neutral-500">
                {state.data.installedDaysAgo} days ago
              </p>
            </div>
            <div className="rounded-lg border border-border bg-surface-100 p-4">
              <p className="text-xs font-medium text-neutral-500">
                Management K8s
              </p>
              <p className="mt-1 font-mono text-lg font-semibold text-white">
                {state.data.managementK8sVersion}
              </p>
              <p className="mt-0.5 text-xs text-neutral-500">
                {state.data.managementK8sUpToDate
                  ? "Up to date"
                  : "Update available"}
              </p>
            </div>
            <div className="rounded-lg border border-border bg-surface-100 p-4">
              <p className="text-xs font-medium text-neutral-500">Domain</p>
              <p className="mt-1 text-lg font-semibold text-white">
                {state.data.domain}
              </p>
              <p className="mt-0.5 text-xs text-neutral-500">
                {state.data.wildcardTls
                  ? "Wildcard TLS active"
                  : "TLS not configured"}
              </p>
            </div>
          </div>
        ) : null}

        {/* Clusters table */}
        <section>
          <h2 className="mb-3 text-sm font-medium text-white">Clusters</h2>
          {clusters.loading ? (
            <TableSkeleton columns={6} rows={3} />
          ) : clusters.error ? (
            <ErrorState error={clusters.error} onRetry={clusters.refetch} />
          ) : !clusters.data || clusters.data.length === 0 ? (
            <EmptyState
              title="No clusters"
              description="No clusters have been created yet."
            />
          ) : (
            <div className="overflow-hidden rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-surface-100">
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      Name
                    </th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      Type
                    </th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      K8s Version
                    </th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      Region
                    </th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      Nodes
                    </th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      Status
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {clusters.data.map((cluster) => (
                    <tr
                      key={cluster.name}
                      className="border-b border-border last:border-0 transition-colors hover:bg-surface-200"
                    >
                      <td className="px-4 py-3 font-medium text-white">
                        {cluster.name}
                      </td>
                      <td className="px-4 py-3 text-neutral-400 capitalize">
                        {cluster.type}
                      </td>
                      <td className="px-4 py-3 font-mono text-xs text-neutral-400">
                        {cluster.k8sVersion}
                      </td>
                      <td className="px-4 py-3 text-neutral-400">
                        {cluster.region}
                      </td>
                      <td className="px-4 py-3 text-neutral-300">
                        {cluster.nodes}
                      </td>
                      <td className="px-4 py-3">
                        <StatusBadge status={cluster.status} />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>

        {/* Modules per cluster - version matrix */}
        <section>
          <h2 className="mb-3 text-sm font-medium text-white">
            Module Versions
          </h2>
          {modules.loading || clusters.loading ? (
            <TableSkeleton columns={5} rows={6} />
          ) : modules.error ? (
            <ErrorState error={modules.error} onRetry={modules.refetch} />
          ) : !modules.data || modules.data.length === 0 ? (
            <EmptyState
              title="No modules"
              description="No modules have been installed yet."
            />
          ) : (
            <div className="overflow-hidden rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-surface-100">
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      Module
                    </th>
                    {(clusters.data || []).map((c) => (
                      <th
                        key={c.name}
                        className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500"
                      >
                        {c.name}
                      </th>
                    ))}
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      Latest
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {modules.data.map((mod) => (
                    <tr
                      key={mod.name}
                      className="border-b border-border last:border-0 transition-colors hover:bg-surface-200"
                    >
                      <td className="px-4 py-3 font-medium text-white">
                        {mod.name}
                      </td>
                      {(clusters.data || []).map((c) => (
                        <td
                          key={c.name}
                          className="px-4 py-3 font-mono text-xs text-neutral-400"
                        >
                          {mod.installed}
                        </td>
                      ))}
                      <td className="px-4 py-3 font-mono text-xs">
                        <span
                          className={
                            mod.status === "update_available"
                              ? "text-amber-400"
                              : "text-neutral-400"
                          }
                        >
                          {mod.latest}
                        </span>
                      </td>
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
