import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { ProgressBar } from "@/components/progress-bar";
import { mockClusters } from "@/lib/mock-data";
import { notFound } from "next/navigation";

export default async function ClusterDetailPage({
  params,
}: {
  params: Promise<{ name: string }>;
}) {
  const { name } = await params;
  const cluster = mockClusters.find((c) => c.name === name);

  if (!cluster) {
    notFound();
  }

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">{cluster.name}</h1>
            <p className="mt-1 text-sm text-neutral-500">
              {cluster.type === "dedicated" ? "Dedicated" : "Shared"} cluster
              {cluster.tenant && <span> &middot; Tenant: {cluster.tenant}</span>}
              {" "}&middot; {cluster.region}
            </p>
          </div>
          <StatusBadge status={cluster.status} />
        </div>

        {/* Overview cards */}
        <div className="grid grid-cols-4 gap-4">
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <p className="text-xs font-medium text-neutral-500">Kubernetes Version</p>
            <p className="mt-1 font-mono text-lg font-semibold text-white">{cluster.k8sVersion}</p>
            {cluster.upgradeAvailable && (
              <p className="mt-0.5 text-xs text-amber-400">Upgrade available: {cluster.upgradeAvailable}</p>
            )}
          </div>
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <p className="text-xs font-medium text-neutral-500">Nodes</p>
            <p className="mt-1 text-lg font-semibold text-white">{cluster.nodes}</p>
            <p className="mt-0.5 text-xs text-neutral-500">Worker nodes</p>
          </div>
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <p className="text-xs font-medium text-neutral-500">Pods</p>
            <p className="mt-1 text-lg font-semibold text-white">{cluster.pods.used} / {cluster.pods.total}</p>
            <p className="mt-0.5 text-xs text-neutral-500">{cluster.pods.total - cluster.pods.used} available</p>
          </div>
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <p className="text-xs font-medium text-neutral-500">PVCs</p>
            <p className="mt-1 text-lg font-semibold text-white">{cluster.pvcs.used} / {cluster.pvcs.total}</p>
            <p className="mt-0.5 text-xs text-neutral-500">{cluster.pvcs.total - cluster.pvcs.used} available</p>
          </div>
        </div>

        {/* Resource usage */}
        <section>
          <h2 className="mb-3 text-sm font-medium text-white">Resource Usage</h2>
          <div className="grid grid-cols-2 gap-4">
            <div className="rounded-lg border border-border bg-surface-100 p-4">
              <div className="mb-2 flex items-center justify-between">
                <p className="text-sm text-neutral-400">CPU</p>
                <p className="text-sm font-medium text-white">{cluster.cpuPercent}%</p>
              </div>
              <ProgressBar percent={cluster.cpuPercent} size="md" />
            </div>
            <div className="rounded-lg border border-border bg-surface-100 p-4">
              <div className="mb-2 flex items-center justify-between">
                <p className="text-sm text-neutral-400">RAM</p>
                <p className="text-sm font-medium text-white">{cluster.ramPercent}%</p>
              </div>
              <ProgressBar percent={cluster.ramPercent} size="md" />
            </div>
          </div>
        </section>

        {/* Actions */}
        <section>
          <h2 className="mb-3 text-sm font-medium text-white">Actions</h2>
          <div className="flex gap-3">
            <button className="rounded-lg border border-border bg-surface-100 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-surface-200">
              Upgrade Kubernetes
            </button>
            <button className="rounded-lg border border-border bg-surface-100 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-surface-200">
              Scale Nodes
            </button>
          </div>
        </section>
      </div>
    </Shell>
  );
}
