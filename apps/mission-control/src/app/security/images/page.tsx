"use client";

import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { StatusBadge } from "@/components/status-badge";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { StatCardRowSkeleton, TableSkeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import type { ImageScanResult, ImageScanStats } from "@/lib/api";
import { useApi } from "@/hooks/use-api";
import { Bug, RefreshCw } from "lucide-react";

function scanStatusBadge(status: string): "healthy" | "warning" | "error" | "idle" {
  switch (status) {
    case "clean":
      return "healthy";
    case "low":
    case "medium":
      return "warning";
    case "high":
    case "critical":
      return "error";
    default:
      return "idle";
  }
}

function vulnCount(count: number, color: string) {
  return (
    <span className={`font-mono text-xs ${count > 0 ? color : "text-neutral-600"}`}>
      {count}
    </span>
  );
}

export default function ImageScanningPage() {
  const apiClient = getApi();
  const stats = useApi<ImageScanStats>(() => apiClient.security.imageStats());
  const { data: images, loading, error, refetch } = useApi<ImageScanResult[]>(
    () => apiClient.security.images()
  );

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-lg font-semibold text-white">Image Scanning</h1>
          <button
            onClick={refetch}
            className="flex items-center gap-1.5 rounded-lg border border-border bg-surface-100 px-3 py-1.5 text-sm text-neutral-400 hover:bg-surface-200 hover:text-white transition-colors"
          >
            <RefreshCw className="h-3.5 w-3.5" />
            Rescan
          </button>
        </div>

        {/* Stats */}
        {stats.loading ? (
          <StatCardRowSkeleton />
        ) : stats.data ? (
          <div className="grid grid-cols-4 gap-4">
            <StatCard label="Images Scanned" value={stats.data.totalImages} sub="total images" />
            <StatCard label="Clean" value={stats.data.cleanImages} sub="no vulnerabilities" />
            <StatCard label="Critical" value={stats.data.criticalCount} sub="critical CVEs" alert={stats.data.criticalCount > 0} />
            <StatCard label="High" value={stats.data.highCount} sub="high CVEs" alert={stats.data.highCount > 0} />
          </div>
        ) : null}

        {/* Images Table */}
        {loading ? (
          <TableSkeleton columns={7} rows={5} />
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : !images || images.length === 0 ? (
          <EmptyState
            title="No images scanned"
            description="No container images have been scanned yet."
            icon={Bug}
          />
        ) : (
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Repository</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Tag</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Critical</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">High</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Medium</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Low</th>
                </tr>
              </thead>
              <tbody>
                {images.map((img) => (
                  <tr key={`${img.repository}:${img.tag}`} className="border-b border-border last:border-0 transition-colors hover:bg-surface-200">
                    <td className="px-4 py-3 font-mono text-xs text-white">{img.repository}</td>
                    <td className="px-4 py-3">
                      <span className="rounded bg-surface-300 px-1.5 py-0.5 text-xs text-neutral-300">
                        {img.tag}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <StatusBadge status={scanStatusBadge(img.scanStatus)} label={img.scanStatus} />
                    </td>
                    <td className="px-4 py-3">{vulnCount(img.critical, "text-red-400")}</td>
                    <td className="px-4 py-3">{vulnCount(img.high, "text-orange-400")}</td>
                    <td className="px-4 py-3">{vulnCount(img.medium, "text-amber-400")}</td>
                    <td className="px-4 py-3">{vulnCount(img.low, "text-neutral-400")}</td>
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
