"use client";

import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { useApi } from "@/hooks/use-api";
import { getApi } from "@/lib/get-api";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { useParams } from "next/navigation";
import { useState, useMemo } from "react";
import { ArrowLeft, RefreshCw } from "lucide-react";
import Link from "next/link";
import type { TimeSeriesPoint, MonitoringLogEntry } from "@/lib/api";

const TIME_RANGES = ["1h", "6h", "24h", "7d"] as const;

const levelColors: Record<string, string> = {
  info: "text-emerald-400",
  warn: "text-amber-400",
  error: "text-red-400",
  debug: "text-neutral-500",
};

const podStatusColors: Record<string, string> = {
  Running: "bg-emerald-400/10 text-emerald-400",
  Pending: "bg-amber-400/10 text-amber-400",
  Failed: "bg-red-400/10 text-red-400",
  Succeeded: "bg-blue-400/10 text-blue-400",
  Unknown: "bg-neutral-400/10 text-neutral-400",
};

export default function AppMonitoringPage() {
  const params = useParams();
  const appId = params.appId as string;
  const { monitoring, appsDeploy } = getApi();
  const [range, setRange] = useState<string>("1h");

  const { data: appData } = useApi(
    () => appsDeploy.list(),
    []
  );
  const app = appData?.items?.find((a: { id: string }) => a.id === appId);

  const { data: overview, loading: overviewLoading, refetch: refetchOverview } = useApi(
    () => monitoring.getOverview(appId),
    [appId]
  );

  const { data: cpuData } = useApi(
    () => monitoring.getTimeSeries(appId, "cpu", range),
    [appId, range]
  );

  const { data: memData } = useApi(
    () => monitoring.getTimeSeries(appId, "memory", range),
    [appId, range]
  );

  const { data: reqData } = useApi(
    () => monitoring.getTimeSeries(appId, "requests", range),
    [appId, range]
  );

  const { data: latData } = useApi(
    () => monitoring.getTimeSeries(appId, "latency", range),
    [appId, range]
  );

  const { data: logsData } = useApi(
    () => monitoring.getLogs(appId, { limit: 20, since: range }),
    [appId, range]
  );

  const { data: podsData } = useApi(
    () => monitoring.getPods(appId),
    [appId]
  );

  if (overviewLoading && !overview) {
    return (
      <Shell>
        <PageWithTableSkeleton cols={4} rows={6} />
      </Shell>
    );
  }

  return (
    <Shell>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Link
              href="/monitoring"
              className="rounded-lg border border-border p-1.5 text-neutral-400 transition-colors hover:border-border-hover hover:text-white"
            >
              <ArrowLeft className="h-4 w-4" />
            </Link>
            <div>
              <h1 className="text-lg font-semibold text-white">
                {app?.name || "App"} Monitoring
              </h1>
              <p className="text-sm text-neutral-500">
                Metrics, logs, and pod health
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => refetchOverview()}
              className="rounded-lg border border-border p-1.5 text-neutral-400 transition-colors hover:border-border-hover hover:text-white"
            >
              <RefreshCw className="h-4 w-4" />
            </button>
            <div className="flex rounded-lg border border-border bg-surface-100">
              {TIME_RANGES.map((r) => (
                <button
                  key={r}
                  onClick={() => setRange(r)}
                  className={`px-3 py-1.5 text-xs transition-colors ${
                    range === r
                      ? "bg-accent-500/15 text-accent-400"
                      : "text-neutral-400 hover:text-white"
                  }`}
                >
                  {r}
                </button>
              ))}
            </div>
          </div>
        </div>

        {/* Stat Cards */}
        {overview && (
          <div className="grid grid-cols-6 gap-3">
            <StatCard
              label="CPU"
              value={`${overview.cpu_percent.toFixed(1)}%`}
              sub="total usage"
            />
            <StatCard
              label="Memory"
              value={`${overview.memory_mb.toFixed(0)}MB`}
              sub={`${overview.memory_percent.toFixed(1)}%`}
            />
            <StatCard
              label="Requests"
              value={`${overview.request_rate.toFixed(1)}/s`}
              sub="req rate"
            />
            <StatCard
              label="Error Rate"
              value={`${overview.error_rate.toFixed(2)}%`}
              sub="5xx"
            />
            <StatCard
              label="P95 Latency"
              value={`${overview.p95_latency_ms.toFixed(0)}ms`}
              sub="response time"
            />
            <StatCard
              label="Pods"
              value={overview.pod_count}
              sub="running"
            />
          </div>
        )}

        {/* Charts Grid */}
        <div className="grid grid-cols-2 gap-4">
          <ChartPanel
            title="CPU Usage (%)"
            points={cpuData?.points}
            color="#818cf8"
          />
          <ChartPanel
            title="Memory Usage (MB)"
            points={memData?.points}
            color="#34d399"
          />
          <ChartPanel
            title="Requests / min"
            points={reqData?.points}
            color="#60a5fa"
            type="bar"
          />
          <ChartPanel
            title="P95 Latency (ms)"
            points={latData?.points}
            color="#fbbf24"
          />
        </div>

        {/* Logs Preview */}
        <section>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-sm font-medium text-white">Recent Logs</h2>
            <Link
              href={`/logs?app=${appId}`}
              className="text-xs text-accent-400 hover:text-accent-300"
            >
              View all logs
            </Link>
          </div>
          <LogsPreview entries={logsData?.entries} />
        </section>

        {/* Pods Table */}
        <section>
          <h2 className="mb-3 text-sm font-medium text-white">Pods</h2>
          <PodsTable pods={podsData?.pods} />
        </section>
      </div>
    </Shell>
  );
}

