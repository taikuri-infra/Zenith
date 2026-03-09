"use client";

import { useState, useEffect } from "react";
import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { TableSkeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import { demoApi } from "@/lib/demo-api";
import type { Module } from "@/lib/api";
import { useApiWithFallback } from "@/hooks/use-api";
import { Package } from "lucide-react";

export default function ModulesPage() {
  const apiClient = getApi();

  const { data: modules, loading, error, refetch, isDemo } = useApiWithFallback<Module[]>(
    () => apiClient.modules.list(),
    () => demoApi.modules.list()
  );

  const [localModules, setLocalModules] = useState<Module[]>([]);
  const [updating, setUpdating] = useState(false);
  const [updatingModule, setUpdatingModule] = useState<string | null>(null);

  useEffect(() => {
    if (modules) {
      setLocalModules(modules);
    }
  }, [modules]);

  const updatesAvailable = localModules.filter(
    (m) => m.status === "update_available"
  ).length;

  const handleUpdateAll = () => {
    setUpdating(true);
    setTimeout(() => {
      setLocalModules((prev) =>
        prev.map((mod) =>
          mod.status === "update_available"
            ? { ...mod, installed: mod.latest, status: "up_to_date" as const }
            : mod
        )
      );
      setUpdating(false);
    }, 1500);
  };

  const handleUpdateSingle = (moduleName: string) => {
    setUpdatingModule(moduleName);
    setTimeout(() => {
      setLocalModules((prev) =>
        prev.map((mod) =>
          mod.name === moduleName
            ? { ...mod, installed: mod.latest, status: "up_to_date" as const }
            : mod
        )
      );
      setUpdatingModule(null);
    }, 1000);
  };

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Modules</h1>
            {localModules.length > 0 && (
              <p className="mt-1 text-sm text-neutral-500">
                {localModules.length} installed &middot; {updatesAvailable} update
                {updatesAvailable !== 1 ? "s" : ""} available
              </p>
            )}
            {isDemo && (
              <p className="mt-1 text-xs text-amber-400/70">Showing sample data</p>
            )}
          </div>
          {updatesAvailable > 0 && (
            <button
              onClick={handleUpdateAll}
              disabled={updating}
              className="rounded-lg bg-accent-600 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-accent-500 disabled:opacity-50"
            >
              {updating ? "Updating..." : "Update All"}
            </button>
          )}
        </div>

        {loading ? (
          <TableSkeleton columns={5} rows={6} />
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : localModules.length === 0 ? (
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
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Module</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Description</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Installed</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Latest</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Action</th>
                </tr>
              </thead>
              <tbody>
                {localModules.map((mod) => (
                  <tr
                    key={mod.name}
                    className="border-b border-border last:border-0 transition-colors hover:bg-surface-200"
                  >
                    <td className="px-4 py-3 font-medium text-white">{mod.name}</td>
                    <td className="px-4 py-3 text-neutral-400">{mod.description}</td>
                    <td className="px-4 py-3 font-mono text-xs text-neutral-400">{mod.installed}</td>
                    <td className="px-4 py-3 font-mono text-xs text-neutral-400">
                      {mod.latest}
                      {mod.status === "update_available" && (
                        <span className="ml-1.5 text-amber-400">&#9888;</span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <StatusBadge status={mod.status === "up_to_date" ? "healthy" : "warning"} label={mod.status === "up_to_date" ? "Up to date" : "Update available"} />
                    </td>
                    <td className="px-4 py-3">
                      {mod.status === "update_available" && (
                        <button
                          onClick={() => handleUpdateSingle(mod.name)}
                          disabled={updatingModule === mod.name || updating}
                          className="rounded-md bg-accent-600 px-2.5 py-1 text-xs font-medium text-white transition-colors hover:bg-accent-500 disabled:opacity-50"
                        >
                          {updatingModule === mod.name ? "Updating..." : "Update"}
                        </button>
                      )}
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
