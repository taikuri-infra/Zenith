"use client";

import { Shell } from "@/components/shell";
import { BuildLogViewer } from "@/components/build-log-viewer";
import { AIErrorAnalysis } from "@/components/ai-error-analysis";
import { useApi } from "@/hooks/use-api";
import { useProject } from "@/hooks/use-project";
import { getApi, isDemoMode } from "@/lib/get-api";
import { demoAggregatedLogs } from "@/lib/demo-api";
import { useState, useMemo, useEffect, useRef, useCallback } from "react";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { Search, Radio, Clock, Check } from "lucide-react";
import { getAccessToken } from "@/lib/api";
import { API_BASE_URL } from "@/lib/runtime-env";

const LOG_LEVELS = [
  { value: "all", label: "All levels" },
  { value: "info", label: "Info" },
  { value: "warn", label: "Warning" },
  { value: "error", label: "Error" },
  { value: "build", label: "Build" },
  { value: "deploy", label: "Deploy" },
];

const TIME_RANGES = [
  { value: "1h", label: "Last 1 hour" },
  { value: "6h", label: "Last 6 hours" },
  { value: "24h", label: "Last 24 hours" },
  { value: "7d", label: "Last 7 days" },
];

export default function LogsPage() {
  const { appsDeploy, monitoring } = getApi();
  const projectId = useProject();
  const [selectedApps, setSelectedApps] = useState<string[]>([]);
  const [levelFilter, setLevelFilter] = useState<string>("all");
  const [searchQuery, setSearchQuery] = useState("");
  const [timeRange, setTimeRange] = useState("1h");
  const [streaming, setStreaming] = useState(false);
  const [streamEntries, setStreamEntries] = useState<Array<{ timestamp: string; level: string; message: string }>>([]);
  const [showAppDropdown, setShowAppDropdown] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const eventSourceRef = useRef<EventSource | null>(null);

  const { data: deployData, loading } = useApi(
    () => appsDeploy.list(projectId || undefined),
    [projectId]
  );
  const apps = deployData?.items ?? [];

  // Fetch logs: aggregated multi-app endpoint or single-app
  const { data: logsData } = useApi(
    () => {
      if (isDemoMode()) return Promise.resolve(null);
      if (selectedApps.length === 0) return Promise.resolve(null);

      const params = {
        level: levelFilter !== "all" ? levelFilter : undefined,
        search: searchQuery || undefined,
        limit: 200,
        since: timeRange,
      };

      // Use aggregated endpoint for multi-app, single-app endpoint for one
      if (selectedApps.length === 1) {
        return monitoring.getLogs(selectedApps[0], params);
      }
      return monitoring.getAggregatedLogs(selectedApps, params);
    },
    [selectedApps.join(","), levelFilter, searchQuery, timeRange]
  );

  // Build the combined log list
  const allLogs = useMemo(() => {
    if (isDemoMode()) return demoAggregatedLogs;
    if (logsData?.entries) {
      return logsData.entries.map((e) => ({
        timestamp: e.timestamp,
        level: e.level || "info",
        message: e.line,
      }));
    }
    return [];
  }, [logsData]);

  const filteredLogs = useMemo(() => {
    let logs = streaming && streamEntries.length > 0 ? streamEntries : allLogs;

    // In demo mode, apply client-side filters
    if (isDemoMode()) {
      if (selectedApps.length > 0) {
        logs = logs.filter((l) =>
          selectedApps.some((app) => l.message.includes(`[${app}]`))
        );
      }
      if (levelFilter !== "all") {
        logs = logs.filter((l) => l.level === levelFilter);
      }
      if (searchQuery.trim()) {
        const q = searchQuery.toLowerCase();
        logs = logs.filter((l) => l.message.toLowerCase().includes(q));
      }
    }

    return logs;
  }, [allLogs, selectedApps, levelFilter, searchQuery, streaming, streamEntries]);

  // Toggle app selection
  const toggleApp = useCallback((appId: string) => {
    setSelectedApps((prev) =>
      prev.includes(appId)
        ? prev.filter((id) => id !== appId)
        : prev.length < 10
          ? [...prev, appId]
          : prev
    );
  }, []);

  // Select / deselect all
  const toggleAllApps = useCallback(() => {
    if (selectedApps.length === apps.length) {
      setSelectedApps([]);
    } else {
      setSelectedApps(apps.map((a) => (isDemoMode() ? a.name : a.id)));
    }
  }, [selectedApps.length, apps]);

  // SSE streaming toggle (works for single app only)
  const toggleStreaming = useCallback(() => {
    if (streaming) {
      eventSourceRef.current?.close();
      eventSourceRef.current = null;
      setStreaming(false);
      return;
    }

    if (selectedApps.length !== 1 || isDemoMode()) return;

    const token = getAccessToken();
    const url = `${API_BASE_URL}/api/v1/apps/${selectedApps[0]}/logs/stream?token=${token}`;
    const es = new EventSource(url);
    eventSourceRef.current = es;
    setStreaming(true);
    setStreamEntries([]);

    es.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        setStreamEntries((prev) => [
          ...prev,
          {
            timestamp: data.timestamp,
            level: data.level || "info",
            message: data.line,
          },
        ]);
      } catch {
        // ignore
      }
    };

    es.addEventListener("done", () => {
      es.close();
      setStreaming(false);
    });

    es.onerror = () => {
      es.close();
      setStreaming(false);
    };
  }, [streaming, selectedApps]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      eventSourceRef.current?.close();
    };
  }, []);

  // Read appId from URL query params
  useEffect(() => {
    if (typeof window === "undefined") return;
    const params = new URLSearchParams(window.location.search);
    const urlApp = params.get("app");
    if (urlApp) setSelectedApps([urlApp]);
  }, []);

  // Close dropdown on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setShowAppDropdown(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  if (loading) {
    return (
      <Shell>
        <PageWithTableSkeleton cols={3} rows={5} />
      </Shell>
    );
  }

  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">Logs</h1>
          <p className="text-sm text-neutral-500">
            Aggregated logs across your services
          </p>
        </div>

        {/* Filters */}
        <div className="flex items-center gap-3 flex-wrap">
          {/* Search */}
          <div className="relative flex-1 min-w-[200px]">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-500" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search logs..."
              className="w-full rounded-lg border border-border bg-surface-100 py-1.5 pl-9 pr-3 text-sm text-white placeholder:text-neutral-500 focus:border-accent-500 focus:outline-none"
            />
          </div>

          {/* Multi-select app dropdown */}
          <div className="relative" ref={dropdownRef}>
            <button
              onClick={() => setShowAppDropdown(!showAppDropdown)}
              className="flex items-center gap-1.5 rounded-lg border border-border bg-surface-100 px-3 py-1.5 text-sm text-neutral-400 hover:text-white transition-colors"
            >
              {selectedApps.length === 0
                ? "Select services"
                : `${selectedApps.length} service${selectedApps.length > 1 ? "s" : ""}`}
              <svg className="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
              </svg>
            </button>

            {showAppDropdown && (
              <div className="absolute left-0 top-full z-50 mt-1 w-56 rounded-lg border border-border bg-surface-200 py-1 shadow-xl">
                {/* Select all */}
                <button
                  onClick={toggleAllApps}
                  className="flex w-full items-center gap-2 px-3 py-1.5 text-xs text-neutral-400 hover:bg-surface-100 hover:text-white"
                >
                  <div className={`flex h-4 w-4 items-center justify-center rounded border ${
                    selectedApps.length === apps.length
                      ? "border-accent-500 bg-accent-500"
                      : "border-neutral-600"
                  }`}>
                    {selectedApps.length === apps.length && <Check className="h-3 w-3 text-white" />}
                  </div>
                  {selectedApps.length === apps.length ? "Deselect all" : "Select all"}
                </button>
                <div className="my-1 border-t border-border" />
                {apps.map((app) => {
                  const appId = isDemoMode() ? app.name : app.id;
                  const checked = selectedApps.includes(appId);
                  return (
                    <button
                      key={app.id}
                      onClick={() => toggleApp(appId)}
                      className="flex w-full items-center gap-2 px-3 py-1.5 text-sm text-neutral-300 hover:bg-surface-100 hover:text-white"
                    >
                      <div className={`flex h-4 w-4 items-center justify-center rounded border ${
                        checked
                          ? "border-accent-500 bg-accent-500"
                          : "border-neutral-600"
                      }`}>
                        {checked && <Check className="h-3 w-3 text-white" />}
                      </div>
                      {app.name}
                    </button>
                  );
                })}
              </div>
            )}
          </div>

          {/* Level filter */}
          <select
            value={levelFilter}
            onChange={(e) => setLevelFilter(e.target.value)}
            className="rounded-lg border border-border bg-surface-100 px-3 py-1.5 text-sm text-neutral-400 focus:border-accent-500 focus:outline-none"
          >
            {LOG_LEVELS.map((level) => (
              <option key={level.value} value={level.value}>
                {level.label}
              </option>
            ))}
          </select>

          {/* Time range */}
          <div className="flex items-center gap-1">
            <Clock className="h-3.5 w-3.5 text-neutral-500" />
            <select
              value={timeRange}
              onChange={(e) => setTimeRange(e.target.value)}
              className="rounded-lg border border-border bg-surface-100 px-3 py-1.5 text-sm text-neutral-400 focus:border-accent-500 focus:outline-none"
            >
              {TIME_RANGES.map((r) => (
                <option key={r.value} value={r.value}>
                  {r.label}
                </option>
              ))}
            </select>
          </div>

          {/* Stream toggle (single app only) */}
          {selectedApps.length === 1 && !isDemoMode() && (
            <button
              onClick={toggleStreaming}
              className={`flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-xs transition-colors ${
                streaming
                  ? "border-emerald-500/50 bg-emerald-500/10 text-emerald-400"
                  : "border-border bg-surface-100 text-neutral-400 hover:text-white"
              }`}
            >
              <Radio className={`h-3 w-3 ${streaming ? "animate-pulse" : ""}`} />
              {streaming ? "Live" : "Stream"}
            </button>
          )}

          {/* Count */}
          <span className="text-xs text-neutral-600 shrink-0">
            {filteredLogs.length} entries
          </span>
        </div>

        {/* Active filters */}
        {(selectedApps.length > 0 ||
          levelFilter !== "all" ||
          searchQuery.trim()) && (
          <div className="flex items-center gap-2 flex-wrap">
            {selectedApps.map((appId) => (
              <span
                key={appId}
                className="inline-flex items-center gap-1.5 rounded-full bg-accent-500/10 px-2.5 py-1 text-xs text-accent-400"
              >
                {apps.find((a) => (isDemoMode() ? a.name : a.id) === appId)?.name || appId}
                <button
                  onClick={() => setSelectedApps((prev) => prev.filter((id) => id !== appId))}
                  className="hover:text-white"
                >
                  &times;
                </button>
              </span>
            ))}
            {levelFilter !== "all" && (
              <span className="inline-flex items-center gap-1.5 rounded-full bg-accent-500/10 px-2.5 py-1 text-xs text-accent-400">
                Level: {levelFilter}
                <button
                  onClick={() => setLevelFilter("all")}
                  className="hover:text-white"
                >
                  &times;
                </button>
              </span>
            )}
            {searchQuery.trim() && (
              <span className="inline-flex items-center gap-1.5 rounded-full bg-accent-500/10 px-2.5 py-1 text-xs text-accent-400">
                Search: &ldquo;{searchQuery}&rdquo;
                <button
                  onClick={() => setSearchQuery("")}
                  className="hover:text-white"
                >
                  &times;
                </button>
              </span>
            )}
            <button
              onClick={() => {
                setSelectedApps([]);
                setLevelFilter("all");
                setSearchQuery("");
              }}
              className="text-xs text-neutral-500 hover:text-white transition-colors"
            >
              Clear all
            </button>
          </div>
        )}

        {/* AI Error Analysis (single app) */}
        {selectedApps.length === 1 && (
          <AIErrorAnalysis appId={selectedApps[0]} />
        )}

        {/* Empty state */}
        {selectedApps.length === 0 && !isDemoMode() && (
          <div className="flex flex-col items-center justify-center rounded-lg border border-border bg-surface-100 py-16 text-center">
            <Search className="mb-3 h-8 w-8 text-neutral-600" />
            <p className="text-sm text-neutral-400">Select one or more services to view logs</p>
            <p className="mt-1 text-xs text-neutral-600">You can select up to 10 services at once</p>
          </div>
        )}

        {/* Log viewer */}
        {(selectedApps.length > 0 || isDemoMode()) && (
          <BuildLogViewer entries={filteredLogs} streaming={streaming} />
        )}
      </div>
    </Shell>
  );
}
