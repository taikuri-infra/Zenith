"use client";

import { Shell } from "@/components/shell";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { Skeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import type { GrafanaDashboard } from "@/lib/api";
import { useApi } from "@/hooks/use-api";
import { BarChart3, ExternalLink } from "lucide-react";

const categoryColors: Record<string, string> = {
  infrastructure: "bg-blue-500/10 text-blue-400",
  application: "bg-emerald-500/10 text-emerald-400",
  database: "bg-amber-500/10 text-amber-400",
  networking: "bg-purple-500/10 text-purple-400",
  security: "bg-red-500/10 text-red-400",
  business: "bg-accent-500/10 text-accent-400",
};

export default function DashboardsPage() {
  const apiClient = getApi();
  const { data: dashboards, loading, error, refetch } = useApi<GrafanaDashboard[]>(
    () => apiClient.dashboards.list()
  );

  return (
    <Shell>
      <div className="space-y-6">
        <h1 className="text-lg font-semibold text-white">Dashboards</h1>

        {loading ? (
          <div className="grid grid-cols-3 gap-4">
            {Array.from({ length: 6 }).map((_, i) => (
              <div key={i} className="rounded-lg border border-border bg-surface-100 p-5">
                <Skeleton className="h-5 w-40 mb-2" />
                <Skeleton className="h-3 w-56 mb-4" />
                <Skeleton className="h-4 w-20" />
              </div>
            ))}
          </div>
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : !dashboards || dashboards.length === 0 ? (
          <EmptyState
            title="No dashboards"
            description="No Grafana dashboards are available."
            icon={BarChart3}
          />
        ) : (
          <div className="grid grid-cols-3 gap-4">
            {dashboards.map((dashboard) => (
              <a
                key={dashboard.uid}
                href={dashboard.url}
                target="_blank"
                rel="noopener noreferrer"
                className="group rounded-lg border border-border bg-surface-100 p-5 hover:bg-surface-200 transition-colors"
              >
                <div className="flex items-start justify-between">
                  <h3 className="text-sm font-medium text-white group-hover:text-accent-400 transition-colors">
                    {dashboard.title}
                  </h3>
                  <ExternalLink className="h-3.5 w-3.5 text-neutral-500 group-hover:text-white transition-colors" />
                </div>
                {dashboard.description && (
                  <p className="mt-1 text-xs text-neutral-500 line-clamp-2">
                    {dashboard.description}
                  </p>
                )}
                <div className="mt-3 flex items-center gap-2">
                  <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${(dashboard.category && categoryColors[dashboard.category]) || "bg-neutral-500/10 text-neutral-400"}`}>
                    {dashboard.category}
                  </span>
                  {dashboard.starred && (
                    <span className="text-amber-400 text-xs">Starred</span>
                  )}
                </div>
              </a>
            ))}
          </div>
        )}
      </div>
    </Shell>
  );
}
