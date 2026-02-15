"use client";

import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import {
  StatCardRowSkeleton,
  TableSkeleton,
} from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import type { InfraOverview } from "@/lib/api";
import { useApi } from "@/hooks/use-api";
import { HardDrive } from "lucide-react";

export default function InfrastructurePage() {
  const apiClient = getApi();
  const { data: infra, loading, error, refetch } = useApi<InfraOverview>(
    () => apiClient.infrastructure.overview()
  );

  return (
    <Shell>
      <div className="space-y-6">
        <h1 className="text-lg font-semibold text-white">Infrastructure</h1>

        {/* Stat cards */}
        {loading ? (
          <StatCardRowSkeleton />
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : !infra ? (
          <EmptyState
            title="No infrastructure data"
            description="Infrastructure information is not available."
            icon={HardDrive}
          />
        ) : (
          <>
            <div className="grid grid-cols-4 gap-4">
              <StatCard
                label="Servers"
                value={infra.servers}
                sub="Hetzner Cloud"
              />
              <StatCard
                label="Volumes"
                value={infra.volumes}
                sub={`${infra.volumeSize} total`}
              />
              <StatCard
                label="Load Balancers"
                value={infra.loadBalancers}
                sub={`${infra.lbPublic} public, ${infra.lbInternal} internal`}
              />
              <StatCard
                label="Monthly Cost"
                value={infra.monthlyCost}
                sub="Current billing period"
              />
            </div>

            {/* Resource breakdown */}
            <section>
              <h2 className="mb-3 text-sm font-medium text-white">
                Resource Breakdown
              </h2>
              {infra.resources.length === 0 ? (
                <EmptyState
                  title="No resources"
                  description="No infrastructure resources found."
                />
              ) : (
                <div className="overflow-hidden rounded-lg border border-border">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-border bg-surface-100">
                        <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                          Resource
                        </th>
                        <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                          Type
                        </th>
                        <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                          Count
                        </th>
                        <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                          Cluster
                        </th>
                        <th className="px-4 py-2.5 text-right text-xs font-medium text-neutral-500">
                          Monthly Cost
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {infra.resources.map((resource, i) => (
                        <tr
                          key={i}
                          className={`transition-colors hover:bg-surface-200 ${
                            i < infra.resources.length - 1
                              ? "border-b border-border"
                              : ""
                          }`}
                        >
                          <td className="px-4 py-3 font-medium text-white">
                            {resource.name}
                          </td>
                          <td className="px-4 py-3 text-neutral-400">
                            {resource.type}
                          </td>
                          <td className="px-4 py-3 text-neutral-300">
                            {resource.count}
                          </td>
                          <td className="px-4 py-3 text-neutral-400">
                            {resource.cluster}
                          </td>
                          <td className="px-4 py-3 text-right font-mono text-xs text-neutral-300">
                            {resource.monthlyCost}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </section>
          </>
        )}
      </div>
    </Shell>
  );
}
