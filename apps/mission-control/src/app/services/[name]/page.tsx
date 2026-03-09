"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { ErrorState } from "@/components/error-state";
import { Skeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import type { ServiceDetailItem } from "@/lib/api";
import { useApi } from "@/hooks/use-api";
import { useParams } from "next/navigation";
import Link from "next/link";
import { RefreshCw, RotateCcw } from "lucide-react";
import { useState } from "react";

function statusToBadge(status: string): "healthy" | "warning" | "error" | "idle" {
  switch (status) {
    case "healthy":
      return "healthy";
    case "degraded":
      return "warning";
    case "down":
      return "error";
    default:
      return "idle";
  }
}

export default function ServiceDetailPage() {
  const apiClient = getApi();
  const params = useParams();
  const name = decodeURIComponent(params.name as string);
  const { data: svc, loading, error, refetch } = useApi<ServiceDetailItem>(
    () => apiClient.services.get(name)
  );
  const [restarting, setRestarting] = useState(false);

  const handleRestart = async () => {
    if (!confirm(`Restart ${name}? This will perform a rolling restart.`)) return;
    setRestarting(true);
    try {
      await apiClient.services.restart(name);
      setTimeout(refetch, 2000);
    } catch {
      // handled
    } finally {
      setRestarting(false);
    }
  };

  return (
    <Shell>
      <div className="space-y-6">
        <Link
          href="/services"
          className="inline-flex items-center gap-1.5 text-sm text-neutral-500 hover:text-neutral-300 transition-colors"
        >
          &larr; Back to services
        </Link>

        {loading ? (
          <div className="space-y-4">
            <Skeleton className="h-6 w-48" />
            <Skeleton className="h-4 w-32" />
            <div className="grid grid-cols-4 gap-4 mt-6">
              {[1, 2, 3, 4].map((i) => (
                <div key={i} className="rounded-lg border border-border bg-surface-100 p-4">
                  <Skeleton className="h-4 w-20 mb-2" />
                  <Skeleton className="h-6 w-16" />
                </div>
              ))}
            </div>
          </div>
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : !svc ? (
          <div className="rounded-xl border border-border bg-surface-100 py-12 text-center text-sm text-neutral-500">
            Service not found
          </div>
        ) : (
          <>
            {/* Header */}
            <div className="flex items-center justify-between">
              <div>
                <div className="flex items-center gap-3">
                  <h1 className="text-lg font-semibold text-white">{svc.name}</h1>
                  <StatusBadge status={statusToBadge(svc.status)} label={svc.status} />
                </div>
                <p className="mt-1 text-sm text-neutral-500">
                  {svc.kind} in <span className="font-mono text-neutral-400">{svc.namespace}</span>
                </p>
              </div>
              <div className="flex gap-2">
                <button
                  onClick={refetch}
                  className="flex items-center gap-1.5 rounded-lg border border-border bg-surface-100 px-3 py-1.5 text-sm text-neutral-400 hover:bg-surface-200 hover:text-white transition-colors"
                >
                  <RefreshCw className="h-3.5 w-3.5" />
                  Refresh
                </button>
                <button
                  onClick={handleRestart}
                  disabled={restarting}
                  className="flex items-center gap-1.5 rounded-lg bg-orange-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-orange-700 transition-colors disabled:opacity-50"
                >
                  <RotateCcw className="h-3.5 w-3.5" />
                  {restarting ? "Restarting..." : "Restart"}
                </button>
              </div>
            </div>

            {/* Stats */}
            <div className="grid grid-cols-4 gap-4">
              <div className="rounded-lg border border-border bg-surface-100 p-4">
                <p className="text-xs text-neutral-500 mb-1">Replicas</p>
                <p className={`text-lg font-semibold ${svc.readyReplicas < svc.totalReplicas ? "text-amber-400" : "text-emerald-400"}`}>
                  {svc.readyReplicas}/{svc.totalReplicas}
                </p>
              </div>
              <div className="rounded-lg border border-border bg-surface-100 p-4">
                <p className="text-xs text-neutral-500 mb-1">Restarts</p>
                <p className={`text-lg font-semibold ${svc.restarts > 0 ? "text-amber-400" : "text-white"}`}>
                  {svc.restarts}
                </p>
              </div>
              <div className="rounded-lg border border-border bg-surface-100 p-4">
                <p className="text-xs text-neutral-500 mb-1">Kind</p>
                <p className="text-lg font-semibold text-white">{svc.kind}</p>
              </div>
              <div className="rounded-lg border border-border bg-surface-100 p-4">
                <p className="text-xs text-neutral-500 mb-1">Namespace</p>
                <p className="text-lg font-semibold text-white font-mono text-sm">{svc.namespace}</p>
              </div>
            </div>

            {/* Pods */}
            <section>
              <h2 className="mb-3 text-sm font-medium text-white">Pods</h2>
              {svc.pods && svc.pods.length > 0 ? (
                <div className="overflow-hidden rounded-lg border border-border">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-border bg-surface-100">
                        <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Pod Name</th>
                        <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                        <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Restarts</th>
                        <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Age</th>
                      </tr>
                    </thead>
                    <tbody>
                      {svc.pods.map((pod) => (
                        <tr key={pod.name} className="border-b border-border last:border-0 transition-colors hover:bg-surface-200">
                          <td className="px-4 py-3 font-mono text-xs text-white">{pod.name}</td>
                          <td className="px-4 py-3">
                            <StatusBadge
                              status={pod.status === "Running" ? "healthy" : pod.status === "Pending" ? "warning" : "error"}
                              label={pod.status}
                            />
                          </td>
                          <td className="px-4 py-3 text-neutral-300">{pod.restarts}</td>
                          <td className="px-4 py-3 text-neutral-400">{pod.age}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <div className="rounded-lg border border-border bg-surface-100 py-6 text-center text-sm text-neutral-500">
                  No pod information available
                </div>
              )}
            </section>

            {/* Events */}
            <section>
              <h2 className="mb-3 text-sm font-medium text-white">Recent Events</h2>
              {svc.events && svc.events.length > 0 ? (
                <div className="space-y-2">
                  {svc.events.map((event, i) => (
                    <div key={i} className="rounded-lg border border-border bg-surface-100 px-4 py-3 text-sm text-neutral-300">
                      {typeof event === "string" ? event : JSON.stringify(event)}
                    </div>
                  ))}
                </div>
              ) : (
                <div className="rounded-lg border border-border bg-surface-100 py-6 text-center text-sm text-neutral-500">
                  No recent events
                </div>
              )}
            </section>
          </>
        )}
      </div>
    </Shell>
  );
}
