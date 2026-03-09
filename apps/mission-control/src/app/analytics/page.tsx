"use client";

import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { ErrorState } from "@/components/error-state";
import { StatCardRowSkeleton, TableSkeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import type { RevenueStats, GrowthStats, UsageAnalytics } from "@/lib/api";
import { useApi } from "@/hooks/use-api";

export default function AnalyticsPage() {
  const apiClient = getApi();
  const revenue = useApi<RevenueStats>(() => apiClient.analytics.revenue());
  const growth = useApi<GrowthStats>(() => apiClient.analytics.growth());
  const usage = useApi<UsageAnalytics>(() => apiClient.analytics.usage());

  return (
    <Shell>
      <div className="space-y-6">
        <h1 className="text-lg font-semibold text-white">Analytics</h1>

        {/* Revenue KPIs */}
        {revenue.loading ? (
          <StatCardRowSkeleton />
        ) : revenue.error ? (
          <ErrorState error={revenue.error} onRetry={revenue.refetch} />
        ) : revenue.data ? (
          <>
            <div className="grid grid-cols-4 gap-4">
              <StatCard label="MRR" value={`€${revenue.data.mrr.toLocaleString()}`} sub="monthly recurring" />
              <StatCard label="ARR" value={`€${revenue.data.arr.toLocaleString()}`} sub="annual recurring" />
              <StatCard label="Churn Rate" value={`${revenue.data.churnRate.toFixed(1)}%`} sub="this month" alert={revenue.data.churnRate > 5} />
              <StatCard label="LTV" value={`€${revenue.data.ltv.toLocaleString()}`} sub="lifetime value" />
            </div>

            {/* Revenue by Plan */}
            <section>
              <h2 className="mb-3 text-sm font-medium text-white">Revenue by Plan</h2>
              <div className="overflow-hidden rounded-lg border border-border">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-border bg-surface-100">
                      <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Plan</th>
                      <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Revenue</th>
                      <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Customers</th>
                    </tr>
                  </thead>
                  <tbody>
                    {revenue.data.revenueByPlan?.map((p) => (
                      <tr key={p.plan} className="border-b border-border last:border-0">
                        <td className="px-4 py-3 text-white capitalize">{p.plan}</td>
                        <td className="px-4 py-3 text-neutral-300">€{p.revenue.toLocaleString()}</td>
                        <td className="px-4 py-3 text-neutral-300">{p.count}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>
          </>
        ) : null}

        {/* Growth */}
        {growth.loading ? (
          <StatCardRowSkeleton count={3} />
        ) : growth.error ? (
          <ErrorState error={growth.error} onRetry={growth.refetch} />
        ) : growth.data ? (
          <section>
            <h2 className="mb-3 text-sm font-medium text-white">Growth</h2>
            <div className="grid grid-cols-3 gap-4">
              <StatCard label="Total Users" value={growth.data.totalUsers} sub="all time" />
              <StatCard label="New This Month" value={growth.data.newThisMonth} sub="signups" />
              <StatCard label="Churned" value={growth.data.churnedThisMonth} sub="this month" alert={growth.data.churnedThisMonth > 0} />
            </div>
          </section>
        ) : null}

        {/* Feature Usage */}
        {usage.loading ? (
          <TableSkeleton columns={3} rows={5} />
        ) : usage.error ? (
          <ErrorState error={usage.error} onRetry={usage.refetch} />
        ) : usage.data && usage.data.topFeatures ? (
          <section>
            <h2 className="mb-3 text-sm font-medium text-white">Feature Usage</h2>
            <div className="overflow-hidden rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-surface-100">
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Feature</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Usage Count</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Users</th>
                  </tr>
                </thead>
                <tbody>
                  {usage.data.topFeatures.map((f) => (
                    <tr key={f.feature} className="border-b border-border last:border-0">
                      <td className="px-4 py-3 text-white">{f.feature}</td>
                      <td className="px-4 py-3 text-neutral-300">{f.usageCount}</td>
                      <td className="px-4 py-3 text-neutral-300">{f.userCount}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        ) : null}
      </div>
    </Shell>
  );
}
