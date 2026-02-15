"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { useApi } from "@/hooks/use-api";
import { useProject } from "@/hooks/use-project";
import { databases, type Database } from "@/lib/api";
import Link from "next/link";

const engineBadge: Record<string, { label: string; className: string }> = {
  postgresql: { label: "P", className: "bg-blue-500/20 text-blue-400" },
  mysql: { label: "M", className: "bg-orange-500/20 text-orange-400" },
  mongodb: { label: "M", className: "bg-green-500/20 text-green-400" },
  redis: { label: "R", className: "bg-red-500/20 text-red-400" },
};

export default function DatabasesPage() {
  const projectId = useProject();

  const {
    data: dbsData,
    loading,
    error,
    refetch,
  } = useApi(() => databases.list(projectId), [projectId]);

  if (loading) {
    return (
      <Shell>
        <PageWithTableSkeleton cols={10} rows={3} />
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

  const dbList: Database[] = dbsData?.items ?? [];
  const runningCount = dbList.filter((d) => d.status === "running").length;
  const pgCount = dbList.filter((d) => d.engine === "postgresql").length;
  const redisCount = dbList.filter((d) => d.engine === "redis").length;

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Databases</h1>
            <p className="text-sm text-neutral-500">
              {dbList.length} instances, {runningCount} running
              {pgCount > 0 ? `, ${pgCount} PostgreSQL` : ""}
              {redisCount > 0 ? `, ${redisCount} Redis` : ""}
            </p>
          </div>
          <button className="rounded-lg bg-accent-500 hover:bg-accent-600 text-white px-3 py-1.5 text-sm transition-colors">
            + Create Instance
          </button>
        </div>

        {/* Filter bar */}
        <div className="flex items-center gap-3">
          <div className="relative flex-1">
            <svg
              className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-500"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M21 21l-4.35-4.35M11 19a8 8 0 100-16 8 8 0 000 16z"
              />
            </svg>
            <input
              type="text"
              placeholder="Filter instances..."
              className="w-full rounded-lg border border-border bg-surface-100 py-1.5 pl-9 pr-3 text-sm text-white placeholder:text-neutral-500 focus:border-accent-500 focus:outline-none"
            />
          </div>
          <select className="rounded-lg border border-border bg-surface-100 px-3 py-1.5 text-sm text-neutral-400 focus:border-accent-500 focus:outline-none">
            <option value="">All engines</option>
            <option value="postgresql">PostgreSQL</option>
            <option value="redis">Redis</option>
            <option value="mysql">MySQL</option>
          </select>
        </div>

        {/* Table or Empty State */}
        {dbList.length === 0 ? (
          <EmptyState
            title="No databases yet"
            description="Create your first database instance to get started."
            actionLabel="+ Create Instance"
          />
        ) : (
          <div className="overflow-hidden rounded-lg border border-border">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b border-border bg-surface-100">
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                      DB Identifier
                    </th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                      Engine
                    </th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                      Version
                    </th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                      Status
                    </th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                      Storage
                    </th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                      Port
                    </th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                      Created
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {dbList.map((db) => {
                    const badge = engineBadge[db.engine] ?? {
                      label: "?",
                      className: "bg-neutral-500/20 text-neutral-400",
                    };

                    return (
                      <tr
                        key={db.name}
                        className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors"
                      >
                        <td className="whitespace-nowrap px-4 py-3">
                          <Link
                            href={`/databases/${db.name}`}
                            className="font-medium text-white hover:text-accent-400 transition-colors"
                          >
                            {db.name}
                          </Link>
                        </td>
                        <td className="whitespace-nowrap px-4 py-3">
                          <span
                            className={`inline-flex h-5 w-5 items-center justify-center rounded text-[10px] font-bold ${badge.className}`}
                          >
                            {badge.label}
                          </span>
                          <span className="ml-2 text-xs capitalize text-neutral-300">
                            {db.engine}
                          </span>
                        </td>
                        <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-300">
                          {db.version}
                        </td>
                        <td className="whitespace-nowrap px-4 py-3">
                          <StatusBadge status={db.status as "running" | "creating" | "stopped"} />
                        </td>
                        <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-400">
                          {db.storage}
                        </td>
                        <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-300">
                          {db.port}
                        </td>
                        <td className="whitespace-nowrap px-4 py-3 text-xs text-neutral-400">
                          {db.created_at}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>
    </Shell>
  );
}
