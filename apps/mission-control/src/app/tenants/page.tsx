"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { TableSkeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import type { Tenant } from "@/lib/api";
import { useApi } from "@/hooks/use-api";
import { Users } from "lucide-react";

export default function TenantsPage() {
  const apiClient = getApi();
  const { data: tenants, loading, error, refetch } = useApi<Tenant[]>(
    () => apiClient.tenants.list()
  );

  const activeCount = tenants
    ? tenants.filter((t) => t.status === "active").length
    : 0;

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Tenants</h1>
            {tenants && (
              <p className="mt-1 text-sm text-neutral-500">
                {tenants.length} tenants &middot; {activeCount} active
              </p>
            )}
          </div>
        </div>

        {loading ? (
          <TableSkeleton columns={7} rows={5} />
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : !tenants || tenants.length === 0 ? (
          <EmptyState
            title="No tenants"
            description="No tenants have been registered yet."
            icon={Users}
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
                    Plan
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    Apps
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    Databases
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    CPU (used / limit)
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    RAM (used / limit)
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    Status
                  </th>
                </tr>
              </thead>
              <tbody>
                {tenants.map((tenant) => (
                  <tr
                    key={tenant.name}
                    className="border-b border-border last:border-0 transition-colors hover:bg-surface-200"
                  >
                    <td className="px-4 py-3 font-medium text-white">
                      {tenant.name}
                    </td>
                    <td className="px-4 py-3">
                      <span
                        className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                          tenant.plan === "pro"
                            ? "bg-accent-500/10 text-accent-400"
                            : "bg-neutral-500/10 text-neutral-400"
                        }`}
                      >
                        {tenant.plan}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-neutral-300">
                      {tenant.apps}
                    </td>
                    <td className="px-4 py-3 text-neutral-300">
                      {tenant.databases}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-neutral-400">
                      {tenant.cpuUsed} / {tenant.cpuLimit} cores
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-neutral-400">
                      {tenant.ramUsed} / {tenant.ramLimit} GB
                    </td>
                    <td className="px-4 py-3">
                      <StatusBadge status={tenant.status} />
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
