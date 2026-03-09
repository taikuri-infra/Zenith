"use client";

import { useState } from "react";
import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { StatCardRowSkeleton, TableSkeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import type { ActiveSession } from "@/lib/api";
import { useApi, useMutation } from "@/hooks/use-api";
import { Monitor, X } from "lucide-react";

export default function SessionsPage() {
  const apiClient = getApi();
  const { data: sessions, loading, error, refetch } = useApi<ActiveSession[]>(
    () => apiClient.security.sessions()
  );
  const [terminatingId, setTerminatingId] = useState<string | null>(null);

  const terminateMutation = useMutation<string, void>(
    (sessionId) => apiClient.security.terminateSession(sessionId)
  );

  const handleTerminate = async (sessionId: string) => {
    if (!confirm("Are you sure you want to terminate this session?")) return;
    setTerminatingId(sessionId);
    try {
      await terminateMutation.execute(sessionId);
      refetch();
    } finally {
      setTerminatingId(null);
    }
  };

  const totalSessions = sessions?.length ?? 0;
  const uniqueUsers = sessions ? new Set(sessions.map((s) => s.email)).size : 0;

  return (
    <Shell>
      <div className="space-y-6">
        <h1 className="text-lg font-semibold text-white">Active Sessions</h1>

        {/* Stats */}
        {loading ? (
          <StatCardRowSkeleton count={3} />
        ) : sessions ? (
          <div className="grid grid-cols-3 gap-4">
            <StatCard label="Active Sessions" value={totalSessions} sub="currently active" />
            <StatCard label="Unique Users" value={uniqueUsers} sub="with active sessions" />
            <StatCard
              label="Multiple Sessions"
              value={totalSessions - uniqueUsers}
              sub="users with >1 session"
              alert={totalSessions - uniqueUsers > 0}
            />
          </div>
        ) : null}

        {/* Sessions Table */}
        {loading ? (
          <TableSkeleton columns={6} rows={5} />
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : !sessions || sessions.length === 0 ? (
          <EmptyState
            title="No active sessions"
            description="No users are currently signed in."
            icon={Monitor}
          />
        ) : (
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Email</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">IP Address</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Device</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Location</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Last Seen</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Actions</th>
                </tr>
              </thead>
              <tbody>
                {sessions.map((session) => (
                  <tr key={session.id} className="border-b border-border last:border-0 transition-colors hover:bg-surface-200">
                    <td className="px-4 py-3">
                      <span className="font-medium text-white">{session.email}</span>
                      {session.isAdmin && (
                        <span className="ml-2 rounded bg-accent-600/15 px-1.5 py-0.5 text-[10px] font-medium text-accent-400">
                          Admin
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-neutral-400">{session.ipAddress}</td>
                    <td className="px-4 py-3 text-neutral-300 text-xs">{session.device}</td>
                    <td className="px-4 py-3 text-neutral-400 text-xs">{session.location || "—"}</td>
                    <td className="px-4 py-3 text-xs text-neutral-500">{session.lastSeen}</td>
                    <td className="px-4 py-3">
                      <button
                        onClick={() => handleTerminate(session.id)}
                        disabled={terminatingId === session.id}
                        className="flex items-center gap-1 rounded-md border border-red-500/30 bg-red-500/10 px-2 py-1 text-xs text-red-400 hover:bg-red-500/20 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        {terminatingId === session.id ? (
                          <div className="h-3 w-3 animate-spin rounded-full border border-red-400/30 border-t-red-400" />
                        ) : (
                          <X className="h-3 w-3" />
                        )}
                        Terminate
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </Shell>
  );
}
