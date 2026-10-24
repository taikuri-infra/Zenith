"use client";

import { useEffect, useState } from "react";
import type { DeployApp, MetricsOverview } from "@/lib/api";
import { getApi } from "@/lib/get-api";

/**
 * Live CPU/memory per running app. Failures are ignored per app: a missing
 * metric just leaves that row without numbers. Refreshes every 15 s.
 */
export function useAppMetrics(apps: DeployApp[]): Record<string, MetricsOverview> {
  const [metrics, setMetrics] = useState<Record<string, MetricsOverview>>({});
  const ids = apps.filter((a) => a.status === "running").map((a) => a.id).join(",");

  useEffect(() => {
    if (!ids) {
      setMetrics({});
      return;
    }
    let cancelled = false;
    const { monitoring } = getApi();
    const load = async () => {
      const list = ids.split(",");
      const results = await Promise.allSettled(list.map((id) => monitoring.getOverview(id)));
      if (cancelled) return;
      const next: Record<string, MetricsOverview> = {};
      results.forEach((r, i) => {
        if (r.status === "fulfilled" && r.value) next[list[i]] = r.value;
      });
      setMetrics(next);
    };
    load();
    const timer = setInterval(load, 15000);
    return () => {
      cancelled = true;
      clearInterval(timer);
    };
  }, [ids]);

  return metrics;
}
