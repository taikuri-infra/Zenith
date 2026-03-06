"use client";

import { Shell } from "@/components/shell";
import { BuildLogViewer } from "@/components/build-log-viewer";
import { useApi } from "@/hooks/use-api";
import { getApi, isDemoMode } from "@/lib/get-api";
import { demoAggregatedLogs } from "@/lib/demo-api";
import { useState, useMemo } from "react";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { Search } from "lucide-react";

const LOG_LEVELS = [
  { value: "all", label: "All levels" },
  { value: "info", label: "Info" },
  { value: "warn", label: "Warning" },
  { value: "error", label: "Error" },
  { value: "build", label: "Build" },
  { value: "deploy", label: "Deploy" },
];

export default function LogsPage() {
  const { appsDeploy } = getApi();
  const [appFilter, setAppFilter] = useState<string>("all");
  const [levelFilter, setLevelFilter] = useState<string>("all");
  const [searchQuery, setSearchQuery] = useState("");

  const { data: deployData, loading } = useApi(() => appsDeploy.list(), []);
  const apps = deployData?.items ?? [];

  const allLogs = isDemoMode() ? demoAggregatedLogs : [];

  const filteredLogs = useMemo(() => {
    let logs = allLogs;

    // Filter by app
    if (appFilter !== "all") {
      logs = logs.filter((l) => l.message.includes(`[${appFilter}]`));
    }

    // Filter by level
    if (levelFilter !== "all") {
      logs = logs.filter((l) => l.level === levelFilter);
    }

    // Search in message
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      logs = logs.filter((l) => l.message.toLowerCase().includes(q));
    }

    return logs;
  }, [allLogs, appFilter, levelFilter, searchQuery]);

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
              <option key={app.id} value={app.name}>
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

          {/* Count */}
          <span className="text-xs text-neutral-600 shrink-0">
            {filteredLogs.length} / {allLogs.length}
          </span>
        </div>

        {/* Active filters */}
        {(appFilter !== "all" || levelFilter !== "all" || searchQuery.trim()) && (
          <div className="flex items-center gap-2 flex-wrap">
            {appFilter !== "all" && (
              <span className="inline-flex items-center gap-1.5 rounded-full bg-accent-500/10 px-2.5 py-1 text-xs text-accent-400">
                App: {appFilter}
                <button onClick={() => setAppFilter("all")} className="hover:text-white">&times;</button>
              </span>
            )}
            {levelFilter !== "all" && (
              <span className="inline-flex items-center gap-1.5 rounded-full bg-accent-500/10 px-2.5 py-1 text-xs text-accent-400">
                Level: {levelFilter}
                <button onClick={() => setLevelFilter("all")} className="hover:text-white">&times;</button>
              </span>
            )}
            {searchQuery.trim() && (
              <span className="inline-flex items-center gap-1.5 rounded-full bg-accent-500/10 px-2.5 py-1 text-xs text-accent-400">
                Search: &ldquo;{searchQuery}&rdquo;
                <button onClick={() => setSearchQuery("")} className="hover:text-white">&times;</button>
              </span>
            )}
            <button
              onClick={() => { setAppFilter("all"); setLevelFilter("all"); setSearchQuery(""); }}
              className="text-xs text-neutral-500 hover:text-white transition-colors"
            >
              Clear all
            </button>
          </div>
        )}

        {/* Log viewer */}
        <BuildLogViewer entries={filteredLogs} streaming={false} />
      </div>
    </Shell>
  );
}
