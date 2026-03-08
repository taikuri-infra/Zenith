"use client";

import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { useApi } from "@/hooks/use-api";
import { useProject } from "@/hooks/use-project";
import { getApi } from "@/lib/get-api";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { Activity, Cpu, MemoryStick, ArrowRight } from "lucide-react";
import Link from "next/link";

export default function MonitoringPage() {
  const { appsDeploy } = getApi();
  const projectId = useProject();

  const { data, loading } = useApi(
    () => appsDeploy.list(projectId || undefined),
    [projectId]
  );
  const apps = data?.items ?? [];

  if (loading) {
    return (
      <Shell>
        <PageWithTableSkeleton cols={3} rows={4} />
      </Shell>
    );
  }

  return (
    <Shell>
      <div className="space-y-8">
        <div>
          <h1 className="text-lg font-semibold text-white">Monitoring</h1>
          <p className="text-sm text-neutral-500">
            Select an app to view metrics, logs, and pod health
          </p>
        </div>

        {/* Stats Row */}
        <div className="grid grid-cols-4 gap-4">
          <StatCard label="Total Apps" value={apps.length} sub="deployed" />
          <StatCard
            label="Running"
            value={apps.filter((a) => a.status === "running").length}
            sub="healthy"
          />
          <StatCard
            label="Building"
            value={apps.filter((a) => a.status === "building").length}
            sub="in progress"
          />
          <StatCard
            label="Failed"
            value={apps.filter((a) => a.status === "failed").length}
            sub="need attention"
          />
        </div>

        {/* App Grid */}
        {apps.length === 0 ? (
          <div className="flex flex-col items-center justify-center rounded-lg border border-border bg-surface-100 py-16">
            <Activity className="mb-3 h-8 w-8 text-neutral-600" />
            <p className="text-sm text-neutral-400">No apps deployed yet</p>
            <p className="mt-1 text-xs text-neutral-600">
              Deploy an app to start monitoring
            </p>
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {apps.map((app) => (
              <AppMonitorCard key={app.id} app={app} />
            ))}
          </div>
        )}
      </div>
    </Shell>
  );
}

const statusColors: Record<string, { dot: string; text: string }> = {
  running: { dot: "bg-emerald-400", text: "text-emerald-400" },
  building: { dot: "bg-amber-400", text: "text-amber-400" },
  deploying: { dot: "bg-blue-400", text: "text-blue-400" },
  sleeping: { dot: "bg-indigo-400", text: "text-indigo-400" },
  failed: { dot: "bg-red-400", text: "text-red-400" },
  stopped: { dot: "bg-neutral-500", text: "text-neutral-500" },
  pending: { dot: "bg-neutral-400", text: "text-neutral-400" },
};

function AppMonitorCard({ app }: { app: { id: string; name: string; status: string } }) {
  const { monitoring } = getApi();
  const { data: overview } = useApi(
    () => monitoring.getOverview(app.id),
    [app.id]
  );

  const colors = statusColors[app.status] || statusColors.pending;

  return (
    <Link
      href={`/monitoring/${app.id}`}
      className="group rounded-lg border border-border bg-surface-100 p-4 transition-colors hover:border-border-hover hover:bg-surface-200"
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className={`h-2 w-2 rounded-full ${colors.dot}`} />
          <span className="text-sm font-medium text-white">{app.name}</span>
        </div>
        <ArrowRight className="h-4 w-4 text-neutral-600 transition-colors group-hover:text-neutral-400" />
      </div>

      <div className="mt-1 mb-3">
        <span className={`text-xs ${colors.text}`}>{app.status}</span>
      </div>

      {overview ? (
        <div className="grid grid-cols-3 gap-2">
          <div className="flex items-center gap-1.5">
            <Cpu className="h-3 w-3 text-neutral-500" />
            <span className="text-xs text-neutral-400">
              {overview.cpu_percent.toFixed(1)}%
            </span>
          </div>
          <div className="flex items-center gap-1.5">
            <MemoryStick className="h-3 w-3 text-neutral-500" />
            <span className="text-xs text-neutral-400">
              {overview.memory_mb.toFixed(0)}MB
            </span>
          </div>
          <div className="flex items-center gap-1.5">
            <Activity className="h-3 w-3 text-neutral-500" />
            <span className="text-xs text-neutral-400">
              {overview.pod_count} pod{overview.pod_count !== 1 ? "s" : ""}
            </span>
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-3 gap-2">
          {[0, 1, 2].map((i) => (
            <div key={i} className="h-4 animate-pulse rounded bg-surface-200" />
          ))}
        </div>
      )}
    </Link>
  );
}
