"use client";

import { Shell } from "@/components/shell";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import {
  CardSkeleton,
  TableSkeleton,
} from "@/components/loading-skeleton";
import { DemoButton } from "@/components/demo-button";
import { getApi, isDemoMode } from "@/lib/get-api";
import type { PlatformUpdate, UpdateHistoryEntry } from "@/lib/api";
import { useApi, useMutation } from "@/hooks/use-api";
import { ArrowUpCircle } from "lucide-react";

export default function UpdatesPage() {
  const apiClient = getApi();
  const demo = isDemoMode();

  const {
    data: platformUpdate,
    loading: updateLoading,
    error: updateError,
    refetch: refetchUpdate,
  } = useApi<PlatformUpdate>(() => apiClient.updates.check());

  const {
    data: history,
    loading: historyLoading,
    error: historyError,
    refetch: refetchHistory,
  } = useApi<UpdateHistoryEntry[]>(() => apiClient.updates.history());

  const applyMutation = useMutation((version: string) =>
    apiClient.updates.apply(version)
  );

  const handleUpgrade = async () => {
    if (demo || !platformUpdate) return;
    try {
      await applyMutation.execute(platformUpdate.version);
      refetchUpdate();
      refetchHistory();
    } catch {
      // error is set in the mutation hook
    }
  };

  return (
    <Shell>
      <div className="space-y-6">
        <h1 className="text-lg font-semibold text-white">Platform Updates</h1>

        {/* Available update */}
        {updateLoading ? (
          <CardSkeleton />
        ) : updateError ? (
          <ErrorState error={updateError} onRetry={refetchUpdate} />
        ) : !platformUpdate ? (
          <EmptyState
            title="No updates available"
            description="You are running the latest version of Zenith."
            icon={ArrowUpCircle}
          />
        ) : (
          <div className="rounded-lg border border-accent-600/30 bg-accent-600/5 p-5">
            <div className="flex items-start justify-between">
              <div>
                <div className="flex items-center gap-2">
                  <span className="rounded bg-accent-600/20 px-1.5 py-0.5 text-[10px] font-medium text-accent-400">
                    NEW
                  </span>
                  <h2 className="text-base font-semibold text-white">
                    Zenith {platformUpdate.version}
                  </h2>
                </div>
                <p className="mt-1 text-sm text-neutral-500">
                  Released {platformUpdate.releasedAt} &middot; Current version:{" "}
                  {platformUpdate.current}
                </p>
                {platformUpdate.breakingChanges && (
                  <p className="mt-1 text-xs text-amber-400">
                    Contains breaking changes
                  </p>
                )}
              </div>
              <DemoButton
                onClick={handleUpgrade}
                disabled={applyMutation.loading}
                className="rounded-lg bg-accent-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-accent-500 disabled:opacity-50"
              >
                {applyMutation.loading
                  ? "Upgrading..."
                  : `Upgrade to ${platformUpdate.version}`}
              </DemoButton>
            </div>

            {applyMutation.error && (
              <p className="mt-2 text-xs text-red-400">
                Upgrade failed: {applyMutation.error.message}
              </p>
            )}

            <div className="mt-4">
              <h3 className="text-sm font-medium text-white">New Features</h3>
              <ul className="mt-2 space-y-1.5">
                {platformUpdate.features.map((feature, i) => (
                  <li
                    key={i}
                    className="flex items-start gap-2 text-sm text-neutral-300"
                  >
                    <span className="mt-1.5 h-1 w-1 flex-shrink-0 rounded-full bg-accent-400" />
                    {feature}
                  </li>
                ))}
              </ul>
            </div>
          </div>
        )}

        {/* Update history */}
        <section>
          <h2 className="mb-3 text-sm font-medium text-white">
            Update History
          </h2>
          {historyLoading ? (
            <TableSkeleton columns={3} rows={4} />
          ) : historyError ? (
            <ErrorState error={historyError} onRetry={refetchHistory} />
          ) : !history || history.length === 0 ? (
            <EmptyState
              title="No update history"
              description="No previous updates have been recorded."
            />
          ) : (
            <div className="overflow-hidden rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-surface-100">
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      Version
                    </th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      Date
                    </th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      Status
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {history.map((entry, i) => (
                    <tr
                      key={entry.version}
                      className={`transition-colors hover:bg-surface-200 ${
                        i < history.length - 1
                          ? "border-b border-border"
                          : ""
                      }`}
                    >
                      <td className="px-4 py-3 font-mono text-xs text-white">
                        {entry.version}
                      </td>
                      <td className="px-4 py-3 text-neutral-400">
                        {entry.date}
                      </td>
                      <td className="px-4 py-3 text-xs">
                        <span
                          className={
                            entry.status === "installed"
                              ? "text-emerald-400"
                              : "text-neutral-500"
                          }
                        >
                          {entry.status === "installed"
                            ? "Installed (current)"
                            : "Superseded"}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </div>
    </Shell>
  );
}
