"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { Skeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import type { ServiceHealthItem } from "@/lib/api";
import { useApi } from "@/hooks/use-api";
import Link from "next/link";
import { Server, RefreshCw } from "lucide-react";

function statusToBadge(status: string): "healthy" | "warning" | "error" | "idle" {
  switch (status) {
    case "healthy":
      return "healthy";
    case "degraded":
      return "warning";
    case "down":
      return "error";
    default:
      return "idle";
  }
}

export default function ServicesPage() {
  const apiClient = getApi();
  const { data: services, loading, error, refetch } = useApi<ServiceHealthItem[]>(
    () => apiClient.services.list()
  );

  const healthyCount = services?.filter((s) => s.status === "healthy").length ?? 0;
  const totalCount = services?.length ?? 0;

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Services</h1>
            {services && (
              <p className="mt-1 text-sm text-neutral-500">
                {healthyCount} of {totalCount} healthy
              </p>
            )}
          </div>
          <button
            onClick={refetch}
            className="flex items-center gap-1.5 rounded-lg border border-border bg-surface-100 px-3 py-1.5 text-sm text-neutral-400 hover:bg-surface-200 hover:text-white transition-colors"
          >
            <RefreshCw className="h-3.5 w-3.5" />
            Refresh
          </button>
        </div>

        {loading ? (
          <div className="grid grid-cols-4 gap-4">
            {Array.from({ length: 12 }).map((_, i) => (
              <div key={i} className="rounded-lg border border-border bg-surface-100 p-4">
                <Skeleton className="h-4 w-24 mb-2" />
                <Skeleton className="h-3 w-16 mb-3" />
                <div className="flex items-center gap-4">
                  <Skeleton className="h-3 w-20" />
                  <Skeleton className="h-3 w-14" />
                </div>
              </div>
            ))}
          </div>
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : !services || services.length === 0 ? (
          <EmptyState
            title="No services"
            description="No services are being monitored yet."
            icon={Server}
          />
        ) : (
          <div className="grid grid-cols-4 gap-4">
            {services.map((svc) => (
              <Link
                key={svc.name}
                href={`/services/${encodeURIComponent(svc.name)}`}
                className="rounded-lg border border-border bg-surface-100 p-4 hover:bg-surface-200 transition-colors"
              >
                <div className="flex items-center justify-between mb-1">
                  <h3 className="text-sm font-medium text-white truncate">{svc.name}</h3>
                  <StatusBadge status={statusToBadge(svc.status)} />
                </div>
                <p className="text-xs text-neutral-500 mb-3">{svc.namespace}</p>
                <div className="flex items-center gap-4 text-xs">
                  <span className="text-neutral-400">
                    Replicas{" "}
                    <span className={svc.readyReplicas < svc.totalReplicas ? "text-amber-400" : "text-emerald-400"}>
                      {svc.readyReplicas}/{svc.totalReplicas}
                    </span>
                  </span>
                  {svc.restarts > 0 && (
                    <span className="text-amber-400">
                      {svc.restarts} restarts
                    </span>
                  )}
                </div>
                {svc.version && (
                  <p className="mt-2 font-mono text-[10px] text-neutral-500">{svc.version}</p>
                )}
              </Link>
            ))}
          </div>
        )}
      </div>
    </Shell>
  );
}
