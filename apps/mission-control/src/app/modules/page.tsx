"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { TableSkeleton } from "@/components/loading-skeleton";
import { api } from "@/lib/api";
import type { Module } from "@/lib/api";
import { useApi, useMutation } from "@/hooks/use-api";
import { Package } from "lucide-react";

export default function ModulesPage() {
  const { data: modules, loading, error, refetch } = useApi<Module[]>(
    () => api.modules.list()
  );

  const updateAllMutation = useMutation(() => api.modules.updateAll());

  const updatesAvailable = modules
    ? modules.filter((m) => m.status === "update_available").length
    : 0;

  const handleUpdateAll = async () => {
    try {
      await updateAllMutation.execute(undefined);
      refetch();
    } catch {
      // error is set in the mutation hook
    }
  };

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Modules</h1>
            {modules && (
              <p className="mt-1 text-sm text-neutral-500">
                {modules.length} installed &middot; {updatesAvailable} update
                {updatesAvailable !== 1 ? "s" : ""} available
              </p>
            )}
          </div>
          {updatesAvailable > 0 && (
            <button
              onClick={handleUpdateAll}
              disabled={updateAllMutation.loading}
              className="rounded-lg bg-accent-600 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-accent-500 disabled:opacity-50"
            >
              {updateAllMutation.loading ? "Updating..." : "Update All"}
            </button>
          )}
        </div>

        {updateAllMutation.error && (
          <div className="rounded-lg border border-red-500/20 bg-red-500/5 px-4 py-2 text-xs text-red-400">
            Failed to update all modules: {updateAllMutation.error.message}
          </div>
        )}

        {loading ? (
          <TableSkeleton columns={5} rows={6} />
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : !modules || modules.length === 0 ? (
          <EmptyState
            title="No modules"
            description="No modules have been installed yet."
            icon={Package}
          />
        ) : (
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    Module
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    Description
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    Installed
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    Latest
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    Status
                  </th>
                </tr>
              </thead>
              <tbody>
                {modules.map((mod) => (
                  <tr
                    key={mod.name}
                    className="border-b border-border last:border-0 transition-colors hover:bg-surface-200"
                  >
                    <td className="px-4 py-3 font-medium text-white">
                      {mod.name}
                    </td>
                    <td className="px-4 py-3 text-neutral-400">
                      {mod.description}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-neutral-400">
                      {mod.installed}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-neutral-400">
                      {mod.latest}
                      {mod.status === "update_available" && (
                        <span className="ml-1.5 text-amber-400">&#9888;</span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <StatusBadge status={mod.status} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </Shell>
  );
}
