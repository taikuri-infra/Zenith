"use client";

import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { StatusBadge } from "@/components/status-badge";
import { ProgressBar } from "@/components/progress-bar";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import {
  StatCardRowSkeleton,
  TableSkeleton,
  ActivityListSkeleton,
  Skeleton,
} from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import type { Cluster, Module, AuditEntry, PlatformUpdate, CustomerStats, PlatformUsageSummary } from "@/lib/api";
import { useApi } from "@/hooks/use-api";
import { ArrowUpRight } from "lucide-react";
import Link from "next/link";

export default function DashboardPage() {
  const apiClient = getApi();

  const clusters = useApi<Cluster[]>(() => apiClient.clusters.list());
  const modules = useApi<Module[]>(() => apiClient.modules.list());
  const audit = useApi<AuditEntry[]>(() => apiClient.audit.list({ limit: 4 }));
  const platformUpdate = useApi<PlatformUpdate>(() => apiClient.updates.check());
  const dashboardStats = useApi(() => apiClient.dashboard.stats());
  const customerStats = useApi<CustomerStats>(() => apiClient.customers.stats());
  const platformUsage = useApi<PlatformUsageSummary>(() => apiClient.dashboard.usage());

  const updatesAvailable = modules.data
    ? modules.data.filter((m) => m.status === "update_available").length
    : 0;

  return (
    <Shell>
      <div className="space-y-6">
        {/* Page title */}
        <h1 className="text-lg font-semibold text-white">Platform Overview</h1>

        {/* Stat cards */}
        {dashboardStats.loading ? (
          <StatCardRowSkeleton />
        ) : dashboardStats.error ? (
          <ErrorState error={dashboardStats.error} onRetry={dashboardStats.refetch} />
        ) : dashboardStats.data ? (
          <div className="grid grid-cols-4 gap-4">
            <StatCard
              label="Clusters"
              value={dashboardStats.data.clusterCount}
              sub={dashboardStats.data.allHealthy ? "all healthy" : "issues detected"}
              alert={!dashboardStats.data.allHealthy}
            />
            <StatCard
              label="Customers"
              value={customerStats.data?.totalCustomers ?? dashboardStats.data.tenantCount}
              sub={`${customerStats.data?.activeCustomers ?? dashboardStats.data.activeToday} active`}
            />
            <StatCard
              label="MRR"
              value={customerStats.data?.mrr ?? dashboardStats.data.monthlyCost}
              sub={`${customerStats.data?.newThisMonth ?? 0} new this month`}
            />
            <StatCard
              label="Updates"
              value={updatesAvailable}
              sub={`${updatesAvailable} available`}
              alert={updatesAvailable > 0}
            />
          </div>
        ) : null}

        {/* Clusters table */}
        <section>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-sm font-medium text-white">Clusters</h2>
            <Link
              href="/clusters"
              className="flex items-center gap-1 text-xs text-neutral-500 transition-colors hover:text-white"
            >
              View all <ArrowUpRight className="h-3 w-3" />
            </Link>
          </div>
          {clusters.loading ? (
            <TableSkeleton columns={6} rows={3} />
          ) : clusters.error ? (
            <ErrorState error={clusters.error} onRetry={clusters.refetch} />
          ) : !clusters.data || clusters.data.length === 0 ? (
            <EmptyState
              title="No clusters"
              description="No clusters have been created yet."
            />
          ) : (
            <div className="overflow-hidden rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-surface-100">
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      Name
                    </th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      K8s
                    </th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      Nodes
                    </th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      CPU
                    </th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      RAM
                    </th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      Status
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {clusters.data.map((cluster) => (
                    <tr
                      key={cluster.name}
                      className="border-b border-border last:border-0 transition-colors hover:bg-surface-200"
                    >
                      <td className="px-4 py-3">
                        <Link
                          href={`/clusters/${cluster.name}`}
                          className="font-medium text-white hover:text-accent-400 transition-colors"
                        >
                          {cluster.name}
                        </Link>
                        <span className="ml-2 text-xs text-neutral-500">
                          {cluster.region}
                        </span>
                      </td>
                      <td className="px-4 py-3 font-mono text-xs text-neutral-400">
                        {cluster.k8sVersion}
                        {cluster.upgradeAvailable && (
                          <span className="ml-1.5 text-amber-400">&#9888;</span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-neutral-300">
                        {cluster.nodes}
                      </td>
                      <td className="w-36 px-4 py-3">
                        <ProgressBar
                          percent={cluster.cpuPercent}
                          label={`${cluster.cpuPercent}%`}
                        />
                      </td>
                      <td className="w-36 px-4 py-3">
                        <ProgressBar
                          percent={cluster.ramPercent}
                          label={`${cluster.ramPercent}%`}
                        />
                      </td>
                      <td className="px-4 py-3">
                        <StatusBadge status={cluster.status} />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>

        {/* Platform Resource Usage */}
        {platformUsage.loading ? (
          <StatCardRowSkeleton />
        ) : platformUsage.data ? (
          <section>
            <h2 className="mb-3 text-sm font-medium text-white">Platform Resource Usage</h2>
            <div className="grid grid-cols-4 gap-4">
              <StatCard
                label="Total CPU"
                value={`${platformUsage.data.totalCpu} cores`}
                sub={`${platformUsage.data.customersReporting} customers reporting`}
              />
              <StatCard
                label="Total RAM"
                value={`${platformUsage.data.totalRam} GB`}
                sub="across all customers"
              />
              <StatCard
                label="Total Storage"
                value={`${platformUsage.data.totalStorage} GB`}
                sub="DB + volumes"
              />
              <StatCard
                label="Reporting"
                value={platformUsage.data.customersReporting}
                sub="customers with metering"
              />
            </div>
          </section>
        ) : null}

        {/* Two columns: Updates + Activity */}
        <div className="grid grid-cols-2 gap-6">
          {/* Available Updates */}
          <section>
            <div className="mb-3 flex items-center justify-between">
              <h2 className="text-sm font-medium text-white">
                Available Updates
              </h2>
              <Link
                href="/updates"
                className="flex items-center gap-1 text-xs text-neutral-500 transition-colors hover:text-white"
              >
                View all <ArrowUpRight className="h-3 w-3" />
              </Link>
            </div>
            {platformUpdate.loading || modules.loading ? (
              <div className="space-y-2">
                <Skeleton className="h-12 w-full rounded-lg" />
                <Skeleton className="h-12 w-full rounded-lg" />
                <Skeleton className="h-12 w-full rounded-lg" />
              </div>
            ) : platformUpdate.error ? (
              <ErrorState
                error={platformUpdate.error}
                onRetry={platformUpdate.refetch}
              />
            ) : (
              <div className="space-y-2">
                {/* Platform update */}
                {platformUpdate.data && (
                  <div className="rounded-lg border border-accent-600/30 bg-accent-600/5 p-3">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <span className="rounded bg-accent-600/20 px-1.5 py-0.5 text-[10px] font-medium text-accent-400">
                          NEW
                        </span>
                        <span className="text-sm font-medium text-white">
                          Zenith {platformUpdate.data.version}
                        </span>
                      </div>
                      <span className="text-xs text-neutral-500">
                        Current: {platformUpdate.data.current}
                      </span>
                    </div>
                  </div>
                )}

                {/* Module updates */}
                {modules.data
                  ?.filter((m) => m.status === "update_available")
                  .map((mod) => (
                    <div
                      key={mod.name}
                      className="flex items-center justify-between rounded-lg border border-border bg-surface-100 p-3"
                    >
                      <div>
                        <span className="text-sm text-white">{mod.name}</span>
                        <span className="ml-2 text-xs text-neutral-500">
                          {mod.installed} &rarr; {mod.latest}
                        </span>
                      </div>
                      <Link
                        href={`/modules/${encodeURIComponent(mod.name)}`}
                        className="text-xs text-accent-400 hover:text-accent-300"
                      >
                        View
                      </Link>
                    </div>
                  ))}

                {!platformUpdate.data &&
                  (!modules.data ||
                    modules.data.filter((m) => m.status === "update_available")
                      .length === 0) && (
                    <EmptyState
                      title="All up to date"
                      description="No updates are available at this time."
                    />
                  )}
              </div>
            )}
          </section>

          {/* Recent Activity */}
          <section>
            <div className="mb-3 flex items-center justify-between">
              <h2 className="text-sm font-medium text-white">
                Recent Activity
              </h2>
              <Link
                href="/audit"
                className="flex items-center gap-1 text-xs text-neutral-500 transition-colors hover:text-white"
              >
                View all <ArrowUpRight className="h-3 w-3" />
              </Link>
            </div>
            {audit.loading ? (
              <ActivityListSkeleton rows={4} />
            ) : audit.error ? (
              <ErrorState error={audit.error} onRetry={audit.refetch} />
            ) : !audit.data || audit.data.length === 0 ? (
              <EmptyState
                title="No activity"
                description="No audit events have been recorded yet."
              />
            ) : (
              <div className="space-y-0 rounded-lg border border-border bg-surface-100">
                {audit.data.map((entry, i) => (
                  <div
                    key={i}
                    className="flex items-start gap-3 border-b border-border px-3 py-2.5 last:border-0"
                  >
                    <span className="mt-px font-mono text-xs text-neutral-500">
                      {entry.time}
                    </span>
                    <div className="min-w-0 flex-1">
                      <span className="text-sm text-neutral-300">
                        <span className="font-medium text-white">
                          {entry.actor}
                        </span>{" "}
                        {entry.action}
                      </span>
                      {entry.cluster && (
                        <span className="ml-1.5 text-xs text-neutral-500">
                          on {entry.cluster}
                        </span>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </section>
        </div>
      </div>
    </Shell>
  );
}
