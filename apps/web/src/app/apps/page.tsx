"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { DeployWizard } from "@/components/deploy-wizard";
import { UpgradeNudge } from "@/components/upgrade-nudge";
import { useApi } from "@/hooks/use-api";
import { useProject } from "@/hooks/use-project";
import { type App, type DeployApp } from "@/lib/api";
import { getApi } from "@/lib/get-api";
import { useToast } from "@/components/toast";
import Link from "next/link";
import { useState, useEffect } from "react";
import {
  Rocket,
  Box,
  ExternalLink,
  Clock,
  Container,
  Globe,
  Cog,
  Trash2,
  AlertTriangle,
} from "lucide-react";

export default function AppsPage() {
  const projectId = useProject();
  const { apps, appsDeploy, userPlan } = getApi();

  // CRD-based K8s apps
  const {
    data: appsData,
    loading: crdLoading,
    error: crdError,
    refetch: crdRefetch,
  } = useApi(
    () => projectId ? apps.list(projectId) : Promise.resolve({ items: [] }),
    [projectId]
  );

  // Deploy Engine apps
  const {
    data: deployData,
    loading: deployLoading,
    error: deployError,
  } = useApi(() => appsDeploy.list(projectId || undefined), [projectId]);

  // User plan (for showing Zenith registry hint)
  const { data: planData } = useApi(() => userPlan.get(), []);
  const tier = planData?.tier ?? "free";
  const isPro = tier !== "free";

  const { toast } = useToast();
  const [appList, setAppList] = useState<App[]>([]);
  const [showDeploy, setShowDeploy] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<DeployApp | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState("");
  const [deleting, setDeleting] = useState(false);

  const deployApps: DeployApp[] = deployData?.items ?? [];

  const handleDeleteApp = async () => {
    if (!deleteTarget || deleteConfirm !== deleteTarget.name) return;
    setDeleting(true);
    try {
      await appsDeploy.delete(deleteTarget.id);
      toast("success", `App "${deleteTarget.name}" deleted`);
      setDeleteTarget(null);
      setDeleteConfirm("");
      window.location.reload();
    } catch {
      toast("error", "Failed to delete app");
    } finally {
      setDeleting(false);
    }
  };

  useEffect(() => {
    if (appsData?.items) {
      setAppList(appsData.items);
    }
  }, [appsData]);

  const loading = crdLoading && deployLoading;
  const error = crdError && deployError ? crdError : null;

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
        <ErrorState message={error} onRetry={crdRefetch} />
      </Shell>
    );
  }

  const runningCount = appList.filter((a) => a.status === "running").length;
  const stoppedCount = appList.length - runningCount;

  const statusColor = (status: string) => {
    switch (status) {
      case "running":
        return "text-emerald-400";
      case "building":
      case "deploying":
        return "text-amber-400";
      case "failed":
        return "text-red-400";
      case "sleeping":
        return "text-indigo-400";
      case "stopped":
        return "text-neutral-500";
      default:
        return "text-neutral-400";
    }
  };

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Apps</h1>
            <p className="text-sm text-neutral-500">
              {appList.length + deployApps.length} app{appList.length + deployApps.length !== 1 ? "s" : ""}
              {runningCount > 0 ? `, ${runningCount} running` : ""}
              {stoppedCount > 0 ? `, ${stoppedCount} stopped` : ""}
            </p>
          </div>
          <button
            onClick={() => setShowDeploy(true)}
            className="flex items-center gap-2 rounded-lg bg-accent-500 hover:bg-accent-600 text-white px-3 py-1.5 text-sm transition-colors"
          >
            <Rocket className="h-4 w-4" />
            Deploy App
          </button>
        </div>

        {/* Upgrade nudge for plan limit */}
        {planData && tier === "free" && (
          <UpgradeNudge
            resource="apps"
            current={deployApps.length}
            limit={planData.limits.max_apps}
          />
        )}

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
              placeholder="Filter apps..."
              className="w-full rounded-lg border border-border bg-surface-100 py-1.5 pl-9 pr-3 text-sm text-white placeholder:text-neutral-500 focus:border-accent-500 focus:outline-none"
            />
          </div>
          <select className="rounded-lg border border-border bg-surface-100 px-3 py-1.5 text-sm text-neutral-400 focus:border-accent-500 focus:outline-none">
            <option value="">All statuses</option>
            <option value="running">Running</option>
            <option value="stopped">Stopped</option>
            <option value="deploying">Deploying</option>
            <option value="failed">Failed</option>
          </select>
        </div>

        {/* ── CRD Apps Table ── */}
        {appList.length === 0 && deployApps.length === 0 ? (
          <EmptyState
            title="No apps yet"
            description="Deploy a container image to get started."
            actionLabel="Deploy App"
          />
        ) : appList.length > 0 ? (
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
        ) : null}

        {/* ── Deploy Engine Apps ── */}
        {deployApps.length > 0 && (
          <div>
            <div className="flex items-center gap-2.5 mb-4">
              <Rocket className="h-5 w-5 text-accent-400" />
              <h2 className="text-base font-semibold text-white">Deployed Apps</h2>
              <span className="text-xs text-neutral-500">
                {deployApps.length} app{deployApps.length !== 1 ? "s" : ""}
              </span>
            </div>

            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {deployApps.map((app) => (
                <div
                  key={app.id}
                  className="group relative rounded-xl border border-border bg-surface-100 p-5 transition-all hover:border-accent-500/40 hover:shadow-lg hover:shadow-accent-500/5"
                >
                  {/* Status dot */}
                  <div className="absolute right-4 top-4">
                    <span
                      className={`inline-block h-2.5 w-2.5 rounded-full ${
                        app.status === "running"
                          ? "bg-emerald-400 shadow-sm shadow-emerald-400/50"
                          : app.status === "sleeping"
                            ? "bg-indigo-400 animate-pulse"
                            : app.status === "building" || app.status === "deploying"
                              ? "bg-amber-400 animate-pulse"
                              : app.status === "failed"
                                ? "bg-red-400"
                                : "bg-neutral-600"
                      }`}
                    />
                  </div>

                  <Link href={`/apps/${app.id}`} className="block">
                    <div className="flex items-center gap-2">
                      {(app.app_type ?? "web") === "web" && <Globe className="h-4 w-4 text-blue-400" />}
                      {app.app_type === "worker" && <Cog className="h-4 w-4 text-amber-400" />}
                      {app.app_type === "cron" && <Clock className="h-4 w-4 text-purple-400" />}
                      <h3 className="text-base font-semibold text-white group-hover:text-accent-400 transition-colors">
                        {app.name}
                      </h3>
                    </div>
                  </Link>

                  <div className="mt-3 space-y-2">
                    <div className="flex items-center gap-2 text-xs text-neutral-400">
                      <Box className="h-3.5 w-3.5" />
                      <span>{app.framework || "—"}</span>
                    </div>
                    {(app.app_type ?? "web") === "web" && (
                      <div className="flex items-center gap-2 text-xs text-neutral-400">
                        <Container className="h-3.5 w-3.5" />
                        <span className="font-mono text-neutral-500">{app.port}</span>
                      </div>
                    )}
                    {app.app_type === "cron" && app.cron_schedule && (
                      <div className="flex items-center gap-2 text-xs text-neutral-400">
                        <Clock className="h-3.5 w-3.5" />
                        <span className="font-mono text-neutral-500">{app.cron_schedule}</span>
                      </div>
                    )}
                    <div className="flex items-center gap-2 text-xs">
                      <Clock className="h-3.5 w-3.5 text-neutral-500" />
                      <span className={statusColor(app.status)}>
                        {app.status}
                      </span>
                    </div>
                    {app.status === "sleeping" && (
                      <p className="text-[11px] text-indigo-400/70">
                        Wakes on request (~3s)
                      </p>
                    )}
                  </div>

                  {/* Footer */}
                  <div className="mt-4 flex items-center justify-between border-t border-border pt-3">
                    <div>
                      {(app.app_type ?? "web") === "web" ? (
                        app.url ? (
                          <a
                            href={app.url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="flex items-center gap-1.5 text-xs text-accent-400 hover:text-accent-300 transition-colors"
                          >
                            <ExternalLink className="h-3 w-3" />
                            {app.subdomain}
                          </a>
                        ) : (
                          <span className="text-xs text-neutral-600">No URL yet</span>
                        )
                      ) : app.app_type === "cron" ? (
                        <span className="text-xs text-neutral-500">Schedule: {app.cron_schedule}</span>
                      ) : (
                        <span className="text-xs text-neutral-500">Background process</span>
                      )}
                    </div>
                    <button
                      onClick={(e) => {
                        e.preventDefault();
                        setDeleteTarget(app);
                        setDeleteConfirm("");
                      }}
                      className="rounded p-1 text-neutral-600 hover:bg-red-500/10 hover:text-red-400 transition-colors"
                      title="Delete app"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Deploy Wizard */}
      {showDeploy && (
        <DeployWizard onClose={() => setShowDeploy(false)} isPro={isPro} projectId={projectId} />
      )}

      {/* Delete Confirmation Modal */}
      {deleteTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="w-full max-w-md rounded-xl border border-border bg-surface-100 p-6 shadow-2xl">
            <div className="flex items-center gap-3 mb-4">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-red-500/10">
                <AlertTriangle className="h-5 w-5 text-red-400" />
              </div>
              <div>
                <h3 className="text-base font-semibold text-white">Delete App</h3>
                <p className="text-xs text-neutral-500">This action cannot be undone</p>
              </div>
            </div>

            <p className="text-sm text-neutral-300 mb-4">
              This will permanently delete <span className="font-semibold text-white">{deleteTarget.name}</span> and all its data, including databases, storage, and deployments.
            </p>

            <div className="mb-4">
              <label className="block text-xs text-neutral-400 mb-1.5">
                Type <span className="font-mono font-semibold text-white">{deleteTarget.name}</span> to confirm
              </label>
              <input
                type="text"
                value={deleteConfirm}
                onChange={(e) => setDeleteConfirm(e.target.value)}
                placeholder={deleteTarget.name}
                className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-red-500 focus:outline-none"
                autoFocus
              />
            </div>

            <div className="flex gap-3 justify-end">
              <button
                onClick={() => {
                  setDeleteTarget(null);
                  setDeleteConfirm("");
                }}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-300 hover:bg-surface-200 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleDeleteApp}
                disabled={deleteConfirm !== deleteTarget.name || deleting}
                className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              >
                {deleting ? "Deleting..." : "Delete App"}
              </button>
            </div>
          </div>
        </div>
      )}
    </Shell>
  );
}
