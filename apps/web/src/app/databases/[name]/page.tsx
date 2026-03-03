"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { StatCard } from "@/components/stat-card";
import { DatabaseDetailSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { useApi } from "@/hooks/use-api";
import { useProject } from "@/hooks/use-project";
import { type Database } from "@/lib/api";
import { getApi } from "@/lib/get-api";
import { useParams } from "next/navigation";

const engineColors: Record<string, string> = {
  postgresql: "bg-blue-500/20 text-blue-400",
  mysql: "bg-orange-500/20 text-orange-400",
  mongodb: "bg-green-500/20 text-green-400",
  redis: "bg-red-500/20 text-red-400",
};

export default function DatabaseDetailPage() {
  const { name } = useParams<{ name: string }>();
  const projectId = useProject();
  const { databases } = getApi();

  const {
    data: db,
    loading,
    error,
    refetch,
  } = useApi(
    () => projectId ? databases.get(projectId, name) : Promise.resolve(null),
    [projectId, name]
  );

  if (loading) {
    return (
      <Shell>
        <DatabaseDetailSkeleton />
      </Shell>
    );
  }

  if (error) {
    return (
      <Shell>
        <ErrorState message={error} onRetry={refetch} />
      </Shell>
    );
  }

  if (!db) {
    return (
      <Shell>
        <ErrorState message={`Database "${name}" not found`} />
      </Shell>
    );
  }

  return (
    <Shell>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div
              className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-sm font-bold ${
                engineColors[db.engine] ?? "bg-neutral-500/20 text-neutral-400"
              }`}
            >
              {db.engine[0].toUpperCase()}
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-lg font-semibold text-white">{db.name}</h1>
                <span
                  className={`rounded-md px-2 py-0.5 text-xs font-medium capitalize ${
                    engineColors[db.engine] ??
                    "bg-neutral-500/20 text-neutral-400"
                  }`}
                >
                  {db.engine}
                </span>
                <StatusBadge status={db.status as "running" | "creating" | "stopped"} />
              </div>
              <p className="mt-0.5 text-sm text-neutral-500">
                Version {db.version}
              </p>
            </div>
          </div>
        </div>

        {/* Connection string */}
        <div className="rounded-lg border border-border bg-surface-100 p-4">
          <div className="mb-2 flex items-center justify-between">
            <h3 className="text-sm font-medium text-white">
              Connection String
            </h3>
            <button className="rounded-md border border-border bg-surface-200 px-2.5 py-1 text-xs text-neutral-400 hover:bg-surface-300 hover:text-white transition-colors">
              Copy
            </button>
          </div>
          <div className="rounded-lg bg-surface-200 p-3">
            <code className="font-mono text-xs text-neutral-300 break-all">
              {db.connection_string}
            </code>
          </div>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-4 gap-4">
          <StatCard
            label="Engine"
            value={db.engine}
            sub={`Version ${db.version}`}
          />
          <StatCard label="Storage" value={db.storage} sub="allocated" />
          <StatCard label="Port" value={db.port} sub="listening" />
          <StatCard label="Created" value={db.created_at} />
        </div>
      </div>
    </Shell>
  );
}
