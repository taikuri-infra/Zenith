"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { useApi } from "@/hooks/use-api";
import { getApi } from "@/lib/get-api";
import { type AppDatabase } from "@/lib/api";
import Link from "next/link";

const engineBadge: Record<string, { label: string; className: string }> = {
  postgresql: { label: "P", className: "bg-blue-500/20 text-blue-400" },
  mysql: { label: "M", className: "bg-orange-500/20 text-orange-400" },
  redis: { label: "R", className: "bg-red-500/20 text-red-400" },
};

export default function DatabasesPage() {
  const { userDatabases } = getApi();

  const {
    data: dbList,
    loading,
    error,
    refetch,
  } = useApi(() => userDatabases.list(), []);

  if (loading) {
    return (
      <Shell>
        <PageWithTableSkeleton cols={7} rows={3} />
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

  const dbs: AppDatabase[] = dbList || [];
  const readyCount = dbs.filter((d) => d.status === "ready").length;
  const pgCount = dbs.filter((d) => d.engine === "postgresql").length;
  const redisCount = dbs.filter((d) => d.engine === "redis").length;

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Databases</h1>
            <p className="text-sm text-neutral-500">
              {dbs.length} instances, {readyCount} ready
              {pgCount > 0 ? `, ${pgCount} PostgreSQL` : ""}
              {redisCount > 0 ? `, ${redisCount} Redis` : ""}
            </p>
          </div>
          <p className="text-xs text-neutral-600">
            Databases are managed per-app. Go to an app&apos;s Databases tab to create one.
          </p>
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
        {dbs.length === 0 ? (
          <EmptyState
            title="No databases yet"
            description="Go to an app and add a database from the Databases tab."
          />
        ) : (
          <div className="overflow-hidden rounded-lg border border-border">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b border-border bg-surface-100">
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                      Name
                    </th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                      Engine
                    </th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                      App
                    </th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                      Status
                    </th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                      Size
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
                  {dbs.map((db) => {
                    const badge = engineBadge[db.engine] ?? {
                      label: "?",
                      className: "bg-neutral-500/20 text-neutral-400",
                    };

                    return (
                      <tr
                        key={db.id}
                        className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors"
                      >
                        <td className="whitespace-nowrap px-4 py-3">
                          <span className="font-medium text-white">
                            {db.name}
                          </span>
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
                        <td className="whitespace-nowrap px-4 py-3">
                          <Link
                            href={`/apps/${db.app_id}`}
                            className="text-xs text-accent-400 hover:underline"
                          >
                            {db.app_id.slice(0, 8)}...
                          </Link>
                        </td>
                        <td className="whitespace-nowrap px-4 py-3">
                          <StatusBadge status={db.status as "ready" | "running" | "creating" | "stopped"} />
                        </td>
                        <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-400">
                          {db.size_mb} / {db.max_size_mb} MB
                        </td>
                        <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-300">
                          {db.port}
                        </td>
                        <td className="whitespace-nowrap px-4 py-3 text-xs text-neutral-400">
                          {new Date(db.created_at).toLocaleDateString()}
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
