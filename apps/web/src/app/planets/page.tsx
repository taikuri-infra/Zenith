"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { TableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { getApi } from "@/lib/get-api";
import { useApi } from "@/hooks/use-api";
import type { HetznerNode } from "@/lib/api";

function formatRam(mb: number): string {
  if (mb >= 1024) return `${(mb / 1024).toFixed(0)} GB`;
  return `${mb} MB`;
}

function nodeStatus(status: string): "running" | "stopped" | "error" | "provisioning" {
  switch (status) {
    case "running": return "running";
    case "off":
    case "stopped": return "stopped";
    case "initializing":
    case "starting": return "provisioning";
    default: return "error";
  }
}

export default function PlanetsPage() {
  const apiClient = getApi();
  const { data, loading, error } = useApi<{ items: HetznerNode[]; total: number }>(
    () => apiClient.autoscaler.listNodes()
  );

  const nodes = data?.items ?? [];
  const runningCount = nodes.filter((n) => n.status === "running").length;

  if (loading) return <Shell><TableSkeleton /></Shell>;
  if (error) return <Shell><ErrorState message={error} /></Shell>;

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Planets</h1>
            <p className="text-sm text-neutral-500">
              {nodes.length} compute nodes, {runningCount} running
            </p>
          </div>
        </div>

        {nodes.length === 0 ? (
          <EmptyState
            title="No nodes yet"
            description="Planets are provisioned automatically as your platform scales."
          />
        ) : (
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Name</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Type</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">IP</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Status</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">CPU</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Memory</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Cost</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Created</th>
                </tr>
              </thead>
              <tbody>
                {nodes.map((node) => (
                  <tr key={node.server_id} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                    <td className="whitespace-nowrap px-4 py-3 font-medium text-white">{node.name}</td>
                    <td className="whitespace-nowrap px-4 py-3">
                      <span className="inline-flex rounded bg-surface-300 px-1.5 py-0.5 font-mono text-xs text-neutral-300">
                        {node.server_type}
                      </span>
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-400">{node.ip}</td>
                    <td className="whitespace-nowrap px-4 py-3">
                      <StatusBadge status={nodeStatus(node.status)} />
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-300">{node.cpu_cores} vCPU</td>
                    <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-300">{formatRam(node.ram_mb)}</td>
                    <td className="whitespace-nowrap px-4 py-3 text-xs text-neutral-400">€{node.monthly_cost.toFixed(2)}/mo</td>
                    <td className="whitespace-nowrap px-4 py-3 text-xs text-neutral-400">
                      {new Date(node.created_at).toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" })}
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
