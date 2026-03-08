"use client";

import { Shell } from "@/components/shell";
import { BuildLogViewer } from "@/components/build-log-viewer";
import { useApi } from "@/hooks/use-api";
import { useProject } from "@/hooks/use-project";
import { getApi, isDemoMode } from "@/lib/get-api";
import { demoAggregatedLogs } from "@/lib/demo-api";
import { useState, useMemo, useEffect, useRef, useCallback } from "react";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { Search, Radio } from "lucide-react";
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

export default function LogsPage() {
  const { appsDeploy, monitoring } = getApi();
  const projectId = useProject();
  const [appFilter, setAppFilter] = useState<string>("all");
  const [levelFilter, setLevelFilter] = useState<string>("all");
  const [searchQuery, setSearchQuery] = useState("");
  const [streaming, setStreaming] = useState(false);
  const [streamEntries, setStreamEntries] = useState<Array<{ timestamp: string; level: string; message: string }>>([]);
  const eventSourceRef = useRef<EventSource | null>(null);

  const { data: deployData, loading } = useApi(
    () => appsDeploy.list(projectId || undefined),
    [projectId]
  );
  const apps = deployData?.items ?? [];

  // Fetch real logs from Loki when an app is selected
  const { data: logsData } = useApi(
    () => {
      if (isDemoMode() || appFilter === "all") return Promise.resolve(null);
      return monitoring.getLogs(appFilter, {
        level: levelFilter !== "all" ? levelFilter : undefined,
        search: searchQuery || undefined,
        limit: 200,
        since: "1h",
      });
    },
    [appFilter, levelFilter, searchQuery]
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
      if (appFilter !== "all") {
        logs = logs.filter((l) => l.message.includes(`[${appFilter}]`));
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
  }, [allLogs, appFilter, levelFilter, searchQuery, streaming, streamEntries]);

  // SSE streaming toggle
  const toggleStreaming = useCallback(() => {
    if (streaming) {
      eventSourceRef.current?.close();
      eventSourceRef.current = null;
      setStreaming(false);
      return;
    }

    if (appFilter === "all" || isDemoMode()) return;

    const token = getAccessToken();
    const url = `${API_BASE_URL}/api/v1/apps/${appFilter}/logs/stream?token=${token}`;
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
  }, [streaming, appFilter]);

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
    if (urlApp) setAppFilter(urlApp);
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
            Aggregated logs across all apps
          </p>
        </div>

        {/* Filters */}
        <div className="flex items-center gap-3">
          {/* Search */}
          <div className="relative flex-1">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-500" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search logs..."
              className="w-full rounded-lg border border-border bg-surface-100 py-1.5 pl-9 pr-3 text-sm text-white placeholder:text-neutral-500 focus:border-accent-500 focus:outline-none"
            />
          </div>

          {/* App filter */}
          <select
            value={appFilter}
            onChange={(e) => setAppFilter(e.target.value)}
            className="rounded-lg border border-border bg-surface-100 px-3 py-1.5 text-sm text-neutral-400 focus:border-accent-500 focus:outline-none"
          >
            <option value="all">All apps</option>
            {apps.map((app) => (
              <option key={app.id} value={isDemoMode() ? app.name : app.id}>
                {app.name}
              </option>
            ))}
          </select>

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

          {/* Stream toggle */}
          {appFilter !== "all" && !isDemoMode() && (
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
        {(appFilter !== "all" ||
          levelFilter !== "all" ||
          searchQuery.trim()) && (
          <div className="flex items-center gap-2 flex-wrap">
            {appFilter !== "all" && (
              <span className="inline-flex items-center gap-1.5 rounded-full bg-accent-500/10 px-2.5 py-1 text-xs text-accent-400">
                App: {apps.find((a) => (isDemoMode() ? a.name : a.id) === appFilter)?.name || appFilter}
                <button
                  onClick={() => setAppFilter("all")}
                  className="hover:text-white"
                >
                  &times;
                </button>
              </span>
            )}
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
                setAppFilter("all");
                setLevelFilter("all");
                setSearchQuery("");
              }}
              className="text-xs text-neutral-500 hover:text-white transition-colors"
            >
              Clear all
            </button>
          </div>
        )}

        {/* Log viewer */}
        <BuildLogViewer entries={filteredLogs} streaming={streaming} />
      </div>
    </Shell>
  );
}
