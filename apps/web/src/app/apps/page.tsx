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
import { SHOW_PLANS } from "@/lib/edition";
import { useToast } from "@/components/toast";
import { ProgressBar } from "@/components/progress-bar";
import { useAppMetrics } from "@/hooks/use-app-metrics";
import {
  appTypeLabel, formatMemory, formatPercent, frameworkLabel, sourceKind, sourceLabel, statusTone, timeAgo, toneDot, toneText,
} from "@/lib/app-view";
import Link from "next/link";
import { useMemo, useState, useEffect } from "react";
import {
  Rocket,
  Box,
  ExternalLink,
  GitBranch,
  Search,
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
  const { data: planData } = useApi(() => (SHOW_PLANS ? userPlan.get() : Promise.resolve(null)), []);
  const tier = planData?.tier ?? "free";
  const isPro = tier !== "free";

  const { toast } = useToast();
  const [appList, setAppList] = useState<App[]>([]);
  const [showDeploy, setShowDeploy] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<DeployApp | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState("");
  const [deleting, setDeleting] = useState(false);
  const [query, setQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState("");

  const deployApps: DeployApp[] = useMemo(() => deployData?.items ?? [], [deployData]);
  const metrics = useAppMetrics(deployApps);
  const visibleApps = useMemo(() => {
    const q = query.trim().toLowerCase();
    return deployApps.filter((a) => {
      if (statusFilter && a.status !== statusFilter) return false;
      if (!q) return true;
      return [a.name, a.subdomain, a.image_url, a.repo_url].some((v) => (v ?? "").toLowerCase().includes(q));
    });
  }, [deployApps, query, statusFilter]);
  const statusCounts = useMemo(() => {
    const counts: Record<string, number> = {};
    deployApps.forEach((a) => {
      counts[a.status] = (counts[a.status] ?? 0) + 1;
    });
    return counts;
  }, [deployApps]);

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

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Apps</h1>
            <p className="text-sm text-neutral-500">
              {appList.length + deployApps.length} app{appList.length + deployApps.length !== 1 ? "s" : ""}
              {(statusCounts.running ?? 0) > 0 ? `, ${statusCounts.running} running` : ""}
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

        {/* Status summary + search / filter bar */}
        {deployApps.length > 0 && (
          <div className="flex flex-wrap items-center gap-2">
            <button
              onClick={() => setStatusFilter("")}
              className={`rounded-full border px-3 py-1 text-xs transition-colors ${
                statusFilter === "" ? "border-accent-500/50 bg-accent-500/10 text-accent-400" : "border-border text-neutral-400 hover:text-white"
              }`}
            >
              All {deployApps.length}
            </button>
            {Object.entries(statusCounts).map(([status, count]) => (
              <button
                key={status}
                onClick={() => setStatusFilter(statusFilter === status ? "" : status)}
                className={`flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs transition-colors ${
                  statusFilter === status ? "border-accent-500/50 bg-accent-500/10 text-white" : "border-border text-neutral-400 hover:text-white"
                }`}
              >
                <span className={`h-1.5 w-1.5 rounded-full ${toneDot[statusTone(status)]}`} />
                {status} {count}
              </button>
            ))}
            <div className="relative ml-auto w-full sm:w-72">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-500" />
              <input
                type="text"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Search by name, image or repo…"
                className="w-full rounded-lg border border-border bg-surface-100 py-1.5 pl-9 pr-3 text-sm text-white placeholder:text-neutral-500 focus:border-accent-500 focus:outline-none"
              />
            </div>
          </div>
        )}

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
          <div className="overflow-hidden rounded-xl border border-border bg-surface-100">
            <div className="hidden grid-cols-[1.4fr_1.2fr_1.2fr_1fr_auto] gap-4 border-b border-border px-4 py-2.5 text-[11px] font-medium uppercase tracking-wider text-neutral-500 lg:grid">
              <span>Application</span>
              <span>Source</span>
              <span>Address</span>
              <span>Resources</span>
              <span className="w-8" />
            </div>
            {visibleApps.length === 0 ? (
              <p className="px-4 py-10 text-center text-sm text-neutral-500">No apps match your filters.</p>
            ) : (
              <ul className="divide-y divide-border">
                {visibleApps.map((app) => {
                  const m = metrics[app.id];
                  const tone = statusTone(app.status);
                  const fw = frameworkLabel(app.framework);
                  const isWeb = (app.app_type ?? "web") === "web";
                  return (
                    <li key={app.id} className="group grid grid-cols-1 items-center gap-3 px-4 py-3.5 transition-colors hover:bg-surface-200 lg:grid-cols-[1.4fr_1.2fr_1.2fr_1fr_auto] lg:gap-4">
                      <Link href={`/apps/${app.id}`} className="flex min-w-0 items-center gap-3">
                        <span className={`h-2.5 w-2.5 flex-shrink-0 rounded-full ${toneDot[tone]}`} />
                        <div className="min-w-0">
                          <div className="flex items-center gap-2">
                            <span className="truncate text-sm font-semibold text-white transition-colors group-hover:text-accent-400">{app.name}</span>
                            <span className="rounded border border-border px-1.5 text-[10px] text-neutral-400">{appTypeLabel(app.app_type)}</span>
                            {fw && <span className="rounded border border-border px-1.5 text-[10px] text-neutral-400">{fw}</span>}
                          </div>
                          <p className="text-[11px] text-neutral-500">
                            <span className={toneText[tone]}>{app.status}</span> · updated {timeAgo(app.updated_at)}
                            {app.status === "sleeping" && " · wakes on request (~3s)"}
                          </p>
                        </div>
                      </Link>

                      <div className="flex min-w-0 items-center gap-1.5 text-xs text-neutral-400">
                        {sourceKind(app) === "git" ? <GitBranch className="h-3.5 w-3.5 flex-shrink-0" /> : <Box className="h-3.5 w-3.5 flex-shrink-0" />}
                        <span className="truncate font-mono">{sourceLabel(app)}</span>
                      </div>

                      <div className="min-w-0 text-xs">
                        {isWeb ? (
                          app.url ? (
                            <a
                              href={app.url}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="flex items-center gap-1.5 text-accent-400 transition-colors hover:text-accent-300"
                            >
                              <ExternalLink className="h-3 w-3 flex-shrink-0" />
                              <span className="truncate">{app.url.replace(/^https?:\/\//, "")}</span>
                            </a>
                          ) : (
                            <span className="text-neutral-600">No URL yet</span>
                          )
                        ) : app.app_type === "cron" ? (
                          <span className="font-mono text-neutral-400">{app.cron_schedule || "—"}</span>
                        ) : (
                          <span className="text-neutral-500">Background process</span>
                        )}
                        {isWeb && <p className="mt-0.5 text-[11px] text-neutral-600">port {app.port}</p>}
                      </div>

                      <div className="min-w-0">
                        {m ? (
                          <div className="space-y-1">
                            <ProgressBar percent={m.cpu_percent} size="sm" label={`CPU ${formatPercent(m.cpu_percent)}`} />
                            <p className="text-[11px] text-neutral-500">
                              {formatMemory(m.memory_mb)} RAM · {m.pod_count} replica{m.pod_count === 1 ? "" : "s"}
                            </p>
                          </div>
                        ) : (
                          <p className="text-[11px] text-neutral-600">
                            {app.replicas ? `${app.replicas} replica${app.replicas === 1 ? "" : "s"}` : "—"}
                          </p>
                        )}
                      </div>

                      <button
                        onClick={() => {
                          setDeleteTarget(app);
                          setDeleteConfirm("");
                        }}
                        className="w-8 rounded p-1.5 text-neutral-600 transition-colors hover:bg-red-500/10 hover:text-red-400"
                        title="Delete app"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </li>
                  );
                })}
              </ul>
            )}
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
