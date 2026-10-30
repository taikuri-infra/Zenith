"use client";
import { useCallback, useEffect, useRef, useState } from "react";
import type { DeployApp, PodStatus } from "@/lib/api";
import { monitoring as realMonitoring } from "@/lib/api";
import { getApi } from "@/lib/get-api";
import { ErrorState } from "@/components/error-state";
import { TableSkeleton } from "@/components/loading-skeleton";

const POLL_MS = 10_000;

export function ContainersCard({ app }: { app: DeployApp }) {
  const [pods, setPods] = useState<PodStatus[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const alive = useRef(true);

  const load = useCallback(async () => {
    try {
      const api = getApi().monitoring ?? realMonitoring;
      const res = await api.getPods(app.id);
      if (!alive.current) return;
      setPods(res?.pods ?? []);
      setError(null);
    } catch (e) {
      if (!alive.current) return;
      setError(e instanceof Error ? e.message : "Failed to load containers");
    }
  }, [app.id]);

  useEffect(() => {
    alive.current = true;
    load();
    const t = setInterval(load, POLL_MS);
    return () => {
      alive.current = false;
      clearInterval(t);
    };
  }, [load]);

  return (
    <section className="md:col-span-2 rounded-lg border border-border bg-surface-100 p-5">
      <h2 className="mb-4 text-sm font-medium text-white">Containers</h2>
      {pods === null && !error ? (
        <TableSkeleton cols={5} rows={2} />
      ) : error && pods === null ? (
        <ErrorState message={error} onRetry={load} />
      ) : pods && pods.length === 0 ? (
        <div className="text-xs text-neutral-600">No containers are running for this app.</div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead>
              <tr className="text-neutral-500">
                <th className="pb-2 pr-4 font-medium">Name</th>
                <th className="pb-2 pr-4 font-medium">Status</th>
                <th className="pb-2 pr-4 font-medium">Ready</th>
                <th className="pb-2 pr-4 font-medium">Restarts</th>
                <th className="pb-2 pr-4 font-medium">Started</th>
                <th className="pb-2 font-medium">Reason</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {(pods ?? []).map((p) => (
                <tr key={p.name}>
                  <td className="py-2 pr-4 font-mono text-neutral-300">{p.name}</td>
                  <td className="py-2 pr-4 text-neutral-300">{p.status}</td>
                  <td className={`py-2 pr-4 ${p.ready ? "text-emerald-400" : "text-amber-400"}`}>{p.ready ? "Yes" : "No"}</td>
                  <td className="py-2 pr-4 text-neutral-300">{p.restarts}</td>
                  <td className="py-2 pr-4 text-neutral-400">{p.started_at ? new Date(p.started_at).toLocaleString() : "—"}</td>
                  <td className="py-2 text-neutral-400">{p.status_reason || "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
