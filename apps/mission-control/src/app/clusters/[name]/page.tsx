"use client";

import { use, useState, useEffect } from "react";
import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { ProgressBar } from "@/components/progress-bar";
import { ErrorState } from "@/components/error-state";
import { ClusterDetailSkeleton } from "@/components/loading-skeleton";
import { Modal } from "@/components/modal";
import { getApi } from "@/lib/get-api";
import type { Cluster } from "@/lib/api";
import { useApi } from "@/hooks/use-api";

export default function ClusterDetailPage({
  params,
}: {
  params: Promise<{ name: string }>;
}) {
  const { name } = use(params);
  const apiClient = getApi();

  const {
    data: cluster,
    loading,
    error,
    refetch,
  } = useApi<Cluster>(() => apiClient.clusters.get(name), [name]);

  const [localCluster, setLocalCluster] = useState<Cluster | null>(null);
  const [showScaleModal, setShowScaleModal] = useState(false);
  const [showUpgradeModal, setShowUpgradeModal] = useState(false);
  const [targetNodes, setTargetNodes] = useState(3);
  const [upgradeSuccess, setUpgradeSuccess] = useState(false);

  useEffect(() => {
    if (cluster) {
      setLocalCluster(cluster);
      setTargetNodes(cluster.nodes);
    }
  }, [cluster]);

  const handleScale = () => {
    if (localCluster) {
      setLocalCluster({ ...localCluster, nodes: targetNodes });
    }
    setShowScaleModal(false);
  };

  const handleUpgrade = () => {
    if (localCluster && localCluster.upgradeAvailable) {
      setLocalCluster({
        ...localCluster,
        k8sVersion: localCluster.upgradeAvailable,
        upgradeAvailable: undefined,
      });
      setUpgradeSuccess(true);
      setTimeout(() => setUpgradeSuccess(false), 3000);
    }
    setShowUpgradeModal(false);
  };

  return (
    <Shell>
      {loading ? (
        <ClusterDetailSkeleton />
      ) : error ? (
        <ErrorState error={error} onRetry={refetch} title="Failed to load cluster" />
      ) : !localCluster ? (
        <ErrorState
          error={new Error("Cluster not found")}
          title="Cluster not found"
        />
      ) : (
        <div className="space-y-6">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-lg font-semibold text-white">
                {localCluster.name}
              </h1>
              <p className="mt-1 text-sm text-neutral-500">
                {localCluster.type === "dedicated" ? "Dedicated" : "Shared"} cluster
                {localCluster.tenant && (
                  <span> &middot; Tenant: {localCluster.tenant}</span>
                )}
                {" "}&middot; {localCluster.region}
              </p>
            </div>
            <StatusBadge status={localCluster.status} />
          </div>

          {upgradeSuccess && (
            <div className="rounded-lg border border-emerald-500/20 bg-emerald-500/5 px-4 py-2 text-xs text-emerald-400">
              Kubernetes upgrade completed successfully!
            </div>
          )}

          {/* Overview cards */}
          <div className="grid grid-cols-4 gap-4">
            <div className="rounded-lg border border-border bg-surface-100 p-4">
              <p className="text-xs font-medium text-neutral-500">
                Kubernetes Version
              </p>
              <p className="mt-1 font-mono text-lg font-semibold text-white">
                {localCluster.k8sVersion}
              </p>
              {localCluster.upgradeAvailable && (
                <p className="mt-0.5 text-xs text-amber-400">
                  Upgrade available: {localCluster.upgradeAvailable}
                </p>
              )}
            </div>
            <div className="rounded-lg border border-border bg-surface-100 p-4">
              <p className="text-xs font-medium text-neutral-500">Nodes</p>
              <p className="mt-1 text-lg font-semibold text-white">
                {localCluster.nodes}
              </p>
              <p className="mt-0.5 text-xs text-neutral-500">Worker nodes</p>
            </div>
            <div className="rounded-lg border border-border bg-surface-100 p-4">
              <p className="text-xs font-medium text-neutral-500">Pods</p>
              <p className="mt-1 text-lg font-semibold text-white">
                {localCluster.pods.used} / {localCluster.pods.total}
              </p>
              <p className="mt-0.5 text-xs text-neutral-500">
                {localCluster.pods.total - localCluster.pods.used} available
              </p>
            </div>
            <div className="rounded-lg border border-border bg-surface-100 p-4">
              <p className="text-xs font-medium text-neutral-500">PVCs</p>
              <p className="mt-1 text-lg font-semibold text-white">
                {localCluster.pvcs.used} / {localCluster.pvcs.total}
              </p>
              <p className="mt-0.5 text-xs text-neutral-500">
                {localCluster.pvcs.total - localCluster.pvcs.used} available
              </p>
            </div>
          </div>

          {/* Resource usage */}
          <section>
            <h2 className="mb-3 text-sm font-medium text-white">
              Resource Usage
            </h2>
            <div className="grid grid-cols-2 gap-4">
              <div className="rounded-lg border border-border bg-surface-100 p-4">
                <div className="mb-2 flex items-center justify-between">
                  <p className="text-sm text-neutral-400">CPU</p>
                  <p className="text-sm font-medium text-white">
                    {localCluster.cpuPercent}%
                  </p>
                </div>
                <ProgressBar percent={localCluster.cpuPercent} size="md" />
              </div>
              <div className="rounded-lg border border-border bg-surface-100 p-4">
                <div className="mb-2 flex items-center justify-between">
                  <p className="text-sm text-neutral-400">RAM</p>
                  <p className="text-sm font-medium text-white">
                    {localCluster.ramPercent}%
                  </p>
                </div>
                <ProgressBar percent={localCluster.ramPercent} size="md" />
              </div>
            </div>
          </section>

          {/* Actions */}
          <section>
            <h2 className="mb-3 text-sm font-medium text-white">Actions</h2>
            <div className="flex gap-3">
              <button
                onClick={() => setShowUpgradeModal(true)}
                disabled={!localCluster.upgradeAvailable}
                className="rounded-lg border border-border bg-surface-100 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-surface-200 disabled:cursor-not-allowed disabled:opacity-50"
              >
                Upgrade Kubernetes
              </button>
              <button
                onClick={() => {
                  setTargetNodes(localCluster.nodes);
                  setShowScaleModal(true);
                }}
                className="rounded-lg border border-border bg-surface-100 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-surface-200"
              >
                Scale Nodes
              </button>
            </div>
          </section>
        </div>
      )}

      {showScaleModal && localCluster && (
        <Modal title="Scale Nodes" onClose={() => setShowScaleModal(false)}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              handleScale();
            }}
            className="space-y-4"
          >
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">
                Target Nodes
              </label>
              <input
                type="number"
                value={targetNodes}
                onChange={(e) => setTargetNodes(Number(e.target.value))}
                min={1}
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
              <p className="mt-1 text-xs text-neutral-500">
                Current: {localCluster.nodes} nodes
              </p>
            </div>
            <div className="flex justify-end gap-3 pt-2">
              <button
                type="button"
                onClick={() => setShowScaleModal(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors"
              >
                Scale
              </button>
            </div>
          </form>
        </Modal>
      )}

      {showUpgradeModal && localCluster && (
        <Modal title="Upgrade Kubernetes" onClose={() => setShowUpgradeModal(false)}>
          <div className="space-y-4">
            <p className="text-sm text-neutral-300">
              Upgrade to <span className="font-mono font-medium text-white">{localCluster.upgradeAvailable}</span>?
            </p>
            <p className="text-xs text-neutral-500">
              This will perform a rolling upgrade of all nodes from{" "}
              <span className="font-mono">{localCluster.k8sVersion}</span> to{" "}
              <span className="font-mono">{localCluster.upgradeAvailable}</span>.
            </p>
            <div className="flex justify-end gap-3 pt-2">
              <button
                type="button"
                onClick={() => setShowUpgradeModal(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleUpgrade}
                className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors"
              >
                Proceed
              </button>
            </div>
          </div>
        </Modal>
      )}
    </Shell>
  );
}
