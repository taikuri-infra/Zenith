"use client";

import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { StatCardRowSkeleton, TableSkeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import type { HarborProject, RegistryStats } from "@/lib/api";
import { useApi } from "@/hooks/use-api";
import { ProgressBar } from "@/components/progress-bar";
import { Container, RefreshCw } from "lucide-react";

export default function RegistryPage() {
  const apiClient = getApi();
  const stats = useApi<RegistryStats>(() => apiClient.registry.stats());
  const { data: projects, loading, error, refetch } = useApi<HarborProject[]>(
    () => apiClient.registry.projects()
  );

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-lg font-semibold text-white">Container Registry</h1>
          <button
            onClick={refetch}
            className="flex items-center gap-1.5 rounded-lg border border-border bg-surface-100 px-3 py-1.5 text-sm text-neutral-400 hover:bg-surface-200 hover:text-white transition-colors"
          >
            <RefreshCw className="h-3.5 w-3.5" />
            Refresh
          </button>
        </div>

        {/* Stats */}
        {stats.loading ? (
          <StatCardRowSkeleton />
        ) : stats.data ? (
          <div className="grid grid-cols-4 gap-4">
            <StatCard label="Projects" value={stats.data.totalProjects} sub="Harbor projects" />
            <StatCard label="Repositories" value={stats.data.totalRepos} sub="image repos" />
            <StatCard label="Total Tags" value={stats.data.totalTags} sub="image tags" />
            <StatCard label="Storage Used" value={stats.data.storageUsed} sub={`of ${stats.data.storageQuota}`} />
          </div>
        ) : null}

        {/* Projects Table */}
        {loading ? (
          <TableSkeleton columns={5} rows={4} />
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : !projects || projects.length === 0 ? (
          <EmptyState
            title="No projects"
            description="No Harbor projects have been created."
            icon={Container}
          />
        ) : (
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Project</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Repositories</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Storage</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Access</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Created</th>
                </tr>
              </thead>
              <tbody>
                {projects.map((project) => {
                  const usagePercent = project.storageQuota > 0
                    ? Math.round((project.storageUsed / project.storageQuota) * 100)
                    : 0;
                  return (
                    <tr key={project.name} className="border-b border-border last:border-0 transition-colors hover:bg-surface-200">
                      <td className="px-4 py-3 font-medium text-white">{project.name}</td>
                      <td className="px-4 py-3 text-neutral-300">{project.repoCount}</td>
                      <td className="w-48 px-4 py-3">
                        <div className="flex items-center gap-2">
                          <ProgressBar percent={usagePercent} label="" />
                          <span className="flex-shrink-0 text-xs text-neutral-500">
                            {project.storageUsedDisplay} / {project.storageQuotaDisplay}
                          </span>
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        {project.public ? (
                          <span className="rounded-full bg-blue-500/15 px-2 py-0.5 text-xs font-medium text-blue-400">Public</span>
                        ) : (
                          <span className="rounded-full bg-neutral-500/10 px-2 py-0.5 text-xs font-medium text-neutral-400">Private</span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-xs text-neutral-500">{project.createdAt}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </Shell>
  );
}
