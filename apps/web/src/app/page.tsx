"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { StatusBadge } from "@/components/status-badge";
import { ProgressBar } from "@/components/progress-bar";
import { DashboardSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { useApi } from "@/hooks/use-api";
import { useProject } from "@/hooks/use-project";
import { type App, type AppDatabase, type DeployApp, type Project, type UserPlanResponse } from "@/lib/api";
import { getApi } from "@/lib/get-api";
import { ArrowUpRight, Rocket, GitBranch, Box, Database, HardDrive, Activity, Globe, Cog, Clock } from "lucide-react";
import Link from "next/link";

const engineBadge: Record<string, { label: string; className: string }> = {
  postgresql: { label: "P", className: "bg-blue-500/20 text-blue-400" },
  mysql: { label: "M", className: "bg-orange-500/20 text-orange-400" },
  redis: { label: "R", className: "bg-red-500/20 text-red-400" },
};

export default function OverviewPage() {
  const router = useRouter();
  const projectId = useProject();
  const { projects, apps, appsDeploy, standaloneDatabases, userPlan } = getApi();

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
  } = useApi(() => userPlan.get(), []);

  const { onboarding } = getApi();
  const {
    data: meData,
  } = useApi(() => onboarding.getMe(), []);

  useEffect(() => {
    if (meData && meData.onboarding_completed === false) {
      router.push("/onboarding");
    }
  }, [meData, router]);

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

  const runningApps = appList.filter((a) => a.status === "running").length;
  const runningDeploys = deployList.filter((a) => a.status === "running").length;
  const buildingDeploys = deployList.filter(
    (a) => a.status === "building" || a.status === "deploying"
  ).length;
  const readyDbs = dbList.filter((d) => d.status === "ready").length;

  const webCount = deployList.filter((a) => (a.app_type ?? "web") === "web").length;
  const workerCount = deployList.filter((a) => a.app_type === "worker").length;
  const cronCount = deployList.filter((a) => a.app_type === "cron").length;

  // Total service health: count all healthy services vs total
  const totalServices = deployList.length + dbList.length;
  const healthyServices = runningDeploys + readyDbs;

  const statusColor = (status: string) => {
    switch (status) {
      case "running":
      case "ready":
        return "text-emerald-400";
      case "building":
      case "deploying":
      case "provisioning":
        return "text-amber-400";
      case "failed":
      case "error":
        return "text-red-400";
      default:
        return "text-neutral-500";
    }
  };

  const frameworkLabel = (fw: string) => {
    const labels: Record<string, string> = {
      nextjs: "Next.js",
      go: "Go",
      python: "Python",
      django: "Django",
      flask: "Flask",
      rails: "Rails",
      express: "Express",
      static: "Static",
      dockerfile: "Dockerfile",
    };
    return labels[fw] ?? fw;
  };

  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">
            {project?.name || "Project"}
          </h1>
          <p className="text-sm text-neutral-500">Project overview</p>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5">
          <StatCard
            label="Apps"
            value={`${runningDeploys}/${deployList.length}`}
            sub={`${webCount} web, ${workerCount} workers, ${cronCount} cron`}
          />
          <StatCard
            label="Databases"
            value={`${readyDbs}/${dbList.length}`}
            sub={`${readyDbs} ready`}
          />
          <StatCard
            label="Services Health"
            value={totalServices > 0 ? `${Math.round((healthyServices / totalServices) * 100)}%` : "--"}
            sub={totalServices > 0 ? `${healthyServices}/${totalServices} healthy` : "no services"}
          />
          <StatCard
            label="Plan"
            value={planData?.tier || "--"}
            sub={project?.slug || ""}
          />
          <StatCard
            label="Legacy Apps"
            value={`${runningApps}/${appList.length}`}
            sub={`${runningApps} running`}
          />
        </div>

        {/* Plan usage banner */}
        {plan && (
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-bold uppercase ${
                  plan.tier === "free" ? "bg-neutral-500/20 text-neutral-400" :
                  plan.tier === "pro" ? "bg-accent-500/20 text-accent-400" :
                  plan.tier === "team" ? "bg-purple-500/20 text-purple-400" :
                  "bg-amber-500/20 text-amber-400"
                }`}>
                  {plan.tier}
                </span>
                <span className="text-sm text-neutral-400">Plan</span>
              </div>
              {plan.tier === "free" && (
                <Link
                  href="/billing"
                  className="rounded-lg bg-accent-500 px-3 py-1.5 text-xs font-medium text-white hover:bg-accent-600 transition-colors"
                >
                  Upgrade to Pro
                </Link>
              )}
            </div>
            <div className="grid grid-cols-3 gap-4 text-xs">
              <div>
                <div className="flex items-center justify-between mb-1">
                  <span className="text-neutral-500">Apps</span>
                  <span className="text-neutral-300">{plan.usage.apps}/{plan.limits.max_apps}</span>
                </div>
                <ProgressBar percent={plan.limits.max_apps > 0 ? Math.round((plan.usage.apps / plan.limits.max_apps) * 100) : 0} size="sm" />
              </div>
              <div>
                <div className="flex items-center justify-between mb-1">
                  <span className="text-neutral-500">Databases</span>
                  <span className="text-neutral-300">{plan.usage.databases}/{plan.limits.max_databases}</span>
                </div>
                <ProgressBar percent={plan.limits.max_databases > 0 ? Math.round((plan.usage.databases / plan.limits.max_databases) * 100) : 0} size="sm" />
              </div>
              <div>
                <div className="flex items-center justify-between mb-1">
                  <span className="text-neutral-500">Storage</span>
                  <span className="text-neutral-300">{plan.usage.buckets}/{plan.limits.max_buckets} buckets</span>
                </div>
                <ProgressBar percent={plan.limits.max_buckets > 0 ? Math.round((plan.usage.buckets / plan.limits.max_buckets) * 100) : 0} size="sm" />
              </div>
            </div>
          </div>
        )}

        {/* Service Health Grid */}
        <section>
          <div className="mb-3 flex items-center gap-2">
            <Activity className="h-4 w-4 text-accent-400" />
            <h2 className="text-sm font-medium text-white">Service Health</h2>
          </div>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {/* Apps */}
            {deployList.map((app) => (
              <Link
                key={app.id}
                href={`/apps/${app.id}`}
                className="group flex items-start gap-3 rounded-lg border border-border bg-surface-100 p-4 transition-all hover:border-accent-500/40 hover:shadow-lg hover:shadow-accent-500/5"
              >
                <span
                  className={`mt-1 inline-block h-2 w-2 flex-shrink-0 rounded-full ${
                    app.status === "running"
                      ? "bg-emerald-400"
                      : app.status === "building" || app.status === "deploying"
                        ? "bg-amber-400 animate-pulse"
                        : app.status === "failed"
                          ? "bg-red-400"
                          : "bg-neutral-600"
                  }`}
                />
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    {(app.app_type ?? "web") === "web" && <Globe className="h-3.5 w-3.5 text-blue-400" />}
                    {app.app_type === "worker" && <Cog className="h-3.5 w-3.5 text-amber-400" />}
                    {app.app_type === "cron" && <Clock className="h-3.5 w-3.5 text-purple-400" />}
                    <span className="truncate text-sm font-medium text-white group-hover:text-accent-400 transition-colors">
                      {app.name}
                    </span>
                  </div>
                  <div className="mt-1.5 flex items-center gap-3 text-[11px] text-neutral-500">
                    <span className="flex items-center gap-1">
                      <Box className="h-3 w-3" />
                      {frameworkLabel(app.framework)}
                    </span>
                    <span className="flex items-center gap-1">
                      <GitBranch className="h-3 w-3" />
                      {app.branch}
                    </span>
                    <span className={statusColor(app.status)}>{app.status}</span>
                  </div>
                </div>
              </Link>
            ))}

            {/* Databases */}
            {dbList.map((db) => {
              const badge = engineBadge[db.engine] ?? {
                label: "?",
                className: "bg-neutral-500/20 text-neutral-400",
              };
              const usagePercent = db.max_size_mb > 0 ? Math.round((db.size_mb / db.max_size_mb) * 100) : 0;
              return (
                <Link
                  key={db.id}
                  href={`/apps/${db.app_id}`}
                  className="group flex items-start gap-3 rounded-lg border border-border bg-surface-100 p-4 transition-all hover:border-accent-500/40 hover:shadow-lg hover:shadow-accent-500/5"
                >
                  <span
                    className={`mt-1 inline-block h-2 w-2 flex-shrink-0 rounded-full ${
                      db.status === "ready"
                        ? "bg-emerald-400"
                        : db.status === "provisioning"
                          ? "bg-amber-400 animate-pulse"
                          : db.status === "error"
                            ? "bg-red-400"
                            : "bg-neutral-600"
                    }`}
                  />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <Database className="h-3.5 w-3.5 text-neutral-500" />
                      <span className="truncate text-sm font-medium text-white group-hover:text-accent-400 transition-colors">
                        {db.name}
                      </span>
                      <span className={`inline-flex h-4 w-4 items-center justify-center rounded text-[9px] font-bold ${badge.className}`}>
                        {badge.label}
                      </span>
                    </div>
                    <div className="mt-1.5 flex items-center gap-3 text-[11px] text-neutral-500">
                      <span className="flex items-center gap-1">
                        <HardDrive className="h-3 w-3" />
                        {db.size_mb} / {db.max_size_mb} MB
                      </span>
                      <span className={statusColor(db.status)}>{db.status}</span>
                    </div>
                    {db.max_size_mb > 0 && (
                      <div className="mt-2">
                        <ProgressBar percent={usagePercent} size="sm" />
                      </div>
                    )}
                  </div>
                </Link>
              );
            })}

            {totalServices === 0 && (
              <div className="col-span-full rounded-lg border border-border bg-surface-100 p-8 text-center">
                <Activity className="mx-auto mb-2 h-6 w-6 text-neutral-600" />
                <p className="text-sm text-neutral-500">No services provisioned yet</p>
                <Link
                  href="/apps"
                  className="mt-2 inline-block text-sm text-accent-400 hover:text-accent-300"
                >
                  Deploy your first app
                </Link>
              </div>
            )}
          </div>
        </section>

        {/* Referral banner */}
        <div className="rounded-lg border border-accent-500/20 bg-accent-500/5 p-4 flex items-center justify-between">
          <div>
            <p className="text-sm font-medium text-accent-400">Share Zenith, get 1 month Pro free</p>
            <p className="text-xs text-neutral-500 mt-0.5">Invite a friend — when they deploy their first app, you both get rewarded.</p>
          </div>
          <Link
            href="/settings?tab=referral"
            className="shrink-0 rounded-lg bg-accent-500/10 border border-accent-500/30 px-4 py-2 text-sm font-medium text-accent-400 hover:bg-accent-500/20 transition-colors"
          >
            Get your link
          </Link>
        </div>

        {/* Legacy Apps table */}
        {appList.length > 0 && (
          <section>
            <div className="mb-3 flex items-center justify-between">
              <h2 className="text-sm font-medium text-white">Apps (Legacy)</h2>
              <Link
                href="/apps"
                className="flex items-center gap-1 text-xs text-neutral-500 hover:text-white"
              >
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
      </div>
    </Shell>
  );
}