function ChartPanel({
  title,
  points,
  color,
  type = "area",
}: {
  title: string;
  points?: TimeSeriesPoint[];
  color: string;
  type?: "area" | "bar";
}) {
  const svgContent = useMemo(() => {
    if (!points || points.length === 0) return null;

    const values = points.map((p) => p.value);
    const max = Math.max(...values, 1);
    const min = Math.min(...values, 0);
    const range = max - min || 1;
    const w = 400;
    const h = 120;
    const pad = 4;

    if (type === "bar") {
      const barWidth = Math.max((w - pad * 2) / values.length - 2, 2);
      return (
        <svg viewBox={`0 0 ${w} ${h}`} className="h-full w-full">
          {values.map((v, i) => {
            const barH = ((v - min) / range) * (h - pad * 2);
            const x = pad + (i * (w - pad * 2)) / values.length;
            const y = h - pad - barH;
            return (
              <rect
                key={i}
                x={x}
                y={y}
                width={barWidth}
                height={barH}
                fill={color}
                opacity={0.7}
                rx={1}
              />
            );
          })}
        </svg>
      );
    }

    // Area chart
    const pathPoints = values.map((v, i) => {
      const x = pad + (i / (values.length - 1)) * (w - pad * 2);
      const y = h - pad - ((v - min) / range) * (h - pad * 2);
      return `${x},${y}`;
    });

    const linePath = `M${pathPoints.join(" L")}`;
    const areaPath = `${linePath} L${pad + ((values.length - 1) / (values.length - 1)) * (w - pad * 2)},${h - pad} L${pad},${h - pad} Z`;

    return (
      <svg viewBox={`0 0 ${w} ${h}`} className="h-full w-full">
        <defs>
          <linearGradient id={`grad-${title}`} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={color} stopOpacity={0.3} />
            <stop offset="100%" stopColor={color} stopOpacity={0} />
          </linearGradient>
        </defs>
        <path d={areaPath} fill={`url(#grad-${title})`} />
        <path d={linePath} fill="none" stroke={color} strokeWidth={2} />
      </svg>
    );
  }, [points, color, type, title]);

  return (
    <div className="rounded-lg border border-border bg-surface-100 p-4">
      <p className="mb-2 text-xs font-medium text-neutral-400">{title}</p>
      <div className="h-[120px]">
        {svgContent ? (
          svgContent
        ) : (
          <div className="flex h-full items-center justify-center">
            <span className="text-xs text-neutral-600">No data</span>
          </div>
        )}
      </div>
    </div>
  );
}

function LogsPreview({ entries }: { entries?: MonitoringLogEntry[] }) {
  if (!entries || entries.length === 0) {
    return (
      <div className="rounded-lg bg-[#0d1117] p-4">
        <p className="text-center text-xs text-neutral-600">No log entries</p>
      </div>
    );
  }

  return (
    <div className="overflow-x-auto rounded-lg bg-[#0d1117] p-4">
      <div className="space-y-0.5">
        {entries.map((entry, i) => {
          const ts = new Date(entry.timestamp);
          const time = ts.toLocaleTimeString("en-US", { hour12: false });
          const lvl = entry.level || "info";
          const color = levelColors[lvl] || "text-neutral-400";
          return (
            <div key={i} className="flex gap-2 font-mono text-xs leading-5">
              <span className="flex-shrink-0 text-neutral-600">{time}</span>
              <span
                className={`w-12 flex-shrink-0 font-semibold uppercase ${color}`}
              >
                {lvl}
              </span>
              <span className="text-neutral-400">{entry.line}</span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function PodsTable({
  pods,
}: {
  pods?: Array<{
    name: string;
    status: string;
    ready: boolean;
    restarts: number;
    cpu_millicores: number;
    memory_mb: number;
    started_at: string;
  }>;
}) {
  if (!pods || pods.length === 0) {
    return (
      <div className="rounded-lg border border-border bg-surface-100 p-8 text-center">
        <p className="text-xs text-neutral-600">No pods found</p>
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-lg border border-border">
      <table className="w-full text-sm">
        <thead>
          <tr className="bg-surface-100">
            <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
              Name
            </th>
            <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
              Status
            </th>
            <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
              Restarts
            </th>
            <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
              CPU
            </th>
            <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
              Memory
            </th>
            <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
              Started
            </th>
          </tr>
        </thead>
        <tbody>
          {pods.map((pod) => {
            const statusStyle =
              podStatusColors[pod.status] || podStatusColors.Unknown;
            const started = new Date(pod.started_at);
            const ago = formatTimeAgo(started);
            return (
              <tr
                key={pod.name}
                className="border-t border-border transition-colors hover:bg-surface-200"
              >
                <td className="px-4 py-2.5 font-mono text-xs text-white">
                  {pod.name}
                </td>
                <td className="px-4 py-2.5">
                  <span
                    className={`inline-flex rounded-full px-2 py-0.5 text-[10px] font-medium ${statusStyle}`}
                  >
                    {pod.status}
                  </span>
                </td>
                <td className="px-4 py-2.5 font-mono text-xs text-neutral-400">
                  {pod.restarts}
                </td>
                <td className="px-4 py-2.5 font-mono text-xs text-neutral-400">
                  {pod.cpu_millicores}m
                </td>
                <td className="px-4 py-2.5 font-mono text-xs text-neutral-400">
                  {pod.memory_mb.toFixed(1)}MB
                </td>
                <td className="px-4 py-2.5 text-xs text-neutral-500">
                  {ago}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function formatTimeAgo(date: Date): string {
  const seconds = Math.floor((Date.now() - date.getTime()) / 1000);
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}
