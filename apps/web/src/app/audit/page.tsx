"use client";

import { Shell } from "@/components/shell";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { useApi } from "@/hooks/use-api";
import { type AuditEntry } from "@/lib/api";
import { getApi } from "@/lib/get-api";
import Link from "next/link";
import { useState, useMemo, useCallback } from "react";
import {
  FileText,
  Download,
  Search,
  ChevronLeft,
  ChevronRight,
  Clock,
} from "lucide-react";

const PAGE_SIZE = 50;

const actionOptions = [
  { value: "", label: "All Actions" },
  { value: "deploy", label: "Deploy" },
  { value: "create", label: "Create" },
  { value: "delete", label: "Delete" },
  { value: "update", label: "Update" },
  { value: "login", label: "Login" },
  { value: "logout", label: "Logout" },
  { value: "invite", label: "Invite" },
  { value: "scale", label: "Scale" },
];

function actionBadgeColor(action: string): string {
  const lower = action.toLowerCase();
  if (lower.includes("deploy")) return "bg-blue-500/15 text-blue-400";
  if (lower.includes("create") || lower.includes("add"))
    return "bg-green-500/15 text-green-400";
  if (lower.includes("delete") || lower.includes("remove"))
    return "bg-red-500/15 text-red-400";
  if (lower.includes("update") || lower.includes("edit") || lower.includes("modify"))
    return "bg-yellow-500/15 text-yellow-400";
  if (lower.includes("login") || lower.includes("auth"))
    return "bg-purple-500/15 text-purple-400";
  if (lower.includes("logout")) return "bg-neutral-500/15 text-neutral-400";
  if (lower.includes("invite")) return "bg-pink-500/15 text-pink-400";
  if (lower.includes("scale")) return "bg-cyan-500/15 text-cyan-400";
  return "bg-neutral-500/15 text-neutral-400";
}

function relativeTime(dateStr: string): string {
  const now = Date.now();
  const then = new Date(dateStr).getTime();
  const diffMs = now - then;
  const diffSec = Math.floor(diffMs / 1000);
  const diffMin = Math.floor(diffSec / 60);
  const diffHr = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHr / 24);

  if (diffSec < 60) return "just now";
  if (diffMin < 60) return `${diffMin} minute${diffMin !== 1 ? "s" : ""} ago`;
  if (diffHr < 24) return `${diffHr} hour${diffHr !== 1 ? "s" : ""} ago`;
  if (diffDay < 30) return `${diffDay} day${diffDay !== 1 ? "s" : ""} ago`;
  return new Date(dateStr).toLocaleDateString();
}

function fullTimestamp(dateStr: string): string {
  return new Date(dateStr).toLocaleString();
}

