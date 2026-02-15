"use client";

import { useState, useEffect } from "react";
import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { TableSkeleton } from "@/components/loading-skeleton";
import { Modal } from "@/components/modal";
import { getApi } from "@/lib/get-api";
import type { Tenant } from "@/lib/api";
import { useApi } from "@/hooks/use-api";
import { Users } from "lucide-react";

export default function TenantsPage() {
  const apiClient = getApi();
  const { data: tenants, loading, error, refetch } = useApi<Tenant[]>(
    () => apiClient.tenants.list()
  );

  const [localTenants, setLocalTenants] = useState<Tenant[]>([]);
  const [showModal, setShowModal] = useState(false);
  const [formName, setFormName] = useState("");
  const [formPlan, setFormPlan] = useState<"starter" | "pro">("starter");
  const [formEmail, setFormEmail] = useState("");

  useEffect(() => {
    if (tenants) {
      setLocalTenants(tenants);
    }
  }, [tenants]);

  const activeCount = localTenants.filter((t) => t.status === "active").length;

  const handleCreate = () => {
    const newTenant: Tenant = {
      name: formName || "new-tenant",
      plan: formPlan,
      apps: 0,
      databases: 0,
      cpuUsed: "0.0",
      cpuLimit: formPlan === "starter" ? "4.0" : formPlan === "pro" ? "16.0" : "64.0",
      ramUsed: "0.0",
      ramLimit: formPlan === "starter" ? "8" : formPlan === "pro" ? "32" : "128",
      status: "active",
    };
    setLocalTenants((prev) => [...prev, newTenant]);
    setShowModal(false);
    setFormName("");
    setFormPlan("starter");
    setFormEmail("");
  };

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Tenants</h1>
            {localTenants.length > 0 && (
              <p className="mt-1 text-sm text-neutral-500">
                {localTenants.length} tenants &middot; {activeCount} active
              </p>
            )}
          </div>
          <button
            onClick={() => setShowModal(true)}
            className="rounded-lg bg-accent-600 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-accent-500"
          >
            + Create Tenant
          </button>
        </div>

        {loading ? (
          <TableSkeleton columns={7} rows={5} />
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : localTenants.length === 0 ? (
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
                {localTenants.map((tenant) => (
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

      {showModal && (
        <Modal title="Create Tenant" onClose={() => setShowModal(false)}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              handleCreate();
            }}
            className="space-y-4"
          >
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">
                Tenant Name
              </label>
              <input
                type="text"
                value={formName}
                onChange={(e) => setFormName(e.target.value)}
                placeholder="acme-corp"
                required
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">
                Plan
              </label>
              <select
                value={formPlan}
                onChange={(e) => setFormPlan(e.target.value as "starter" | "pro")}
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
              >
                <option value="starter">Starter</option>
                <option value="pro">Pro</option>
              </select>
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">
                Owner Email
              </label>
              <input
                type="email"
                value={formEmail}
                onChange={(e) => setFormEmail(e.target.value)}
                placeholder="admin@acme.com"
                required
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
            </div>
            <div className="flex justify-end gap-3 pt-2">
              <button
                type="button"
                onClick={() => setShowModal(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors"
              >
                Create Tenant
              </button>
            </div>
          </form>
        </Modal>
      )}
    </Shell>
  );
}
