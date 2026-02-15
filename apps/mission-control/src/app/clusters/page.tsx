"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { ProgressBar } from "@/components/progress-bar";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { TableSkeleton } from "@/components/loading-skeleton";
import { api } from "@/lib/api";
import type { Cluster } from "@/lib/api";
import { useApi } from "@/hooks/use-api";
import Link from "next/link";
import { Server } from "lucide-react";

export default function ClustersPage() {
  const { data: clusters, loading, error, refetch } = useApi<Cluster[]>(
    () => api.clusters.list()
  );

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-lg font-semibold text-white">Clusters</h1>
          <button className="rounded-lg bg-accent-600 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-accent-500">
            + New Cluster
          </button>
        </div>

        {loading ? (
          <TableSkeleton columns={6} rows={3} />
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : !clusters || clusters.length === 0 ? (
          <EmptyState
            title="No clusters"
            description="Create your first cluster to get started."
            icon={Server}
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
                    K8s Version
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    Nodes
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    CPU
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    RAM
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    Status
                  </th>
                </tr>
              </thead>
              <tbody>
                {clusters.map((cluster) => (
                  <tr
                    key={cluster.name}
                    className="border-b border-border last:border-0 transition-colors hover:bg-surface-200"
                  >
                    <td className="px-4 py-3">
                      <Link
                        href={`/clusters/${cluster.name}`}
                        className="font-medium text-white hover:text-accent-400 transition-colors"
                      >
                        {cluster.name}
                      </Link>
                      <span className="ml-2 text-xs text-neutral-500">
                        {cluster.region}
                      </span>
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-neutral-400">
                      {cluster.k8sVersion}
                      {cluster.upgradeAvailable && (
                        <span className="ml-1.5 text-amber-400">&#9888;</span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-neutral-300">
                      {cluster.nodes}
                    </td>
                    <td className="w-36 px-4 py-3">
                      <ProgressBar
                        percent={cluster.cpuPercent}
                        label={`${cluster.cpuPercent}%`}
                      />
                    </td>
                    <td className="w-36 px-4 py-3">
                      <ProgressBar
                        percent={cluster.ramPercent}
                        label={`${cluster.ramPercent}%`}
                      />
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
      </div>
    </Shell>
  );
}
