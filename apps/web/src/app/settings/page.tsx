"use client";

import { Shell } from "@/components/shell";
import { PageHeaderSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { Modal } from "@/components/modal";
import { useApi } from "@/hooks/use-api";
import { useProject } from "@/hooks/use-project";
import { projects, type Project } from "@/lib/api";
import { useState } from "react";

export default function SettingsPage() {
  const projectId = useProject();

  const {
    data: project,
    loading,
    error,
    refetch,
  } = useApi(() => projects.get(projectId), [projectId]);

  const [showDelete, setShowDelete] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState("");
  const [deleted, setDeleted] = useState(false);

  const handleDelete = () => {
    if (deleteConfirm === project?.name) {
      setDeleted(true);
      setShowDelete(false);
      setDeleteConfirm("");
    }
  };

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

        {deleted && (
          <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-4 py-3">
            <p className="text-xs text-red-400">Project deletion initiated. This is a demo -- no actual resources were removed.</p>
          </div>
        )}

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
              <button
                onClick={() => setShowDelete(true)}
                className="rounded-lg bg-red-500 px-3 py-1.5 text-sm font-medium text-white hover:bg-red-600 transition-colors flex-shrink-0"
              >
                Delete Project
              </button>
            </div>
          </div>
        </section>
      </div>

      {showDelete && (
        <Modal title="Delete Project" onClose={() => { setShowDelete(false); setDeleteConfirm(""); }}>
          <div className="space-y-3">
            <p className="text-sm text-neutral-400">
              This action is irreversible. All apps, databases, storage, and configurations will be permanently deleted.
            </p>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">
                Type <span className="font-mono text-white">{project?.name || "project-name"}</span> to confirm
              </label>
              <input
                type="text"
                value={deleteConfirm}
                onChange={(e) => setDeleteConfirm(e.target.value)}
                placeholder={project?.name || "project-name"}
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
            </div>
            <div className="flex justify-end gap-2 pt-4">
              <button
                type="button"
                onClick={() => { setShowDelete(false); setDeleteConfirm(""); }}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleDelete}
                disabled={deleteConfirm !== project?.name}
                className={`rounded-lg px-4 py-2 text-sm font-medium text-white transition-colors ${
                  deleteConfirm === project?.name
                    ? "bg-red-500 hover:bg-red-600"
                    : "bg-red-500/30 cursor-not-allowed"
                }`}
              >
                Delete Project
              </button>
            </div>
          </div>
        </Modal>
      )}
    </Shell>
  );
}
