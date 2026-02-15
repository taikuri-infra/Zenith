"use client";

import { Shell } from "@/components/shell";
import { PageHeaderSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { useApi } from "@/hooks/use-api";
import { useProject } from "@/hooks/use-project";
import { projects, type Project } from "@/lib/api";

export default function SettingsPage() {
  const projectId = useProject();

  const {
    data: project,
    loading,
    error,
    refetch,
  } = useApi(() => projects.get(projectId), [projectId]);

  if (loading) {
    return (
      <Shell>
        <div className="space-y-6">
          <PageHeaderSkeleton />
          <div className="rounded-lg border border-border bg-surface-100 p-5 space-y-4">
            <div className="animate-pulse rounded bg-surface-300 h-8 w-64" />
            <div className="animate-pulse rounded bg-surface-300 h-6 w-20" />
          </div>
        </div>
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

  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">
            Project Settings
          </h1>
          <p className="text-sm text-neutral-500">
            Manage your project configuration
          </p>
        </div>

        {/* General */}
        <section>
          <div className="mb-3">
            <h2 className="text-sm font-medium text-white">General</h2>
          </div>
          <div className="rounded-lg border border-border bg-surface-100 p-5 space-y-4">
            <div>
              <label className="mb-1.5 block text-xs font-medium text-neutral-500">
                Project Name
              </label>
              <input
                type="text"
                readOnly
                value={project?.name || ""}
                className="w-full max-w-sm rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-neutral-300 outline-none cursor-not-allowed"
              />
            </div>
            <div>
              <label className="mb-1.5 block text-xs font-medium text-neutral-500">
                Display Name
              </label>
              <input
                type="text"
                readOnly
                value={project?.display_name || ""}
                className="w-full max-w-sm rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-neutral-300 outline-none cursor-not-allowed"
              />
            </div>
            <div>
              <label className="mb-1.5 block text-xs font-medium text-neutral-500">
                Plan
              </label>
              <span className="inline-flex items-center rounded-full bg-accent-500/10 px-2.5 py-0.5 text-xs font-medium text-accent-400">
                {project?.plan || "--"}
              </span>
            </div>
            <div>
              <label className="mb-1.5 block text-xs font-medium text-neutral-500">
                Region
              </label>
              <span className="text-sm text-neutral-300">
                {project?.region || "--"}
              </span>
            </div>
          </div>
        </section>

        {/* Danger Zone */}
        <section>
          <div className="mb-3">
            <h2 className="text-sm font-medium text-white">Danger Zone</h2>
          </div>
          <div className="rounded-lg border border-red-500/30 bg-red-500/5 p-5">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-white">
                  Delete Project
                </p>
                <p className="mt-0.5 text-xs text-neutral-500">
                  Permanently delete this project and all associated resources.
                  This action cannot be undone.
                </p>
              </div>
              <button className="rounded-lg bg-red-500 px-3 py-1.5 text-sm font-medium text-white hover:bg-red-600 transition-colors flex-shrink-0">
                Delete Project
              </button>
            </div>
          </div>
        </section>
      </div>
    </Shell>
  );
}
