"use client";

import { Shell } from "@/components/shell";
import { DashboardSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { useApi } from "@/hooks/use-api";
import { useProject } from "@/hooks/use-project";
import {
  apps,
  databases,
  projects,
  type App,
  type Database,
  type Project,
} from "@/lib/api";

export default function DocsPage() {
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

  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">Documentation</h1>
          <p className="text-sm text-neutral-500">
            Auto-generated from your infrastructure
          </p>
        </div>

        {/* Architecture */}
        <section>
          <div className="mb-3">
            <h2 className="text-sm font-medium text-white">Architecture</h2>
          </div>
          {appList.length === 0 ? (
            <div className="rounded-lg border border-border bg-surface-100 p-8 text-center">
              <p className="text-sm text-neutral-500">
                No apps deployed yet. Architecture documentation will be
                generated from your deployed services.
              </p>
            </div>
          ) : (
            <div className="overflow-hidden rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-surface-100">
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      App
                    </th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      Image
                    </th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      Port
                    </th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      Replicas
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {appList.map((app) => (
                    <tr
                      key={app.name}
                      className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors"
                    >
                      <td className="px-4 py-3 font-medium text-white">
                        {app.name}
                      </td>
                      <td className="px-4 py-3 font-mono text-xs text-neutral-400">
                        {app.image}
                      </td>
                      <td className="px-4 py-3 font-mono text-xs text-neutral-400">
                        {app.port}
                      </td>
                      <td className="px-4 py-3 text-neutral-400">
                        {app.replicas}
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
          <div className="mb-3">
            <h2 className="text-sm font-medium text-white">Databases</h2>
          </div>
          {dbList.length === 0 ? (
            <div className="rounded-lg border border-border bg-surface-100 p-8 text-center">
              <p className="text-sm text-neutral-500">
                No databases created yet.
              </p>
            </div>
          ) : (
            <div className="space-y-2">
              {dbList.map((db) => (
                <div
                  key={db.name}
                  className="rounded-lg border border-border bg-surface-100 p-4"
                >
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <p className="text-sm font-medium text-white">
                        {db.name}
                      </p>
                      <span className="inline-flex rounded bg-surface-300 px-1.5 py-0.5 text-xs text-neutral-400 capitalize">
                        {db.engine} {db.version}
                      </span>
                    </div>
                    <span className="text-xs text-neutral-500">
                      {db.storage}
                    </span>
                  </div>
                  {db.connection_string && (
                    <div className="mt-2">
                      <p className="mb-1 text-xs text-neutral-500">
                        Connection String
                      </p>
                      <div className="rounded bg-surface-200 px-3 py-2">
                        <code className="font-mono text-xs text-neutral-400">
                          {db.connection_string}
                        </code>
                      </div>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </section>

        {/* Environment */}
        <section>
          <div className="mb-3">
            <h2 className="text-sm font-medium text-white">Environment</h2>
          </div>
          <div className="rounded-lg border border-border bg-surface-100 p-5">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <p className="text-xs text-neutral-500">Project</p>
                <p className="mt-0.5 text-sm font-medium text-white">
                  {project?.display_name || project?.name || "--"}
                </p>
              </div>
              <div>
                <p className="text-xs text-neutral-500">Plan</p>
                <p className="mt-0.5 text-sm font-medium text-white">
                  {project?.plan || "--"}
                </p>
              </div>
              <div>
                <p className="text-xs text-neutral-500">Apps</p>
                <p className="mt-0.5 text-sm font-medium text-white">
                  {appList.length} services (
                  {appList.filter((a) => a.status === "running").length} running)
                </p>
              </div>
              <div>
                <p className="text-xs text-neutral-500">Databases</p>
                <p className="mt-0.5 text-sm font-medium text-white">
                  {dbList.length} instances (
                  {dbList.filter((d) => d.status === "running").length} running)
                </p>
              </div>
              <div>
                <p className="text-xs text-neutral-500">Region</p>
                <p className="mt-0.5 text-sm font-medium text-white">
                  {project?.region || "--"}
                </p>
              </div>
              <div>
                <p className="text-xs text-neutral-500">Status</p>
                <p className="mt-0.5 text-sm font-medium text-white">
                  {project?.status || "--"}
                </p>
              </div>
            </div>
          </div>
        </section>
      </div>
    </Shell>
  );
}
