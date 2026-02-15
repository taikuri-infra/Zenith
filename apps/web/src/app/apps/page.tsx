"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { useApi } from "@/hooks/use-api";
import { useProject } from "@/hooks/use-project";
import { apps, type App } from "@/lib/api";
import Link from "next/link";

export default function AppsPage() {
  const projectId = useProject();

  const {
    data: appsData,
    loading,
    error,
    refetch,
  } = useApi(() => apps.list(projectId), [projectId]);

  if (loading) {
    return (
      <Shell>
        <PageWithTableSkeleton cols={10} rows={5} />
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

  const appList: App[] = appsData?.items ?? [];
  const runningCount = appList.filter((a) => a.status === "running").length;
  const stoppedCount = appList.length - runningCount;

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Apps</h1>
            <p className="text-sm text-neutral-500">
              {appList.length} services, {runningCount} running
              {stoppedCount > 0 ? `, ${stoppedCount} stopped` : ""}
            </p>
          </div>
          <button className="rounded-lg bg-accent-500 hover:bg-accent-600 text-white px-3 py-1.5 text-sm transition-colors">
            + Deploy App
          </button>
        </div>

        {/* Search / filter bar */}
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
              placeholder="Filter services..."
              className="w-full rounded-lg border border-border bg-surface-100 py-1.5 pl-9 pr-3 text-sm text-white placeholder:text-neutral-500 focus:border-accent-500 focus:outline-none"
            />
          </div>
          <select className="rounded-lg border border-border bg-surface-100 px-3 py-1.5 text-sm text-neutral-400 focus:border-accent-500 focus:outline-none">
            <option value="">All statuses</option>
            <option value="running">Running</option>
            <option value="stopped">Stopped</option>
            <option value="deploying">Deploying</option>
            <option value="crashed">Crashed</option>
          </select>
        </div>

        {/* Table or Empty State */}
        {appList.length === 0 ? (
          <EmptyState
            title="No apps yet"
            description="Deploy your first application to get started."
            actionLabel="+ Deploy App"
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
                      Status
                    </th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                      Replicas
                    </th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                      CPU
                    </th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                      Memory
                    </th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                      Image
                    </th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                      Port
                    </th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                      Domain
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {appList.map((app) => (
                    <tr
                      key={app.name}
                      className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors"
                    >
                      <td className="whitespace-nowrap px-4 py-3">
                        <Link
                          href={`/apps/${app.name}`}
                          className="font-medium text-white hover:text-accent-400 transition-colors"
                        >
                          {app.name}
                        </Link>
                      </td>
                      <td className="whitespace-nowrap px-4 py-3">
                        <StatusBadge status={app.status as "running" | "deploying" | "stopped" | "crashed"} />
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-300">
                        {app.replicas}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-300">
                        {app.cpu}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-300">
                        {app.memory}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-400">
                        {app.image}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-300">
                        {app.port}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 text-xs text-neutral-300">
                        {app.domain ? (
                          <span className="text-accent-400">{app.domain}</span>
                        ) : (
                          <span className="text-neutral-500">&mdash;</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>
    </Shell>
  );
}