export default function AuditPage() {
  const { audit, userPlan } = getApi();

  const [page, setPage] = useState(0);
  const [searchInput, setSearchInput] = useState("");
  const [search, setSearch] = useState("");
  const [actionFilter, setActionFilter] = useState("");
  const [exporting, setExporting] = useState<"csv" | "json" | null>(null);

  const { data: planData, loading: planLoading } = useApi(
    () => userPlan.get(),
    []
  );

  const {
    data: auditData,
    loading,
    error,
    refetch,
  } = useApi(
    () =>
      audit.list({
        limit: PAGE_SIZE,
        offset: page * PAGE_SIZE,
        action: actionFilter || undefined,
        search: search || undefined,
      }),
    [page, actionFilter, search]
  );

  const tier = planData?.tier ?? "free";
  const isAllowed = tier === "business" || tier === "enterprise";

  const entries: AuditEntry[] = useMemo(
    () => auditData?.items ?? [],
    [auditData]
  );
  const total = auditData?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  const handleSearch = useCallback(() => {
    setPage(0);
    setSearch(searchInput);
  }, [searchInput]);

  const handleSearchKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter") {
        handleSearch();
      }
    },
    [handleSearch]
  );

  const handleActionFilterChange = useCallback(
    (e: React.ChangeEvent<HTMLSelectElement>) => {
      setPage(0);
      setActionFilter(e.target.value);
    },
    []
  );

  const handleExportCSV = useCallback(async () => {
    if (exporting) return;
    setExporting("csv");
    try {
      const csvText = await audit.exportCSV({
        action: actionFilter || undefined,
        limit: 10000,
      });
      const blob = new Blob([csvText as string], { type: "text/csv" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `audit-log-${new Date().toISOString().slice(0, 10)}.csv`;
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      // silent — user can retry
    } finally {
      setExporting(null);
    }
  }, [audit, actionFilter, exporting]);

  const handleExportJSON = useCallback(async () => {
    if (exporting) return;
    setExporting("json");
    try {
      const jsonData = await audit.exportJSON({
        action: actionFilter || undefined,
        limit: 10000,
      });
      const blob = new Blob([JSON.stringify(jsonData, null, 2)], {
        type: "application/json",
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `audit-log-${new Date().toISOString().slice(0, 10)}.json`;
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      // silent — user can retry
    } finally {
      setExporting(null);
    }
  }, [audit, actionFilter, exporting]);

  // Loading state
  if (loading || planLoading) {
    return (
      <Shell>
        <PageWithTableSkeleton cols={3} rows={5} />
      </Shell>
    );
  }

  // Error state
  if (error) {
    return (
      <Shell>
        <ErrorState message={error} onRetry={refetch} />
      </Shell>
    );
  }

  // Plan gate: require Business or Enterprise
  if (!isAllowed) {
    return (
      <Shell>
        <div className="space-y-6">
          <div>
            <h1 className="text-lg font-semibold text-white">Audit Log</h1>
            <p className="text-sm text-neutral-500">
              Track all actions across your organization
            </p>
          </div>

          <div className="flex flex-col items-center justify-center rounded-xl border border-border bg-surface-100 py-16 px-6">
            <div className="flex h-16 w-16 items-center justify-center rounded-full bg-accent-500/10 mb-5">
              <FileText className="h-8 w-8 text-accent-400" />
            </div>
            <h2 className="text-xl font-semibold text-white mb-2">
              Requires Business Plan or Higher
            </h2>
            <p className="text-sm text-neutral-400 text-center max-w-md mb-6">
              The audit log provides a complete, searchable record of every
              action in your organization. Export logs for compliance and
              security analysis.
            </p>
            <div className="flex flex-wrap items-center justify-center gap-x-6 gap-y-2 text-xs text-neutral-500 mb-8">
              <span className="flex items-center gap-1.5">
                <svg
                  className="h-4 w-4 text-accent-400"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth={2}
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M5 13l4 4L19 7"
                  />
                </svg>
                Full action history
              </span>
              <span className="flex items-center gap-1.5">
                <svg
                  className="h-4 w-4 text-accent-400"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth={2}
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M5 13l4 4L19 7"
                  />
                </svg>
                CSV & JSON export
              </span>
              <span className="flex items-center gap-1.5">
                <svg
                  className="h-4 w-4 text-accent-400"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth={2}
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M5 13l4 4L19 7"
                  />
                </svg>
                Search & filter
              </span>
              <span className="flex items-center gap-1.5">
                <svg
                  className="h-4 w-4 text-accent-400"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth={2}
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M5 13l4 4L19 7"
                  />
                </svg>
                Compliance ready
              </span>
            </div>
            <Link
              href="/billing"
              className="rounded-lg bg-accent-500 hover:bg-accent-600 text-white px-6 py-2.5 text-sm font-medium transition-colors"
            >
              Upgrade to Business
            </Link>
          </div>
        </div>
      </Shell>
    );
  }

  return (
    <Shell>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Audit Log</h1>
            <p className="text-sm text-neutral-500">
              Track all actions across your organization
            </p>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={handleExportCSV}
              disabled={exporting !== null}
              className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm text-neutral-400 hover:text-white hover:bg-surface-300 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <Download className="h-4 w-4" />
              {exporting === "csv" ? "Exporting..." : "Export CSV"}
            </button>
            <button
              onClick={handleExportJSON}
              disabled={exporting !== null}
              className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm text-neutral-400 hover:text-white hover:bg-surface-300 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <Download className="h-4 w-4" />
              {exporting === "json" ? "Exporting..." : "Export JSON"}
            </button>
          </div>
        </div>

        {/* Search & Filter bar */}
        <div className="flex items-center gap-3">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-neutral-500" />
            <input
              type="text"
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              onKeyDown={handleSearchKeyDown}
              placeholder="Search audit entries..."
              className="w-full rounded-lg border border-border bg-surface-50 pl-9 pr-3 py-2 text-sm text-white placeholder-neutral-500 outline-none focus:border-accent-500 focus:ring-1 focus:ring-accent-500"
            />
          </div>
          <button
            onClick={handleSearch}
            className="rounded-lg bg-accent-500 hover:bg-accent-600 px-4 py-2 text-sm font-medium text-white transition-colors"
          >
            Search
          </button>
          <select
            value={actionFilter}
            onChange={handleActionFilterChange}
            className="rounded-lg border border-border bg-surface-50 px-3 py-2 text-sm text-white outline-none focus:border-accent-500"
          >
            {actionOptions.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
        </div>

        {/* Total count */}
        <div className="flex items-center justify-between text-xs text-neutral-500">
          <span>
            {total} {total === 1 ? "entry" : "entries"} found
          </span>
          <span>
            Page {page + 1} of {totalPages}
          </span>
        </div>

        {/* Table */}
        {entries.length === 0 ? (
          <EmptyState
            title="No audit entries"
            description="No audit log entries match your current filters."
          />
        ) : (
          <div className="overflow-hidden rounded-xl border border-border">
            <table className="w-full">
              <thead>
                <tr className="border-b border-border bg-surface-100 text-left text-xs font-medium text-neutral-500">
                  <th className="px-4 py-3">Time</th>
                  <th className="px-4 py-3">Actor</th>
                  <th className="px-4 py-3">Action</th>
                  <th className="px-4 py-3">Cluster</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {entries.map((entry, idx) => (
                  <tr
                    key={`${entry.time}-${idx}`}
                    className="bg-surface-50 transition-colors hover:bg-surface-100"
                  >
                    <td className="px-4 py-3">
                      <span
                        className="flex items-center gap-1.5 text-sm text-neutral-300"
                        title={fullTimestamp(entry.time)}
                      >
                        <Clock className="h-3.5 w-3.5 text-neutral-500 flex-shrink-0" />
                        {relativeTime(entry.time)}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-sm text-neutral-300">
                      {entry.actor}
                    </td>
                    <td className="px-4 py-3">
                      <span
                        className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium ${actionBadgeColor(entry.action)}`}
                      >
                        {entry.action}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-sm text-neutral-500">
                      {entry.cluster || "-"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-center gap-2">
            <button
              onClick={() => setPage((p) => Math.max(0, p - 1))}
              disabled={page === 0}
              className="flex items-center gap-1 rounded-lg border border-border px-3 py-2 text-sm text-neutral-400 hover:text-white hover:bg-surface-300 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <ChevronLeft className="h-4 w-4" />
              Previous
            </button>
            <div className="flex items-center gap-1">
              {Array.from({ length: Math.min(totalPages, 7) }, (_, i) => {
                let pageNum: number;
                if (totalPages <= 7) {
                  pageNum = i;
                } else if (page < 3) {
                  pageNum = i;
                } else if (page > totalPages - 4) {
                  pageNum = totalPages - 7 + i;
                } else {
                  pageNum = page - 3 + i;
                }
                return (
                  <button
                    key={pageNum}
                    onClick={() => setPage(pageNum)}
                    className={`min-w-[2rem] rounded-md px-2 py-1.5 text-xs font-medium transition-colors ${
                      page === pageNum
                        ? "bg-accent-500/15 text-accent-400"
                        : "text-neutral-400 hover:bg-surface-300 hover:text-white"
                    }`}
                  >
                    {pageNum + 1}
                  </button>
                );
              })}
            </div>
            <button
              onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
              disabled={page >= totalPages - 1}
              className="flex items-center gap-1 rounded-lg border border-border px-3 py-2 text-sm text-neutral-400 hover:text-white hover:bg-surface-300 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Next
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        )}
      </div>
    </Shell>
  );
}
