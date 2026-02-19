"use client";

import { use, useState } from "react";
import { useRouter } from "next/navigation";
import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { ProgressBar } from "@/components/progress-bar";
import { ErrorState } from "@/components/error-state";
import { Skeleton } from "@/components/loading-skeleton";
import { Modal } from "@/components/modal";
import { getApi } from "@/lib/get-api";
import { isDemoMode } from "@/lib/get-api";
import type { Customer } from "@/lib/api";
import { useApi } from "@/hooks/use-api";
import { useMutation } from "@/hooks/use-api";
import {
  ArrowLeft,
  Globe,
  Mail,
  User,
  Calendar,
  CreditCard,
  Cpu,
  HardDrive,
  Database,
  Server,
} from "lucide-react";
import Link from "next/link";

export default function CustomerDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const router = useRouter();
  const apiClient = getApi();
  const demo = isDemoMode();

  const { data: customer, loading, error, refetch } = useApi<Customer>(
    () => apiClient.customers.get(id),
    [id]
  );

  const [showDeleteModal, setShowDeleteModal] = useState(false);

  const suspendMutation = useMutation((customerId: string) =>
    apiClient.customers.suspend(customerId)
  );
  const activateMutation = useMutation((customerId: string) =>
    apiClient.customers.activate(customerId)
  );
  const deleteMutation = useMutation((customerId: string) =>
    apiClient.customers.delete(customerId)
  );

  const handleSuspend = async () => {
    try {
      await suspendMutation.execute(id);
      refetch();
    } catch {
      // error is captured in mutation
    }
  };

  const handleActivate = async () => {
    try {
      await activateMutation.execute(id);
      refetch();
    } catch {
      // error is captured in mutation
    }
  };

  const handleDelete = async () => {
    try {
      await deleteMutation.execute(id);
      router.push("/customers");
    } catch {
      // error is captured in mutation
    }
  };

  const formatPrice = (cents: number, currency: string) => {
    const symbol = currency === "EUR" ? "\u20AC" : "$";
    return `${symbol}${(cents / 100).toLocaleString()}`;
  };

  return (
    <Shell>
      <div className="space-y-6">
        {/* Back link */}
        <Link
          href="/customers"
          className="inline-flex items-center gap-1.5 text-sm text-neutral-500 hover:text-white transition-colors"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Customers
        </Link>

        {loading ? (
          <div className="space-y-4">
            <Skeleton className="h-8 w-64 rounded" />
            <Skeleton className="h-48 w-full rounded-lg" />
          </div>
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : customer ? (
          <>
            {/* Header */}
            <div className="flex items-start justify-between">
              <div>
                <h1 className="text-lg font-semibold text-white">
                  {customer.name}
                </h1>
                <p className="mt-1 flex items-center gap-2 text-sm text-neutral-400">
                  <Globe className="h-3.5 w-3.5" />
                  {customer.domain}
                </p>
              </div>
              <div className="flex items-center gap-2">
                {!demo && (
                  <>
                    {customer.status === "active" ? (
                      <button
                        onClick={handleSuspend}
                        disabled={suspendMutation.loading}
                        className="rounded-lg border border-amber-500/30 px-3 py-1.5 text-sm text-amber-400 transition-colors hover:bg-amber-500/10 disabled:opacity-50"
                      >
                        {suspendMutation.loading ? "Suspending..." : "Suspend"}
                      </button>
                    ) : (
                      <button
                        onClick={handleActivate}
                        disabled={activateMutation.loading}
                        className="rounded-lg border border-accent-500/30 px-3 py-1.5 text-sm text-accent-400 transition-colors hover:bg-accent-500/10 disabled:opacity-50"
                      >
                        {activateMutation.loading ? "Activating..." : "Activate"}
                      </button>
                    )}
                    <button
                      onClick={() => setShowDeleteModal(true)}
                      className="rounded-lg border border-red-500/30 px-3 py-1.5 text-sm text-red-400 transition-colors hover:bg-red-500/10"
                    >
                      Delete
                    </button>
                  </>
                )}
              </div>
            </div>

            {/* Status badges */}
            <div className="flex items-center gap-3">
              <StatusBadge status={customer.status === "active" ? "active" : "suspended"} />
              <span className="text-xs text-neutral-500">Cluster:</span>
              <StatusBadge
                status={
                  customer.clusterStatus === "running"
                    ? "healthy"
                    : customer.clusterStatus === "error"
                    ? "error"
                    : "warning"
                }
              />
            </div>

            {/* Info grid */}
            <div className="grid grid-cols-2 gap-6">
              {/* Profile */}
              <div className="rounded-lg border border-border bg-surface-100 p-4">
                <h2 className="mb-3 text-sm font-medium text-white">
                  Customer Info
                </h2>
                <div className="space-y-3">
                  <div className="flex items-center gap-2.5 text-sm">
                    <User className="h-4 w-4 text-neutral-500" />
                    <span className="text-neutral-400">Contact:</span>
                    <span className="text-white">{customer.contactName || "Not set"}</span>
                  </div>
                  <div className="flex items-center gap-2.5 text-sm">
                    <Mail className="h-4 w-4 text-neutral-500" />
                    <span className="text-neutral-400">Email:</span>
                    <span className="text-white">{customer.contactEmail}</span>
                  </div>
                  <div className="flex items-center gap-2.5 text-sm">
                    <Globe className="h-4 w-4 text-neutral-500" />
                    <span className="text-neutral-400">Domain:</span>
                    <span className="font-mono text-white">{customer.domain}</span>
                  </div>
                  <div className="flex items-center gap-2.5 text-sm">
                    <Calendar className="h-4 w-4 text-neutral-500" />
                    <span className="text-neutral-400">Created:</span>
                    <span className="text-white">
                      {new Date(customer.createdAt).toLocaleDateString()}
                    </span>
                  </div>
                  {customer.notes && (
                    <div className="mt-2 rounded-md bg-surface-200 p-2.5 text-sm text-neutral-300">
                      {customer.notes}
                    </div>
                  )}
                </div>
              </div>

              {/* Plan */}
              {customer.plan && (
                <div className="rounded-lg border border-border bg-surface-100 p-4">
                  <h2 className="mb-3 text-sm font-medium text-white">
                    Plan: {customer.plan.name}
                  </h2>
                  <div className="mb-3 text-2xl font-semibold text-accent-400">
                    {formatPrice(customer.plan.priceCents, customer.plan.currency)}
                    <span className="text-sm font-normal text-neutral-500">
                      /{customer.plan.billingCycle}
                    </span>
                  </div>
                  <div className="space-y-2.5">
                    <div className="flex items-center gap-2.5 text-sm">
                      <Cpu className="h-4 w-4 text-neutral-500" />
                      <span className="text-neutral-400">CPU:</span>
                      <span className="text-white">
                        {customer.plan.cpuCores} cores
                      </span>
                    </div>
                    <div className="flex items-center gap-2.5 text-sm">
                      <Server className="h-4 w-4 text-neutral-500" />
                      <span className="text-neutral-400">RAM:</span>
                      <span className="text-white">
                        {customer.plan.ramGb} GB
                      </span>
                    </div>
                    <div className="flex items-center gap-2.5 text-sm">
                      <Database className="h-4 w-4 text-neutral-500" />
                      <span className="text-neutral-400">DB Storage:</span>
                      <span className="text-white">
                        {customer.plan.dbStorageGb} GB
                      </span>
                    </div>
                    <div className="flex items-center gap-2.5 text-sm">
                      <HardDrive className="h-4 w-4 text-neutral-500" />
                      <span className="text-neutral-400">Volume:</span>
                      <span className="text-white">
                        {customer.plan.volumeGb} GB
                      </span>
                    </div>
                    <div className="flex items-center gap-2.5 text-sm">
                      <CreditCard className="h-4 w-4 text-neutral-500" />
                      <span className="text-neutral-400">Load Balancers:</span>
                      <span className="text-white">
                        {customer.plan.lbCount}
                      </span>
                    </div>
                  </div>
                </div>
              )}
            </div>

            {/* Resource usage placeholder */}
            <div className="rounded-lg border border-border bg-surface-100 p-4">
              <h2 className="mb-3 text-sm font-medium text-white">
                Resource Usage
              </h2>
              {customer.plan ? (
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <div className="mb-1 flex justify-between text-xs text-neutral-400">
                      <span>CPU</span>
                      <span>
                        0 / {customer.plan.cpuCores} cores
                      </span>
                    </div>
                    <ProgressBar percent={0} label="0%" />
                  </div>
                  <div>
                    <div className="mb-1 flex justify-between text-xs text-neutral-400">
                      <span>RAM</span>
                      <span>
                        0 / {customer.plan.ramGb} GB
                      </span>
                    </div>
                    <ProgressBar percent={0} label="0%" />
                  </div>
                  <div>
                    <div className="mb-1 flex justify-between text-xs text-neutral-400">
                      <span>DB Storage</span>
                      <span>
                        0 / {customer.plan.dbStorageGb} GB
                      </span>
                    </div>
                    <ProgressBar percent={0} label="0%" />
                  </div>
                  <div>
                    <div className="mb-1 flex justify-between text-xs text-neutral-400">
                      <span>Volumes</span>
                      <span>
                        0 / {customer.plan.volumeGb} GB
                      </span>
                    </div>
                    <ProgressBar percent={0} label="0%" />
                  </div>
                </div>
              ) : (
                <p className="text-sm text-neutral-500">
                  No plan assigned.
                </p>
              )}
              <p className="mt-3 text-xs text-neutral-600">
                Resource usage data will be available once the cluster is provisioned.
              </p>
            </div>
          </>
        ) : null}
      </div>

      {/* Delete confirmation modal */}
      {showDeleteModal && (
        <Modal title="Delete Customer" onClose={() => setShowDeleteModal(false)}>
          <p className="mb-4 text-sm text-neutral-300">
            Are you sure you want to delete{" "}
            <span className="font-medium text-white">{customer?.name}</span>?
            This action cannot be undone.
          </p>
          <div className="flex justify-end gap-3">
            <button
              onClick={() => setShowDeleteModal(false)}
              className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleDelete}
              disabled={deleteMutation.loading}
              className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-500 transition-colors disabled:opacity-50"
            >
              {deleteMutation.loading ? "Deleting..." : "Delete Customer"}
            </button>
          </div>
          {deleteMutation.error && (
            <p className="mt-2 text-xs text-red-400">
              {deleteMutation.error.message}
            </p>
          )}
        </Modal>
      )}
    </Shell>
  );
}
