"use client";

import { useEffect, useState } from "react";
import { SHOW_PLANS } from "@/lib/edition";
import { useRouter } from "next/navigation";
import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { StatusBadge } from "@/components/status-badge";
import { ActivityPanel, ApplicationsPanel, DatabasesPanel, RecentDeploymentsPanel, engineName } from "@/components/overview-panels";
import { useAppMetrics } from "@/hooks/use-app-metrics";
import { formatMemory, formatPercent } from "@/lib/app-view";
import { ProgressBar } from "@/components/progress-bar";
import { DashboardSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { useApi } from "@/hooks/use-api";
import { useProject, useProjectContext } from "@/hooks/use-project";
import { type App, type AppDatabase, type DeployApp, type Project, type UserPlanResponse } from "@/lib/api";
import { getApi } from "@/lib/get-api";
import { ArrowUpRight, Rocket, Trash2, AlertTriangle } from "lucide-react";
import Link from "next/link";

export default function OverviewPage() {
  const router = useRouter();
  const projectId = useProject();
  const { currentProject } = useProjectContext();
  const { projects, apps, appsDeploy, standaloneDatabases, userPlan } = getApi();
  const [deletingProject, setDeletingProject] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  const {
    data: projectData,
    loading: projectLoading,
    error: projectError,
    refetch: refetchProject,
  } = useApi(
    () => projectId ? projects.get(projectId) : Promise.resolve(null),
    [projectId]
  );

  const {
    data: appsData,
    loading: appsLoading,
    error: appsError,
    refetch: refetchApps,
  } = useApi(
    () => projectId ? apps.list(projectId) : Promise.resolve({ items: [] }),
    [projectId]
  );

  const {
    data: deployData,
    loading: deployLoading,
    error: deployError,
    refetch: refetchDeploy,
  } = useApi(() => appsDeploy.list(projectId || undefined), [projectId]);

  const {
    data: dbsData,
    loading: dbsLoading,
    error: dbsError,
    refetch: refetchDbs,
  } = useApi(() => standaloneDatabases.list(projectId || undefined), [projectId]);

  const {
    data: planData,
  } = useApi(() => (SHOW_PLANS ? userPlan.get() : Promise.resolve(null)), []);

  const { onboarding } = getApi();

  const {
    data: meData,
  } = useApi(() => onboarding.getMe(), []);

  useEffect(() => {
    if (meData && meData.onboarding_completed === false) {
      router.push("/onboarding");
    }
  }, [meData, router]);

  const deployListForMetrics: DeployApp[] = deployData?.items ?? [];
  const metrics = useAppMetrics(deployListForMetrics);

  const loading = projectLoading || appsLoading || deployLoading || dbsLoading;
  const error = projectError || appsError || deployError || dbsError;

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
            refetchDeploy();
            refetchDbs();
          }}
        />
      </Shell>
    );
  }

  const appList: App[] = appsData?.items ?? [];
  const deployList: DeployApp[] = deployData?.items ?? [];
  const dbList: AppDatabase[] = dbsData ?? [];
  const project: Project | null = projectData;

  const plan: UserPlanResponse | null = planData ?? null;

  const runningDeploys = deployList.filter((a) => a.status === "running").length;
  const readyDbs = dbList.filter((d) => d.status === "ready").length;

  const webCount = deployList.filter((a) => (a.app_type ?? "web") === "web").length;
  const workerCount = deployList.filter((a) => a.app_type === "worker").length;
  const cronCount = deployList.filter((a) => a.app_type === "cron").length;

  const totalServices = deployList.length + dbList.length;
  const healthyServices = runningDeploys + readyDbs;
  const healthPercent = totalServices > 0 ? Math.round((healthyServices / totalServices) * 100) : null;

  const metricList = Object.values(metrics);
  const totalMemoryMb = metricList.reduce((sum, m) => sum + m.memory_mb, 0);
  const avgCpu = metricList.length > 0 ? metricList.reduce((sum, m) => sum + m.cpu_percent, 0) / metricList.length : undefined;
  const totalPods = metricList.reduce((sum, m) => sum + m.pod_count, 0);

  const engineCounts = dbList.reduce<Record<string, number>>((acc, d) => {
    acc[d.engine] = (acc[d.engine] ?? 0) + 1;
    return acc;
  }, {});
  const engineSummary = Object.entries(engineCounts).map(([e, n]) => `${n} ${engineName(e)}`).join(" · ");

  const handleDeleteProject = async () => {
    if (!projectId) return;
    setDeletingProject(true);
    try {
      await projects.delete(projectId);
      router.push("/");
      router.refresh();
    } catch {
      setDeletingProject(false);
      setShowDeleteConfirm(false);
    }
  };

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-lg font-semibold text-white">{project?.name || currentProject?.name || "Overview"}</h1>
              {(project?.slug || currentProject?.slug) && (
                <span className="rounded-md border border-border bg-surface-200 px-1.5 py-0.5 font-mono text-[11px] text-neutral-400">
                  {project?.slug || currentProject?.slug}
                </span>
              )}
              {project?.status === "draft" && (
                <span className="inline-flex items-center gap-1 rounded-full border border-amber-500/30 bg-amber-500/10 px-2 py-0.5 text-xs font-medium text-amber-400">
                  <AlertTriangle className="h-3 w-3" />
                  Setup Incomplete
                </span>
              )}
            </div>
            <p className="text-sm text-neutral-500">
              {deployList.length} app{deployList.length === 1 ? "" : "s"} · {dbList.length} database{dbList.length === 1 ? "" : "s"}
            </p>
          </div>
          <Link
            href="/projects/new"
            className="inline-flex items-center gap-2 rounded-lg bg-accent-500 px-3.5 py-2 text-sm font-medium text-white transition-colors hover:bg-accent-600"
          >
            <Rocket className="h-4 w-4" />
            New Project
          </Link>
        </div>

        {/* Draft project banner */}
        {project?.status === "draft" && (
          <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 p-4">
            <div className="flex items-start gap-3">
              <AlertTriangle className="mt-0.5 h-4 w-4 flex-shrink-0 text-amber-400" />
              <div>
                <p className="text-sm font-medium text-amber-300">Setup not completed</p>
                <p className="mt-0.5 text-xs text-amber-400/80">
                  This project was created but never fully deployed. Go through the New Project wizard to deploy your services, or delete this project below.
                </p>
                <Link
                  href="/projects/new"
                  className="mt-2 inline-flex items-center gap-1.5 text-xs font-medium text-amber-300 hover:text-amber-200"
                >
                  <Rocket className="h-3 w-3" />
                  Start a new project
                </Link>
              </div>
            </div>
          </div>
        )}

        {/* Delete confirmation modal */}
        {showDeleteConfirm && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
            <div className="mx-4 w-full max-w-sm rounded-xl border border-border bg-surface-100 p-6 shadow-xl">
              <div className="mb-4 flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-red-500/10">
                  <Trash2 className="h-5 w-5 text-red-400" />
                </div>
                <div>
                  <h3 className="text-sm font-semibold text-white">Delete Project</h3>
                  <p className="text-xs text-neutral-500">This cannot be undone</p>
                </div>
              </div>
              <p className="mb-6 text-sm text-neutral-400">
                All apps, databases, and managed services in <span className="font-medium text-white">{project?.name}</span> will be permanently deleted.
              </p>
              <div className="flex gap-3">
                <button
                  onClick={() => setShowDeleteConfirm(false)}
                  disabled={deletingProject}
                  className="flex-1 rounded-lg border border-border bg-surface-200 px-4 py-2 text-sm text-neutral-300 transition-colors hover:text-white"
                >
                  Cancel
                </button>
                <button
                  onClick={handleDeleteProject}
                  disabled={deletingProject}
                  className="flex-1 rounded-lg bg-red-500 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-red-600 disabled:opacity-50"
                >
                  {deletingProject ? "Deleting..." : "Delete"}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Stats */}
        <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
          <StatCard
            label="Apps"
            value={`${runningDeploys}/${deployList.length}`}
            sub={`running · ${webCount} web, ${workerCount} workers, ${cronCount} cron`}
          />
          <StatCard
            label="Databases"
            value={`${readyDbs}/${dbList.length}`}
            sub={engineSummary || "ready"}
          />
          <StatCard
            label="Memory in use"
            value={metricList.length > 0 ? formatMemory(totalMemoryMb) : "--"}
            sub={metricList.length > 0 ? `CPU avg ${formatPercent(avgCpu)} · ${totalPods} pod${totalPods === 1 ? "" : "s"}` : "no running apps"}
          />
          <StatCard
            label="Services health"
            value={healthPercent === null ? "--" : `${healthPercent}%`}
            sub={totalServices > 0 ? `${healthyServices}/${totalServices} healthy` : "no services"}
          />
        </div>

        {/* Plan usage banner */}
        {SHOW_PLANS && plan && (
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <div className="mb-3 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="inline-flex rounded-full bg-accent-500/20 px-2 py-0.5 text-xs font-bold uppercase text-accent-400">{plan.tier}</span>
                <span className="text-sm text-neutral-400">Plan</span>
              </div>
              {plan.tier === "free" && (
                <Link
                  href="/billing"
                  className="rounded-lg bg-accent-500 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-accent-600"
                >
                  Upgrade to Pro
                </Link>
              )}
            </div>
            <div className="grid grid-cols-3 gap-4 text-xs">
              {([
                ["Apps", plan.usage.apps, plan.limits.max_apps, "Apps"],
                ["Databases", plan.usage.databases, plan.limits.max_databases, "Databases"],
                ["Storage", plan.usage.buckets, plan.limits.max_buckets, "buckets"],
              ] as const).map(([label, used, max]) => (
                <div key={label}>
                  <div className="mb-1 flex items-center justify-between">
                    <span className="text-neutral-500">{label}</span>
                    <span className="text-neutral-300">{used}/{max}</span>
                  </div>
                  <ProgressBar percent={max > 0 ? Math.round((used / max) * 100) : 0} size="sm" />
                </div>
              ))}
            </div>
          </div>
        )}

        {totalServices === 0 ? (
          <div className="rounded-xl border border-dashed border-accent-500/30 bg-surface-100 p-10 text-center">
            <Rocket className="mx-auto mb-3 h-8 w-8 text-accent-400" />
            <h3 className="text-base font-semibold text-white">Deploy your first project</h3>
            <p className="mt-1 text-sm text-neutral-500">Import a docker-compose.yml and launch your services in minutes.</p>
            <Link
              href="/projects/new"
              className="mt-4 inline-flex items-center gap-2 rounded-lg bg-accent-500 px-5 py-2.5 text-sm font-medium text-white transition-colors hover:bg-accent-600"
            >
              <Rocket className="h-4 w-4" />
              New Project
            </Link>
          </div>
        ) : (
          <div className="grid gap-4 lg:grid-cols-3">
            <div className="space-y-4 lg:col-span-2">
              <ApplicationsPanel apps={deployList} metrics={metrics} />
              <RecentDeploymentsPanel apps={deployList} />
            </div>
            <div className="space-y-4">
              <DatabasesPanel dbs={dbList} />
              <ActivityPanel />
            </div>
          </div>
        )}

        {/* Referral banner */}
        {SHOW_PLANS && (
          <div className="flex items-center justify-between rounded-lg border border-accent-500/20 bg-accent-500/5 p-4">
            <div>
              <p className="text-sm font-medium text-accent-400">Share Zenith, get 1 month Pro free</p>
              <p className="mt-0.5 text-xs text-neutral-500">Invite a friend — when they deploy their first app, you both get rewarded.</p>
            </div>
            <Link
              href="/settings?tab=referral"
              className="shrink-0 rounded-lg border border-accent-500/30 bg-accent-500/10 px-4 py-2 text-sm font-medium text-accent-400 transition-colors hover:bg-accent-500/20"
            >
              Get your link
            </Link>
          </div>
        )}

        {/* Legacy (cluster) apps — only present on cloud installs */}
        {appList.length > 0 && (
          <section>
            <div className="mb-3 flex items-center justify-between">
              <h2 className="text-sm font-medium text-white">Cluster apps</h2>
              <Link href="/apps" className="flex items-center gap-1 text-xs text-neutral-500 hover:text-white">
                View all <ArrowUpRight className="h-3 w-3" />
              </Link>
            </div>
            <div className="overflow-hidden rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-surface-100">
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Name</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Replicas</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">CPU</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Memory</th>
                  </tr>
                </thead>
                <tbody>
                  {appList.map((app) => (
                    <tr key={app.name} className="border-b border-border transition-colors last:border-0 hover:bg-surface-200">
                      <td className="px-4 py-3">
                        <Link href={`/apps/${app.name}`} className="font-medium text-white transition-colors hover:text-accent-400">
                          {app.name}
                        </Link>
                        {app.domain && <span className="ml-2 text-xs text-neutral-500">{app.domain}</span>}
                      </td>
                      <td className="px-4 py-3">
                        <StatusBadge status={app.status as "running" | "deploying" | "stopped" | "crashed"} />
                      </td>
                      <td className="px-4 py-3 text-neutral-300">{app.replicas}</td>
                      <td className="px-4 py-3 font-mono text-xs text-neutral-400">{app.cpu}</td>
                      <td className="px-4 py-3 font-mono text-xs text-neutral-400">{app.memory}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        )}

        {/* Danger zone */}
        <div className="flex items-center justify-between rounded-lg border border-border bg-surface-100 px-4 py-3">
          <div>
            <p className="text-sm text-neutral-300">Delete this project</p>
            <p className="text-xs text-neutral-500">Removes its apps, databases and managed services permanently.</p>
          </div>
          <button
            onClick={() => setShowDeleteConfirm(true)}
            className="flex items-center gap-1.5 rounded-lg border border-red-500/30 px-3 py-1.5 text-xs font-medium text-red-400 transition-colors hover:bg-red-500/10"
          >
            <Trash2 className="h-3.5 w-3.5" />
            Delete Project
          </button>
        </div>
      </div>
    </Shell>
  );
}
