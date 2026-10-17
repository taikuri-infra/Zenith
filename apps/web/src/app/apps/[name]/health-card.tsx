"use client";
import { Heart } from "lucide-react";
import type { DeployApp } from "@/lib/api";

/** Health-check status for web apps; renders nothing without health_status. */
export function HealthCard({ app }: { app: DeployApp }) {
  const hs = app.health_status;
  if (!hs) return null;
  const hc = app.health_check;

  return (
    <div className="rounded-lg border border-border bg-surface-100 p-5">
      <div className="mb-4 flex items-center gap-2">
        <Heart className="h-4 w-4 text-neutral-400" />
        <h3 className="text-sm font-medium text-white">Health</h3>
      </div>
      <div className="grid grid-cols-2 gap-4">
        <div>
          <p className="mb-1 text-xs text-neutral-500">Status</p>
          <div className="flex items-center gap-2">
            <span
              className={`inline-block h-2 w-2 rounded-full ${
                hs.status === "healthy" ? "bg-emerald-400" : hs.status === "unhealthy" ? "bg-red-400" : "bg-neutral-500"
              }`}
            />
            <span className="text-sm font-medium capitalize text-white">{hs.status}</span>
          </div>
        </div>
        <div>
          <p className="mb-1 text-xs text-neutral-500">Uptime</p>
          <p className="text-sm font-medium text-white">{hs.uptime_percent}%</p>
        </div>
        <div>
          <p className="mb-1 text-xs text-neutral-500">Response Time</p>
          <p className="text-sm font-medium text-white">{hs.response_time_ms}ms</p>
        </div>
        <div>
          <p className="mb-1 text-xs text-neutral-500">Last Check</p>
          <p className="text-sm font-medium text-white">{new Date(hs.last_check).toLocaleTimeString()}</p>
        </div>
      </div>
      {hc && (
        <div className="mt-4 rounded-md bg-surface-200 px-3 py-2 text-xs text-neutral-500">
          <span className="text-neutral-400">Path:</span> <span className="font-mono">{hc.path}</span> &middot;{" "}
          <span className="text-neutral-400">Interval:</span> {hc.interval_seconds}s &middot;{" "}
          <span className="text-neutral-400">Timeout:</span> {hc.timeout_seconds}s
        </div>
      )}
    </div>
  );
}
