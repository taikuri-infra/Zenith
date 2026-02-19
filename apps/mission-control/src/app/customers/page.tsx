"use client";

import { useState } from "react";
import Link from "next/link";
import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { TableSkeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import type { Customer } from "@/lib/api";
import { useApi } from "@/hooks/use-api";
import { Building2, Search } from "lucide-react";

export default function CustomersPage() {
  const apiClient = getApi();
  const { data: customers, loading, error, refetch } = useApi<Customer[]>(
    () => apiClient.customers.list()
  );
  const [search, setSearch] = useState("");

  const filtered = customers?.filter(
    (c) =>
      c.name.toLowerCase().includes(search.toLowerCase()) ||
      c.domain.toLowerCase().includes(search.toLowerCase())
  );

  const activeCount = filtered?.filter((c) => c.status === "active").length ?? 0;

  const formatPrice = (cents: number, currency: string) => {
    const symbol = currency === "EUR" ? "\u20AC" : "$";
    return `${symbol}${(cents / 100).toLocaleString()}`;
  };

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Customers</h1>
            {filtered && filtered.length > 0 && (
              <p className="mt-1 text-sm text-neutral-500">
                {filtered.length} customers &middot; {activeCount} active
              </p>
            )}
          </div>
          <Link
            href="/customers/new"
            className="rounded-lg bg-accent-600 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-accent-500"
          >
            + New Customer
          </Link>
        </div>

        {/* Search */}
        {!loading && !error && customers && customers.length > 0 && (
          <div className="relative">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-500" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search by name or domain..."
              className="w-full rounded-lg border border-border bg-surface-100 py-2 pl-10 pr-4 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
            />
          </div>
        )}

        {loading ? (
          <TableSkeleton columns={6} rows={5} />
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : !filtered || filtered.length === 0 ? (
          <EmptyState
            title="No customers"
            description={
              search
                ? "No customers match your search."
                : "No customers have been registered yet."
            }
            icon={Building2}
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
                    Domain
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    Plan
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    Cluster
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    Status
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    Created
                  </th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((customer) => (
                  <tr
                    key={customer.id}
                    className="border-b border-border last:border-0 transition-colors hover:bg-surface-200"
                  >
                    <td className="px-4 py-3">
                      <Link
                        href={`/customers/${customer.id}`}
                        className="font-medium text-white hover:text-accent-400 transition-colors"
                      >
                        {customer.name}
                      </Link>
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-neutral-400">
                      {customer.domain}
                    </td>
                    <td className="px-4 py-3">
                      <span
                        className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                          customer.plan?.name === "Enterprise"
                            ? "bg-amber-500/10 text-amber-400"
                            : customer.plan?.name === "Pro"
                            ? "bg-accent-500/10 text-accent-400"
                            : "bg-neutral-500/10 text-neutral-400"
                        }`}
                      >
                        {customer.plan?.name ?? "Unknown"}
                      </span>
                      {customer.plan && (
                        <span className="ml-2 text-xs text-neutral-600">
                          {formatPrice(customer.plan.priceCents, customer.plan.currency)}/mo
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <StatusBadge status={customer.clusterStatus === "running" ? "healthy" : customer.clusterStatus === "error" ? "error" : "warning"} />
                    </td>
                    <td className="px-4 py-3">
                      <StatusBadge status={customer.status === "active" ? "active" : "suspended"} />
                    </td>
                    <td className="px-4 py-3 text-xs text-neutral-500">
                      {new Date(customer.createdAt).toLocaleDateString()}
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
