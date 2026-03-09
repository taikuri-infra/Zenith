"use client";

import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { StatusBadge } from "@/components/status-badge";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { StatCardRowSkeleton, TableSkeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import { demoApi } from "@/lib/demo-api";
import type { S3Bucket, PvcVolume, StorageStats } from "@/lib/api";
import { useApiWithFallback } from "@/hooks/use-api";
import { HardDrive } from "lucide-react";

export default function StoragePage() {
  const apiClient = getApi();
  const stats = useApiWithFallback<StorageStats>(
    () => apiClient.storage.stats(),
    () => demoApi.storage.stats(),
    (data) => !data || data.s3Used === "0 B"
  );
  const buckets = useApiWithFallback<S3Bucket[]>(
    () => apiClient.storage.buckets(),
    () => demoApi.storage.buckets()
  );
  const volumes = useApiWithFallback<PvcVolume[]>(
    () => apiClient.storage.volumes(),
    () => demoApi.storage.volumes()
  );

  const anyDemo = stats.isDemo || buckets.isDemo || volumes.isDemo;

  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">Storage</h1>
          {anyDemo && (
            <p className="mt-1 text-xs text-amber-400/70">Showing sample data</p>
          )}
        </div>

        {/* Stats */}
        {stats.loading ? (
          <StatCardRowSkeleton />
        ) : stats.data ? (
          <div className="grid grid-cols-4 gap-4">
            <StatCard label="S3 Buckets" value={stats.data.totalBuckets} sub="object storage" />
            <StatCard label="S3 Used" value={stats.data.s3Used} sub="total usage" />
            <StatCard label="PVC Volumes" value={stats.data.totalVolumes} sub="persistent volumes" />
            <StatCard label="PVC Used" value={stats.data.pvcUsed} sub="total usage" />
          </div>
        ) : null}

        {/* S3 Buckets */}
        <section>
          <h2 className="mb-3 text-sm font-medium text-white">S3 Buckets</h2>
          {buckets.loading ? (
            <TableSkeleton columns={5} rows={3} />
          ) : buckets.error ? (
            <ErrorState error={buckets.error} onRetry={buckets.refetch} />
          ) : !buckets.data || buckets.data.length === 0 ? (
            <EmptyState title="No buckets" description="No S3 buckets have been created." icon={HardDrive} />
          ) : (
            <div className="overflow-hidden rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-surface-100">
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Bucket</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Region</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Size</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Objects</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Created</th>
                  </tr>
                </thead>
                <tbody>
                  {buckets.data.map((bucket) => (
                    <tr key={bucket.name} className="border-b border-border last:border-0 transition-colors hover:bg-surface-200">
                      <td className="px-4 py-3 font-medium text-white">{bucket.name}</td>
                      <td className="px-4 py-3 text-neutral-400">{bucket.region}</td>
                      <td className="px-4 py-3 text-neutral-300">{bucket.size}</td>
                      <td className="px-4 py-3 text-neutral-300">{bucket.objectCount.toLocaleString()}</td>
                      <td className="px-4 py-3 text-xs text-neutral-500">{bucket.createdAt}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>

        {/* PVC Volumes */}
        <section>
          <h2 className="mb-3 text-sm font-medium text-white">Persistent Volumes</h2>
          {volumes.loading ? (
            <TableSkeleton columns={6} rows={4} />
          ) : volumes.error ? (
            <ErrorState error={volumes.error} onRetry={volumes.refetch} />
          ) : !volumes.data || volumes.data.length === 0 ? (
            <EmptyState title="No volumes" description="No persistent volume claims found." icon={HardDrive} />
          ) : (
            <div className="overflow-hidden rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-surface-100">
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Name</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Namespace</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Capacity</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Storage Class</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Bound To</th>
                  </tr>
                </thead>
                <tbody>
                  {volumes.data.map((vol) => (
                    <tr key={`${vol.namespace}-${vol.name}`} className="border-b border-border last:border-0 transition-colors hover:bg-surface-200">
                      <td className="px-4 py-3 font-medium text-white">{vol.name}</td>
                      <td className="px-4 py-3 text-neutral-400">{vol.namespace}</td>
                      <td className="px-4 py-3">
                        <StatusBadge status={vol.status === "Bound" ? "healthy" : "warning"} label={vol.status} />
                      </td>
                      <td className="px-4 py-3 text-neutral-300">{vol.capacity}</td>
                      <td className="px-4 py-3 text-neutral-400 font-mono text-xs">{vol.storageClass}</td>
                      <td className="px-4 py-3 text-neutral-500 text-xs">{vol.boundTo || "—"}</td>
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
