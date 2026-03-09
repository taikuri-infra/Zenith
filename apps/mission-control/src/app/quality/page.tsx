"use client";

import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { StatusBadge } from "@/components/status-badge";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { StatCardRowSkeleton, TableSkeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import type { QualityMetrics, QualityTicket } from "@/lib/api";
import { useApi } from "@/hooks/use-api";
import { ClipboardCheck } from "lucide-react";

function priorityBadge(priority: string) {
  switch (priority) {
    case "critical":
      return <span className="rounded-full bg-red-500/15 px-2 py-0.5 text-xs font-medium text-red-400">Critical</span>;
    case "high":
      return <span className="rounded-full bg-orange-500/15 px-2 py-0.5 text-xs font-medium text-orange-400">High</span>;
    case "normal":
      return <span className="rounded-full bg-neutral-500/15 px-2 py-0.5 text-xs font-medium text-neutral-300">Normal</span>;
    case "low":
      return <span className="rounded-full bg-neutral-500/10 px-2 py-0.5 text-xs font-medium text-neutral-500">Low</span>;
    default:
      return <span className="rounded-full bg-neutral-500/10 px-2 py-0.5 text-xs font-medium text-neutral-400">{priority}</span>;
  }
}

export default function QualityPage() {
  const apiClient = getApi();
  const metrics = useApi<QualityMetrics>(() => apiClient.quality.metrics());
  const tickets = useApi<QualityTicket[]>(() => apiClient.quality.tickets());

  return (
    <Shell>
      <div className="space-y-6">
        <h1 className="text-lg font-semibold text-white">Quality & SLA</h1>

        {/* KPI Cards */}
        {metrics.loading ? (
          <StatCardRowSkeleton />
        ) : metrics.error ? (
          <ErrorState error={metrics.error} onRetry={metrics.refetch} />
        ) : metrics.data ? (
          <div className="grid grid-cols-4 gap-4">
            <StatCard
              label="Open Tickets"
              value={metrics.data.openTickets}
              sub="currently open"
              alert={metrics.data.openTickets > 10}
            />
            <StatCard
              label="Resolved This Week"
              value={metrics.data.resolvedThisWeek}
              sub="tickets closed"
            />
            <StatCard
              label="Avg Response Time"
              value={metrics.data.avgResponseTime}
              sub="first response"
            />
            <StatCard
              label="SLA Compliance"
              value={`${metrics.data.slaCompliance.toFixed(1)}%`}
              sub="target 99.9%"
              alert={metrics.data.slaCompliance < 99}
            />
          </div>
        ) : null}

        {/* Uptime & Error Rate */}
        {metrics.data && (
          <div className="grid grid-cols-3 gap-4">
            <StatCard
              label="Uptime"
              value={`${metrics.data.uptime.toFixed(3)}%`}
              sub="last 30 days"
              alert={metrics.data.uptime < 99.9}
            />
            <StatCard
              label="Error Rate"
              value={`${metrics.data.errorRate.toFixed(2)}%`}
              sub="4xx + 5xx / total"
              alert={metrics.data.errorRate > 1}
            />
            <StatCard
              label="P95 Latency"
              value={metrics.data.p95Latency}
              sub="API response time"
            />
          </div>
        )}

        {/* Tickets by Priority */}
        <section>
          <h2 className="mb-3 text-sm font-medium text-white">Tickets by Priority</h2>
          {tickets.loading ? (
            <TableSkeleton columns={5} rows={5} />
          ) : tickets.error ? (
            <ErrorState error={tickets.error} onRetry={tickets.refetch} />
          ) : !tickets.data || tickets.data.length === 0 ? (
            <EmptyState
              title="No tickets"
              description="All support tickets have been resolved."
              icon={ClipboardCheck}
            />
          ) : (
            <div className="overflow-hidden rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-surface-100">
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Subject</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Customer</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Priority</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Age</th>
                  </tr>
                </thead>
                <tbody>
                  {tickets.data.map((ticket) => (
                    <tr key={ticket.id} className="border-b border-border last:border-0 transition-colors hover:bg-surface-200">
                      <td className="px-4 py-3 text-white">{ticket.subject}</td>
                      <td className="px-4 py-3 text-neutral-400">{ticket.customer}</td>
                      <td className="px-4 py-3">{priorityBadge(ticket.priority)}</td>
                      <td className="px-4 py-3">
                        <StatusBadge status={ticket.status === "open" ? "warning" : ticket.status === "resolved" ? "healthy" : "idle"} label={ticket.status} />
                      </td>
                      <td className="px-4 py-3 text-xs text-neutral-500">{ticket.age}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </div>
    </Shell>
  );
}
