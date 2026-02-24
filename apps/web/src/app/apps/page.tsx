"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { Modal } from "@/components/modal";
import { useApi } from "@/hooks/use-api";
import { useProject } from "@/hooks/use-project";
import { type App, type DeployApp } from "@/lib/api";
import { getApi } from "@/lib/get-api";
import Link from "next/link";
import { useState, useEffect } from "react";
import { Rocket, GitBranch, Box, ExternalLink, Clock } from "lucide-react";

export default function AppsPage() {
  const projectId = useProject();
  const { apps, appsDeploy } = getApi();

  // CRD-based K8s apps
  const {
    data: appsData,
    loading: crdLoading,
    error: crdError,
    refetch: crdRefetch,
  } = useApi(() => apps.list(projectId), [projectId]);

  // Deploy Engine apps
  const {
    data: deployData,
    loading: deployLoading,
    error: deployError,
  } = useApi(() => appsDeploy.list(), []);

  const [appList, setAppList] = useState<App[]>([]);
  const [showDeploy, setShowDeploy] = useState(false);
  const [formName, setFormName] = useState("");
  const [formRepo, setFormRepo] = useState("");
  const [formPort, setFormPort] = useState("8080");
  const [formReplicas, setFormReplicas] = useState("1");

  const deployApps: DeployApp[] = deployData?.items ?? [];

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

  const handleDeploy = () => {
    if (!formName.trim()) return;
    const newApp: App = {
      name: formName.trim(),
      image: formRepo.trim() || `${formName.trim()}:latest`,
      replicas: parseInt(formReplicas) || 1,
      port: parseInt(formPort) || 8080,
      status: "building",
      cpu: "0.25",
      memory: "256Mi",
      domain: undefined,
      env: {},
      created_at: new Date().toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" }),
    };
    setAppList((prev) => [...prev, newApp]);
    setShowDeploy(false);
    setFormName("");
    setFormRepo("");
    setFormPort("8080");
    setFormReplicas("1");
  };

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
              {appList.length} service{appList.length !== 1 ? "s" : ""}
              {runningCount > 0 ? `, ${runningCount} running` : ""}
              {stoppedCount > 0 ? `, ${stoppedCount} stopped` : ""}
              {deployApps.length > 0 && (
                <span className="text-accent-400">
                  {" "}· {deployApps.length} Deploy Engine
                </span>
              )}
            </p>
          </div>
          <button
            onClick={() => setShowDeploy(true)}
            className="rounded-lg bg-accent-500 hover:bg-accent-600 text-white px-3 py-1.5 text-sm transition-colors"
          >
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

        {/* ── CRD Apps Table ── */}
        {appList.length === 0 && deployApps.length === 0 ? (
          <EmptyState
            title="No apps yet"
            description="Deploy your first application to get started."
            actionLabel="+ Deploy App"
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
        <div className="mt-4">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2.5">
              <Rocket className="h-5 w-5 text-accent-400" />
              <h2 className="text-base font-semibold text-white">Deploy Engine</h2>
              <span className="text-xs text-neutral-500">
                {deployApps.length} app{deployApps.length !== 1 ? "s" : ""}
              </span>
            </div>
            <Link
              href="/deploy"
              className="flex items-center gap-1.5 rounded-lg border border-accent-500/30 px-3 py-1.5 text-xs font-medium text-accent-400 hover:bg-accent-500/10 transition-colors"
            >
              <Rocket className="h-3.5 w-3.5" />
              Deploy from Git
            </Link>
          </div>

          {deployApps.length === 0 ? (
            <div className="rounded-lg border border-dashed border-border p-8 text-center">
              <Rocket className="mx-auto h-8 w-8 text-neutral-600 mb-2" />
              <p className="text-sm text-neutral-500">No Deploy Engine apps yet</p>
              <Link
                href="/deploy"
                className="mt-2 inline-block text-xs text-accent-400 hover:text-accent-300 transition-colors"
              >
                Deploy your first app from Git →
              </Link>
            </div>
          ) : (
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

                  <Link href={`/deploy/${app.id}`} className="block">
                    <h3 className="text-base font-semibold text-white group-hover:text-accent-400 transition-colors">
                      {app.name}
                    </h3>
                  </Link>

                  <div className="mt-3 space-y-2">
                    <div className="flex items-center gap-2 text-xs text-neutral-400">
                      <Box className="h-3.5 w-3.5" />
                      <span>{app.framework || "—"}</span>
                    </div>
                    <div className="flex items-center gap-2 text-xs text-neutral-400">
                      <GitBranch className="h-3.5 w-3.5" />
                      <span className="font-mono">{app.branch}</span>
                    </div>
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
                  <div className="mt-4 flex items-center border-t border-border pt-3">
                    {app.url ? (
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
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {showDeploy && (
        <Modal title="Deploy App" onClose={() => setShowDeploy(false)}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              handleDeploy();
            }}
            className="space-y-3"
          >
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">App Name</label>
              <input
                type="text"
                value={formName}
                onChange={(e) => setFormName(e.target.value)}
                placeholder="my-app"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                required
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Git Repository</label>
              <input
                type="text"
                value={formRepo}
                onChange={(e) => setFormRepo(e.target.value)}
                placeholder="https://github.com/org/repo"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Port</label>
              <input
                type="number"
                value={formPort}
                onChange={(e) => setFormPort(e.target.value)}
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Replicas</label>
              <input
                type="number"
                value={formReplicas}
                onChange={(e) => setFormReplicas(e.target.value)}
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
            </div>
            <div className="flex justify-end gap-2 pt-4">
              <button
                type="button"
                onClick={() => setShowDeploy(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors"
              >
                Deploy
              </button>
            </div>
          </form>
        </Modal>
      )}
    </Shell>
  );
}
