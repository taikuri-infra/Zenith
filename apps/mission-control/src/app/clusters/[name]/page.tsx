"use client";

import { use } from "react";
import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { ProgressBar } from "@/components/progress-bar";
import { ErrorState } from "@/components/error-state";
import { ClusterDetailSkeleton } from "@/components/loading-skeleton";
import { DemoButton } from "@/components/demo-button";
import { getApi, isDemoMode } from "@/lib/get-api";
import type { Cluster } from "@/lib/api";
import { useApi, useMutation } from "@/hooks/use-api";

export default function ClusterDetailPage({
  params,
}: {
  params: Promise<{ name: string }>;
}) {
  const { name } = use(params);
  const apiClient = getApi();
  const demo = isDemoMode();

  const {
    data: cluster,
    loading,
    error,
    refetch,
  } = useApi<Cluster>(() => apiClient.clusters.get(name), [name]);

  const upgradeMutation = useMutation((version: string) =>
    apiClient.clusters.upgrade(name, version)
  );

  const handleUpgrade = async () => {
    if (demo || !cluster?.upgradeAvailable) return;
    try {
      await upgradeMutation.execute(cluster.upgradeAvailable);
      refetch();
    } catch {
      // error is set in the mutation hook
    }
  };

  return (
    <Shell>
      {loading ? (
        <ClusterDetailSkeleton />
      ) : error ? (
        <ErrorState error={error} onRetry={refetch} title="Failed to load cluster" />
      ) : !cluster ? (
        <ErrorState
          error={new Error("Cluster not found")}
          title="Cluster not found"
        />
      ) : (
        <div className="space-y-6">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-lg font-semibold text-white">
                {cluster.name}
              </h1>
              <p className="mt-1 text-sm text-neutral-500">
                {cluster.type === "dedicated" ? "Dedicated" : "Shared"} cluster
                {cluster.tenant && (
                  <span> &middot; Tenant: {cluster.tenant}</span>
                )}
                {" "}&middot; {cluster.region}
              </p>
            </div>
            <StatusBadge status={cluster.status} />
          </div>

          {/* Overview cards */}
          <div className="grid grid-cols-4 gap-4">
            <div className="rounded-lg border border-border bg-surface-100 p-4">
              <p className="text-xs font-medium text-neutral-500">
                Kubernetes Version
              </p>
              <p className="mt-1 font-mono text-lg font-semibold text-white">
                {cluster.k8sVersion}
              </p>
              {cluster.upgradeAvailable && (
                <p className="mt-0.5 text-xs text-amber-400">
                  Upgrade available: {cluster.upgradeAvailable}
                </p>
              )}
            </div>
            <div className="rounded-lg border border-border bg-surface-100 p-4">
              <p className="text-xs font-medium text-neutral-500">Nodes</p>
              <p className="mt-1 text-lg font-semibold text-white">
                {cluster.nodes}
              </p>
              <p className="mt-0.5 text-xs text-neutral-500">Worker nodes</p>
            </div>
            <div className="rounded-lg border border-border bg-surface-100 p-4">
              <p className="text-xs font-medium text-neutral-500">Pods</p>
              <p className="mt-1 text-lg font-semibold text-white">
                {cluster.pods.used} / {cluster.pods.total}
              </p>
              <p className="mt-0.5 text-xs text-neutral-500">
                {cluster.pods.total - cluster.pods.used} available
              </p>
            </div>
            <div className="rounded-lg border border-border bg-surface-100 p-4">
              <p className="text-xs font-medium text-neutral-500">PVCs</p>
              <p className="mt-1 text-lg font-semibold text-white">
                {cluster.pvcs.used} / {cluster.pvcs.total}
              </p>
              <p className="mt-0.5 text-xs text-neutral-500">
                {cluster.pvcs.total - cluster.pvcs.used} available
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
                    {cluster.cpuPercent}%
                  </p>
                </div>
                <ProgressBar percent={cluster.cpuPercent} size="md" />
              </div>
              <div className="rounded-lg border border-border bg-surface-100 p-4">
                <div className="mb-2 flex items-center justify-between">
                  <p className="text-sm text-neutral-400">RAM</p>
                  <p className="text-sm font-medium text-white">
                    {cluster.ramPercent}%
                  </p>
                </div>
                <ProgressBar percent={cluster.ramPercent} size="md" />
              </div>
            </div>
          </section>

          {/* Actions */}
          <section>
            <h2 className="mb-3 text-sm font-medium text-white">Actions</h2>
            <div className="flex gap-3">
              <DemoButton
                onClick={handleUpgrade}
                disabled={!cluster.upgradeAvailable || upgradeMutation.loading}
                className="rounded-lg border border-border bg-surface-100 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-surface-200 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {upgradeMutation.loading
                  ? "Upgrading..."
                  : "Upgrade Kubernetes"}
              </DemoButton>
              <DemoButton className="rounded-lg border border-border bg-surface-100 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-surface-200">
                Scale Nodes
              </DemoButton>
            </div>
            {upgradeMutation.error && (
              <p className="mt-2 text-xs text-red-400">
                Upgrade failed: {upgradeMutation.error.message}
              </p>
            )}
          </section>
        </div>
      )}
    </Shell>
  );
}
