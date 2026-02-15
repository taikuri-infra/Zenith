"use client";

import { useState, useEffect } from "react";
import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { ProgressBar } from "@/components/progress-bar";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { TableSkeleton } from "@/components/loading-skeleton";
import { Modal } from "@/components/modal";
import { getApi } from "@/lib/get-api";
import type { Cluster } from "@/lib/api";
import { useApi } from "@/hooks/use-api";
import Link from "next/link";
import { Server } from "lucide-react";

export default function ClustersPage() {
  const apiClient = getApi();
  const { data: clusters, loading, error, refetch } = useApi<Cluster[]>(
    () => apiClient.clusters.list()
  );

  const [localClusters, setLocalClusters] = useState<Cluster[]>([]);
  const [showModal, setShowModal] = useState(false);
  const [formName, setFormName] = useState("");
  const [formRegion, setFormRegion] = useState("eu-central");
  const [formNodes, setFormNodes] = useState(3);
  const [formNodeType, setFormNodeType] = useState("CX22");

  useEffect(() => {
    if (clusters) {
      setLocalClusters(clusters);
    }
  }, [clusters]);

  const handleCreate = () => {
    const newCluster: Cluster = {
      name: formName || "new-cluster",
      k8sVersion: "v1.30.2",
      nodes: formNodes,
      region: formRegion,
      type: "dedicated",
      cpuPercent: 0,
      ramPercent: 0,
      pods: { used: 0, total: 110 },
      pvcs: { used: 0, total: 50 },
      status: "warning",
      upgradeAvailable: undefined,
    };
    setLocalClusters((prev) => [...prev, newCluster]);
    setShowModal(false);
    setFormName("");
    setFormRegion("eu-central");
    setFormNodes(3);
    setFormNodeType("CX22");
  };

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-lg font-semibold text-white">Clusters</h1>
          <button
            onClick={() => setShowModal(true)}
            className="rounded-lg bg-accent-600 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-accent-500"
          >
            + New Cluster
          </button>
        </div>

        {loading ? (
          <TableSkeleton columns={6} rows={3} />
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : localClusters.length === 0 ? (
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
                {localClusters.map((cluster) => (
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

      {showModal && (
        <Modal title="New Cluster" onClose={() => setShowModal(false)}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              handleCreate();
            }}
            className="space-y-4"
          >
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">
                Cluster Name
              </label>
              <input
                type="text"
                value={formName}
                onChange={(e) => setFormName(e.target.value)}
                placeholder="my-cluster"
                required
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">
                Region
              </label>
              <select
                value={formRegion}
                onChange={(e) => setFormRegion(e.target.value)}
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
              >
                <option value="eu-central">eu-central</option>
                <option value="us-east">us-east</option>
                <option value="us-west">us-west</option>
              </select>
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">
                Node Count
              </label>
              <input
                type="number"
                value={formNodes}
                onChange={(e) => setFormNodes(Number(e.target.value))}
                min={1}
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">
                Node Type
              </label>
              <select
                value={formNodeType}
                onChange={(e) => setFormNodeType(e.target.value)}
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
              >
                <option value="CX22">CX22</option>
                <option value="CX33">CX33</option>
                <option value="CX43">CX43</option>
              </select>
            </div>
            <div className="flex justify-end gap-3 pt-2">
              <button
                type="button"
                onClick={() => setShowModal(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors"
              >
                Create Cluster
              </button>
            </div>
          </form>
        </Modal>
      )}
    </Shell>
  );
}
