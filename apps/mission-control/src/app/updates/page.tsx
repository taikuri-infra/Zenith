"use client";

import { useState, useEffect, useRef } from "react";
import { Shell } from "@/components/shell";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import {
  CardSkeleton,
  TableSkeleton,
} from "@/components/loading-skeleton";
import { Modal } from "@/components/modal";
import { getApi } from "@/lib/get-api";
import type { PlatformUpdate, UpdateHistoryEntry } from "@/lib/api";
import { useApi } from "@/hooks/use-api";
import { ArrowUpCircle } from "lucide-react";

const UPGRADE_STEPS = [
  "Downloading...",
  "Installing...",
  "Restarting...",
  "Complete!",
];

export default function UpdatesPage() {
  const apiClient = getApi();

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

  const [localUpdate, setLocalUpdate] = useState<PlatformUpdate | null>(null);
  const [localHistory, setLocalHistory] = useState<UpdateHistoryEntry[]>([]);
  const [showProgressModal, setShowProgressModal] = useState(false);
  const [upgradeStep, setUpgradeStep] = useState(0);
  const [upgradeProgress, setUpgradeProgress] = useState(0);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    if (platformUpdate) {
      setLocalUpdate(platformUpdate);
    }
  }, [platformUpdate]);

  useEffect(() => {
    if (history) {
      setLocalHistory(history);
    }
  }, [history]);

  const handleUpgrade = () => {
    if (!localUpdate) return;
    setShowProgressModal(true);
    setUpgradeStep(0);
    setUpgradeProgress(0);

    let progress = 0;
    let step = 0;

    intervalRef.current = setInterval(() => {
      progress += 5;
      if (progress >= 100) progress = 100;

      const newStep = Math.min(
        Math.floor(progress / 25),
        UPGRADE_STEPS.length - 1
      );
      step = newStep;

      setUpgradeProgress(progress);
      setUpgradeStep(step);

      if (progress >= 100) {
        if (intervalRef.current) clearInterval(intervalRef.current);
      }
    }, 200);
  };

  const handleProgressClose = () => {
    if (intervalRef.current) clearInterval(intervalRef.current);
    if (upgradeProgress >= 100 && localUpdate) {
      const newHistoryEntry: UpdateHistoryEntry = {
        version: localUpdate.version,
        date: new Date().toISOString().split("T")[0],
        status: "installed",
      };
      setLocalHistory((prev) => {
        const updated = prev.map((e) =>
          e.status === "installed" ? { ...e, status: "superseded" as const } : e
        );
        return [newHistoryEntry, ...updated];
      });
      setLocalUpdate(null);
    }
    setShowProgressModal(false);
    setUpgradeStep(0);
    setUpgradeProgress(0);
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
        ) : !localUpdate ? (
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
                    Zenith {localUpdate.version}
                  </h2>
                </div>
                <p className="mt-1 text-sm text-neutral-500">
                  Released {localUpdate.releasedAt} &middot; Current version:{" "}
                  {localUpdate.current}
                </p>
                {localUpdate.breakingChanges && (
                  <p className="mt-1 text-xs text-amber-400">
                    Contains breaking changes
                  </p>
                )}
              </div>
              <button
                onClick={handleUpgrade}
                className="rounded-lg bg-accent-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-accent-500"
              >
                Upgrade to {localUpdate.version}
              </button>
            </div>

            <div className="mt-4">
              <h3 className="text-sm font-medium text-white">New Features</h3>
              <ul className="mt-2 space-y-1.5">
                {localUpdate.features.map((feature, i) => (
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
          ) : localHistory.length === 0 ? (
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
                  {localHistory.map((entry, i) => (
                    <tr
                      key={`${entry.version}-${i}`}
                      className={`transition-colors hover:bg-surface-200 ${
                        i < localHistory.length - 1
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

      {showProgressModal && (
        <Modal title="Upgrading Platform" onClose={handleProgressClose}>
          <div className="space-y-4">
            <div className="space-y-2">
              {UPGRADE_STEPS.map((step, i) => (
                <div
                  key={step}
                  className={`flex items-center gap-2 text-sm ${
                    i < upgradeStep
                      ? "text-emerald-400"
                      : i === upgradeStep
                      ? "text-white font-medium"
                      : "text-neutral-600"
                  }`}
                >
                  {i < upgradeStep ? (
                    <span className="text-emerald-400">&#10003;</span>
                  ) : i === upgradeStep ? (
                    <span className="inline-block h-3 w-3 animate-spin rounded-full border-2 border-accent-500 border-t-transparent" />
                  ) : (
                    <span className="inline-block h-3 w-3 rounded-full border border-neutral-700" />
                  )}
                  {step}
                </div>
              ))}
            </div>
            <div className="h-2 w-full overflow-hidden rounded-full bg-surface-200">
              <div
                className="h-full rounded-full bg-accent-500 transition-all duration-300"
                style={{ width: `${upgradeProgress}%` }}
              />
            </div>
            <p className="text-center text-xs text-neutral-500">
              {upgradeProgress}%
            </p>
            {upgradeProgress >= 100 && (
              <div className="flex justify-end pt-2">
                <button
                  onClick={handleProgressClose}
                  className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors"
                >
                  Done
                </button>
              </div>
            )}
          </div>
        </Modal>
      )}
    </Shell>
  );
}
