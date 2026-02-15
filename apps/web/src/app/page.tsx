"use client";

import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { StatusBadge } from "@/components/status-badge";
import { DashboardSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { useApi } from "@/hooks/use-api";
import { useProject } from "@/hooks/use-project";
import { apps, databases, projects, type App, type Database, type Project } from "@/lib/api";
import { ArrowUpRight } from "lucide-react";
import Link from "next/link";

export default function OverviewPage() {
  const projectId = useProject();

  const {
    data: projectData,
    loading: projectLoading,
    error: projectError,
    refetch: refetchProject,
  } = useApi(() => projects.get(projectId), [projectId]);

  const {
    data: appsData,
    loading: appsLoading,
    error: appsError,
    refetch: refetchApps,
  } = useApi(() => apps.list(projectId), [projectId]);

  const {
    data: dbsData,
    loading: dbsLoading,
    error: dbsError,
    refetch: refetchDbs,
  } = useApi(() => databases.list(projectId), [projectId]);

  const loading = projectLoading || appsLoading || dbsLoading;
  const error = projectError || appsError || dbsError;

  if (loading) {
    return (
      <Shell>
        <DashboardSkeleton />
      </Shell>
    );
  }

  if (error) {
    return (
      <Shell>
        <ErrorState
          message={error}
          onRetry={() => {
            refetchProject();
            refetchApps();
            refetchDbs();
          }}
        />
      </Shell>
    );
  }

  const appList: App[] = appsData?.items ?? [];
  const dbList: Database[] = dbsData?.items ?? [];
  const project: Project | null = projectData;

  const runningApps = appList.filter((a) => a.status === "running").length;

  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">
            {project?.display_name || project?.name || "Project"}
          </h1>
          <p className="text-sm text-neutral-500">Project overview</p>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-4 gap-4">
          <StatCard
            label="Apps"
            value={`${runningApps}/${appList.length}`}
            sub={`${runningApps} running`}
          />
          <StatCard
            label="Databases"
            value={dbList.length}
            sub="all healthy"
          />
          <StatCard
            label="Region"
            value={project?.region || "--"}
            sub={project?.plan || ""}
          />
          <StatCard
            label="Status"
            value={project?.status || "--"}
            sub=""
          />
        </div>

        {/* Apps table */}
        <section>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-sm font-medium text-white">Apps</h2>
            <Link
              href="/apps"
              className="flex items-center gap-1 text-xs text-neutral-500 hover:text-white"
            >
              View all <ArrowUpRight className="h-3 w-3" />
            </Link>
          </div>
          {appList.length === 0 ? (
            <div className="rounded-lg border border-border bg-surface-100 p-8 text-center">
              <p className="text-sm text-neutral-500">No apps deployed yet</p>
              <Link
                href="/apps"
                className="mt-2 inline-block text-sm text-accent-400 hover:text-accent-300"
              >
                Deploy your first app
              </Link>
            </div>
          ) : (
            <div className="overflow-hidden rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-surface-100">
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      Name
                    </th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      Status
                    </th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      Replicas
                    </th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      CPU
                    </th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      Memory
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {appList.map((app) => (
                    <tr
                      key={app.name}
                      className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors"
                    >
                      <td className="px-4 py-3">
                        <Link
                          href={`/apps/${app.name}`}
                          className="font-medium text-white hover:text-accent-400 transition-colors"
                        >
                          {app.name}
                        </Link>
                        {app.domain && (
                          <span className="ml-2 text-xs text-neutral-500">
                            {app.domain}
                          </span>
                        )}
                      </td>
                      <td className="px-4 py-3">
                        <StatusBadge status={app.status as "running" | "deploying" | "stopped" | "crashed"} />
                      </td>
                      <td className="px-4 py-3 text-neutral-300">
                        {app.replicas}
                      </td>
                      <td className="px-4 py-3 font-mono text-xs text-neutral-400">
                        {app.cpu}
                      </td>
                      <td className="px-4 py-3 font-mono text-xs text-neutral-400">
                        {app.memory}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>

        {/* Databases */}
        <section>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-sm font-medium text-white">Databases</h2>
            <Link
              href="/databases"
              className="flex items-center gap-1 text-xs text-neutral-500 hover:text-white"
            >
              View all <ArrowUpRight className="h-3 w-3" />
            </Link>
          </div>
          {dbList.length === 0 ? (
            <div className="rounded-lg border border-border bg-surface-100 p-8 text-center">
              <p className="text-sm text-neutral-500">No databases created yet</p>
              <Link
                href="/databases"
                className="mt-2 inline-block text-sm text-accent-400 hover:text-accent-300"
              >
                Create your first database
              </Link>
            </div>
          ) : (
            <div className="space-y-2">
              {dbList.map((db) => (
                <Link
                  key={db.name}
                  href={`/databases/${db.name}`}
                  className="flex items-center justify-between rounded-lg border border-border bg-surface-100 p-3 transition-colors hover:border-border-hover"
                >
                  <div>
                    <span className="text-sm font-medium text-white">
                      {db.name}
                    </span>
                    <span className="ml-2 text-xs text-neutral-500 capitalize">
                      {db.engine} {db.version}
                    </span>
                  </div>
                  <div className="text-right">
                    <span className="text-xs text-neutral-400">
                      {db.storage}
                    </span>
                  </div>
                </Link>
              ))}
            </div>
          )}
        </section>
      </div>
    </Shell>
  );
}
