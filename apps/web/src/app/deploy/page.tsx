"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { Modal } from "@/components/modal";
import { useApi } from "@/hooks/use-api";
import { useDeployEvents } from "@/hooks/use-deploy-events";
import { type DeployApp } from "@/lib/api";
import { getApi } from "@/lib/get-api";
import Link from "next/link";
import { useState, useCallback } from "react";
import {
  Rocket,
  GitBranch,
  ExternalLink,
  Trash2,
  Clock,
  Box,
} from "lucide-react";

export default function DeployPage() {
  const { appsDeploy } = getApi();

  const {
    data: appsData,
    loading,
    error,
    refetch,
  } = useApi(() => appsDeploy.list(), []);

  // Auto-refresh app list when deployment events arrive
  const handleDeployEvent = useCallback(() => {
    refetch();
  }, [refetch]);
  useDeployEvents(handleDeployEvent);

  const [showCreate, setShowCreate] = useState(false);
  const [formName, setFormName] = useState("");
  const [formRepo, setFormRepo] = useState("");
  const [formBranch, setFormBranch] = useState("main");
  const [creating, setCreating] = useState(false);
  const [deleting, setDeleting] = useState<string | null>(null);

  const apps: DeployApp[] = appsData?.items ?? [];

  const runningCount = apps.filter((a) => a.status === "running").length;
  const buildingCount = apps.filter(
    (a) => a.status === "building" || a.status === "deploying"
  ).length;

  const handleCreate = async () => {
    if (!formName.trim() || !formRepo.trim()) return;
    setCreating(true);
    try {
      await appsDeploy.create({
        name: formName.trim(),
        repo_url: formRepo.trim(),
        branch: formBranch.trim() || "main",
      });
      setShowCreate(false);
      setFormName("");
      setFormRepo("");
      setFormBranch("main");
      refetch();
    } catch {
      /* modal stays open on error */
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Delete this app? All deployments will be removed.")) return;
    setDeleting(id);
    try {
      await appsDeploy.delete(id);
      refetch();
    } finally {
      setDeleting(null);
    }
  };

  if (loading) {
    return (
      <Shell>
        <PageWithTableSkeleton cols={6} rows={4} />
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

  const statusColor = (status: string) => {
    switch (status) {
      case "running":
        return "text-emerald-400";
      case "building":
      case "deploying":
        return "text-amber-400";
      case "failed":
        return "text-red-400";
      case "stopped":
        return "text-neutral-500";
      default:
        return "text-neutral-400";
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
      unknown: "—",
    };
    return labels[fw] ?? fw;
  };

  return (
    <Shell>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <div className="flex items-center gap-2.5">
              <Rocket className="h-5 w-5 text-accent-400" />
              <h1 className="text-lg font-semibold text-white">
                Deploy Engine
              </h1>
            </div>
            <p className="mt-1 text-sm text-neutral-500">
              {apps.length} app{apps.length !== 1 ? "s" : ""}
              {runningCount > 0 && (
                <span className="text-emerald-500">
                  {" "}
                  · {runningCount} running
                </span>
              )}
              {buildingCount > 0 && (
                <span className="text-amber-500">
                  {" "}
                  · {buildingCount} building
                </span>
              )}
            </p>
          </div>
          <button
            onClick={() => setShowCreate(true)}
            className="flex items-center gap-2 rounded-lg bg-accent-500 hover:bg-accent-600 text-white px-4 py-2 text-sm font-medium transition-colors"
          >
            <Rocket className="h-4 w-4" />
            Deploy from Git
          </button>
        </div>

        {/* App grid or empty state */}
        {apps.length === 0 ? (
          <EmptyState
            title="No apps deployed"
            description="Connect a Git repository and deploy your first app in seconds."
            actionLabel="Deploy from Git"
          />
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {apps.map((app) => (
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
                        : app.status === "building" ||
                            app.status === "deploying"
                          ? "bg-amber-400 animate-pulse"
                          : app.status === "failed"
                            ? "bg-red-400"
                            : "bg-neutral-600"
                    }`}
                  />
                </div>

                {/* App info */}
                <Link href={`/deploy/${app.id}`} className="block">
                  <h3 className="text-base font-semibold text-white group-hover:text-accent-400 transition-colors">
                    {app.name}
                  </h3>
                </Link>

                <div className="mt-3 space-y-2">
                  {/* Framework */}
                  <div className="flex items-center gap-2 text-xs text-neutral-400">
                    <Box className="h-3.5 w-3.5" />
                    <span>{frameworkLabel(app.framework)}</span>
                  </div>

                  {/* Branch */}
                  <div className="flex items-center gap-2 text-xs text-neutral-400">
                    <GitBranch className="h-3.5 w-3.5" />
                    <span className="font-mono">{app.branch}</span>
                  </div>

                  {/* Status */}
                  <div className="flex items-center gap-2 text-xs">
                    <Clock className="h-3.5 w-3.5 text-neutral-500" />
                    <span className={statusColor(app.status)}>
                      {app.status}
                    </span>
                  </div>
                </div>

                {/* Footer */}
                <div className="mt-4 flex items-center justify-between border-t border-border pt-3">
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
                    <span className="text-xs text-neutral-600">
                      No URL yet
                    </span>
                  )}
                  <button
                    onClick={() => handleDelete(app.id)}
                    disabled={deleting === app.id}
                    className="rounded p-1 text-neutral-600 hover:bg-red-500/10 hover:text-red-400 transition-colors disabled:opacity-50"
                    title="Delete app"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Create Modal */}
      {showCreate && (
        <Modal title="Deploy from Git" onClose={() => setShowCreate(false)}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              handleCreate();
            }}
            className="space-y-4"
          >
            <div>
              <label className="mb-1.5 block text-xs font-medium text-neutral-400">
                App Name
              </label>
              <input
                type="text"
                value={formName}
                onChange={(e) => setFormName(e.target.value)}
                placeholder="my-app"
                className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2.5 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                required
              />
              <p className="mt-1 text-[11px] text-neutral-600">
                Lowercase letters, numbers, and hyphens only
              </p>
            </div>

            <div>
              <label className="mb-1.5 block text-xs font-medium text-neutral-400">
                Git Repository URL
              </label>
              <input
                type="url"
                value={formRepo}
                onChange={(e) => setFormRepo(e.target.value)}
                placeholder="https://github.com/org/repo"
                className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2.5 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                required
              />
            </div>

            <div>
              <label className="mb-1.5 block text-xs font-medium text-neutral-400">
                Branch
              </label>
              <div className="relative">
                <GitBranch className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-500" />
                <input
                  type="text"
                  value={formBranch}
                  onChange={(e) => setFormBranch(e.target.value)}
                  placeholder="main"
                  className="w-full rounded-lg border border-border bg-surface-200 py-2.5 pl-9 pr-3 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                />
              </div>
            </div>

            <div className="flex justify-end gap-2 pt-2">
              <button
                type="button"
                onClick={() => setShowCreate(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={creating}
                className="flex items-center gap-2 rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50"
              >
                {creating ? (
                  <>
                    <span className="h-4 w-4 animate-spin rounded-full border-2 border-white/30 border-t-white" />
                    Deploying...
                  </>
                ) : (
                  <>
                    <Rocket className="h-4 w-4" />
                    Deploy
                  </>
                )}
              </button>
            </div>
          </form>
        </Modal>
      )}
    </Shell>
  );
}
