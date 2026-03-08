"use client";

import { Shell } from "@/components/shell";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { useApi } from "@/hooks/use-api";
import { type PodExecSession } from "@/lib/api";
import { getApi } from "@/lib/get-api";
import { useState, useMemo, useCallback } from "react";
import {
  Terminal,
  Download,
  ChevronLeft,
  ChevronRight,
  Clock,
} from "lucide-react";

const PAGE_SIZE = 50;

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

function formatDuration(secs: number): string {
  if (secs === 0) return "-";
  const hrs = Math.floor(secs / 3600);
  const mins = Math.floor((secs % 3600) / 60);
  const s = secs % 60;
  if (hrs > 0) return `${hrs}h ${mins}m ${s}s`;
  if (mins > 0) return `${mins}m ${s}s`;
  return `${s}s`;
}

export default function SSHSessionsPage() {
  const { podSessions } = getApi();

  const [page, setPage] = useState(0);
  const [downloadingId, setDownloadingId] = useState<string | null>(null);

  const {
    data: sessionsData,
    loading,
    error,
    refetch,
  } = useApi(
    () => podSessions.list(PAGE_SIZE, page * PAGE_SIZE),
    [page]
  );

  const sessions: PodExecSession[] = useMemo(
    () => sessionsData?.sessions ?? [],
    [sessionsData]
  );
  const total = sessionsData?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  const handleDownloadRecording = useCallback(
    async (sessionId: string) => {
      if (downloadingId) return;
      setDownloadingId(sessionId);
      try {
        const result = await podSessions.getRecordingURL(sessionId);
        window.open(result.url, "_blank");
      } catch {
        // silent
      } finally {
        setDownloadingId(null);
      }
    },
    [podSessions, downloadingId]
  );

  // Loading state
  if (loading) {
    return (
      <Shell>
        <PageWithTableSkeleton cols={8} rows={5} />
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

  return (
    <Shell>
      <div className="space-y-6">
        {/* Header */}
        <div>
          <div className="flex items-center gap-2">
            <Terminal className="h-5 w-5 text-accent-400" />
            <h1 className="text-lg font-semibold text-white">SSH Sessions</h1>
          </div>
          <p className="mt-1 text-sm text-neutral-500">
            Audit trail of all pod terminal sessions (Business+ only)
          </p>
        </div>

        {/* Total count */}
        <div className="flex items-center justify-between text-xs text-neutral-500">
          <span>
            {total} {total === 1 ? "session" : "sessions"} found
          </span>
          <span>
            Page {page + 1} of {totalPages}
          </span>
        </div>

        {/* Table */}
        {sessions.length === 0 ? (
          <EmptyState
            title="No SSH sessions"
            description="No pod terminal sessions have been recorded yet."
          />
        ) : (
          <div className="overflow-hidden rounded-xl border border-border">
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-border bg-surface-100 text-left text-xs font-medium text-neutral-500">
                    <th className="px-4 py-3">User Email</th>
                    <th className="px-4 py-3">App</th>
                    <th className="px-4 py-3">Pod</th>
                    <th className="px-4 py-3">Command</th>
                    <th className="px-4 py-3">Status</th>
                    <th className="px-4 py-3">Started</th>
                    <th className="px-4 py-3">Duration</th>
                    <th className="px-4 py-3">Recording</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {sessions.map((session) => (
                    <tr
                      key={session.id}
                      className="bg-surface-50 transition-colors hover:bg-surface-100"
                    >
                      <td className="px-4 py-3 text-sm text-neutral-300">
                        {session.user_email}
                      </td>
                      <td className="px-4 py-3 text-sm text-neutral-300">
                        {session.app_name}
                      </td>
                      <td className="px-4 py-3">
                        <span className="font-mono text-xs text-neutral-400">
                          {session.pod_name}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <code className="rounded bg-surface-300 px-1.5 py-0.5 text-xs text-neutral-300">
                          {session.command}
                        </code>
                      </td>
                      <td className="px-4 py-3">
                        <span
                          className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium ${
                            session.status === "active"
                              ? "bg-green-500/15 text-green-400"
                              : "bg-neutral-500/15 text-neutral-400"
                          }`}
                        >
                          {session.status}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <span
                          className="flex items-center gap-1.5 text-sm text-neutral-300"
                          title={fullTimestamp(session.started_at)}
                        >
                          <Clock className="h-3.5 w-3.5 text-neutral-500 flex-shrink-0" />
                          {relativeTime(session.started_at)}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-sm text-neutral-300">
                        {formatDuration(session.duration_secs)}
                      </td>
                      <td className="px-4 py-3">
                        {session.recording_key ? (
                          <button
                            onClick={() => handleDownloadRecording(session.id)}
                            disabled={downloadingId === session.id}
                            className="flex items-center gap-1.5 text-sm text-accent-400 hover:text-accent-300 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                          >
                            <Download className="h-3.5 w-3.5" />
                            {downloadingId === session.id
                              ? "Loading..."
                              : "Download"}
                          </button>
                        ) : (
                          <span className="text-xs text-neutral-600">-</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
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
