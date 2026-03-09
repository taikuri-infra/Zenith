"use client";

import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { StatusBadge } from "@/components/status-badge";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { StatCardRowSkeleton, Skeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import type { WarRoomData, ServiceHealthItem } from "@/lib/api";
import { useApi } from "@/hooks/use-api";
import Link from "next/link";
import {
  ArrowUpRight,
  AlertTriangle,
} from "lucide-react";

export default function WarRoomPage() {
  const apiClient = getApi();
  const warRoom = useApi<WarRoomData>(() => apiClient.warRoom.get());
  const services = useApi<ServiceHealthItem[]>(() => apiClient.services.list());

  const kpis = warRoom.data?.kpis;

  return (
    <Shell>
      <div className="space-y-6">
        <h1 className="text-lg font-semibold text-white">Command Center</h1>

        {/* KPI Row */}
        {warRoom.loading ? (
          <StatCardRowSkeleton count={6} />
        ) : warRoom.error ? (
          <ErrorState error={warRoom.error} onRetry={warRoom.refetch} />
        ) : kpis ? (
          <div className="grid grid-cols-6 gap-3">
            <StatCard
              label="MRR"
              value={`€${kpis.mrr.toLocaleString()}`}
              sub={kpis.mrrTrend >= 0 ? `+${kpis.mrrTrend.toFixed(1)}%` : `${kpis.mrrTrend.toFixed(1)}%`}
            />
            <StatCard
              label="Active Customers"
              value={kpis.activeCustomers}
              sub={`of ${kpis.totalCustomers} total`}
            />
            <StatCard
              label="New Signups"
              value={kpis.newSignups}
              sub="this month"
            />
            <StatCard
              label="Churn Rate"
              value={`${kpis.churnRate.toFixed(1)}%`}
              sub="monthly"
              alert={kpis.churnRate > 5}
            />
            <StatCard
              label="Avg Response"
              value={kpis.avgResponseTime || "—"}
              sub="support SLA"
            />
            <StatCard
              label="Health Score"
              value={kpis.healthScore}
              sub="out of 100"
              alert={kpis.healthScore < 70}
            />
          </div>
        ) : null}

        {/* Charts Row */}
        <div className="grid grid-cols-2 gap-4">
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <h2 className="mb-3 text-sm font-medium text-white">Revenue Trend</h2>
            <div className="flex h-48 items-center justify-center text-neutral-500 text-sm">
              Revenue chart — data from /analytics/revenue
            </div>
          </div>
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <h2 className="mb-3 text-sm font-medium text-white">Customer Growth</h2>
            <div className="flex h-48 items-center justify-center text-neutral-500 text-sm">
              Growth chart — data from /analytics/growth
            </div>
          </div>
        </div>

        {/* 3-Column Row */}
        <div className="grid grid-cols-3 gap-4">
          {/* Service Health */}
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <div className="mb-3 flex items-center justify-between">
              <h2 className="text-sm font-medium text-white">Services</h2>
              <Link href="/services" className="flex items-center gap-1 text-xs text-neutral-500 hover:text-white">
                View all <ArrowUpRight className="h-3 w-3" />
              </Link>
            </div>
            {services.loading ? (
              <div className="space-y-1.5">
                {Array.from({ length: 6 }).map((_, i) => (
                  <Skeleton key={i} className="h-7 w-full rounded" />
                ))}
              </div>
            ) : services.data ? (
              <div className="grid grid-cols-2 gap-1.5">
                {services.data.slice(0, 12).map((svc) => (
                  <div key={svc.name} className="flex items-center gap-1.5 rounded px-2 py-1">
                    <div className={`h-1.5 w-1.5 rounded-full ${
                      svc.status === "healthy" ? "bg-emerald-400" :
                      svc.status === "degraded" ? "bg-amber-400" :
                      svc.status === "down" ? "bg-red-400" : "bg-neutral-500"
                    }`} />
                    <span className="truncate text-xs text-neutral-300">{svc.name}</span>
                  </div>
                ))}
              </div>
            ) : null}
          </div>

          {/* Recent Alerts */}
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <div className="mb-3 flex items-center justify-between">
              <h2 className="text-sm font-medium text-white">Alerts</h2>
              <Link href="/alerts" className="flex items-center gap-1 text-xs text-neutral-500 hover:text-white">
                View all <ArrowUpRight className="h-3 w-3" />
              </Link>
            </div>
            {warRoom.data?.recentAlerts && warRoom.data.recentAlerts.length > 0 ? (
              <div className="space-y-2">
                {warRoom.data.recentAlerts.slice(0, 5).map((alert, i) => (
                  <div key={i} className="flex items-start gap-2 text-xs">
                    <AlertTriangle className={`mt-0.5 h-3 w-3 flex-shrink-0 ${
                      alert.severity === "critical" ? "text-red-400" : "text-amber-400"
                    }`} />
                    <span className="text-neutral-300">{alert.name}</span>
                  </div>
                ))}
              </div>
            ) : (
              <EmptyState title="No alerts" description="All systems nominal." />
            )}
          </div>

          {/* Active Tickets */}
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <div className="mb-3 flex items-center justify-between">
              <h2 className="text-sm font-medium text-white">Active Tickets</h2>
              <Link href="/support" className="flex items-center gap-1 text-xs text-neutral-500 hover:text-white">
                View all <ArrowUpRight className="h-3 w-3" />
              </Link>
            </div>
            {warRoom.data?.activeTickets && warRoom.data.activeTickets.length > 0 ? (
              <div className="space-y-2">
                {warRoom.data.activeTickets.map((ticket) => (
                  <Link key={ticket.id} href={`/support/${ticket.id}`} className="block rounded-md border border-border bg-surface-200 px-3 py-2 hover:bg-surface-300 transition-colors">
                    <div className="flex items-center justify-between">
                      <span className="text-xs text-white truncate">{ticket.subject}</span>
                      <StatusBadge status={ticket.priority === "critical" ? "error" : ticket.priority === "high" ? "warning" : "healthy"} />
                    </div>
                    <span className="text-[10px] text-neutral-500">{ticket.age}</span>
                  </Link>
                ))}
              </div>
            ) : (
              <EmptyState title="No tickets" description="All tickets resolved." />
            )}
          </div>
        </div>

        {/* Quick Actions */}
        <div className="flex gap-3">
          {[
            { label: "New Customer", href: "/customers/new" },
            { label: "View Logs", href: "/logs" },
            { label: "Check Backups", href: "/backups" },
            { label: "Security Scan", href: "/security" },
          ].map((action) => (
            <Link
              key={action.href}
              href={action.href}
              className="rounded-md border border-border bg-surface-100 px-4 py-2 text-xs text-neutral-400 hover:bg-surface-200 hover:text-white transition-colors"
            >
              {action.label}
            </Link>
          ))}
        </div>
      </div>
    </Shell>
  );
}
