"use client";

import { use, useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { ProgressBar } from "@/components/progress-bar";
import { ErrorState } from "@/components/error-state";
import { Skeleton } from "@/components/loading-skeleton";
import { Modal } from "@/components/modal";
import { getApi } from "@/lib/get-api";
import { isDemoMode } from "@/lib/get-api";
import type { Customer, CustomerUsage, UsageHistoryEntry } from "@/lib/api";
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
  MapPin,
  Check,
  Loader2,
  Circle,
  ArrowUpCircle,
  Scaling,
} from "lucide-react";
import Link from "next/link";

// ---------- Provisioning Stepper ----------

const provisioningSteps = [
  { key: "pending", label: "Account Created" },
  { key: "provisioning", label: "Provisioning" },
  { key: "installing", label: "Installing" },
  { key: "running", label: "Ready" },
] as const;

function ProvisioningStepper({ clusterStatus }: { clusterStatus: string }) {
  const statusIndex = provisioningSteps.findIndex(
    (s) => s.key === clusterStatus
  );
  const currentIndex = statusIndex >= 0 ? statusIndex : 0;

  return (
    <div className="rounded-lg border border-border bg-surface-100 p-4">
      <h2 className="mb-4 text-sm font-medium text-white">
        Cluster Provisioning
      </h2>
      <div className="flex items-center gap-2">
        {provisioningSteps.map((step, i) => {
          const isComplete = i < currentIndex;
          const isCurrent = i === currentIndex;
          const isLast = i === provisioningSteps.length - 1;

          return (
            <div key={step.key} className="flex items-center gap-2">
              <div className="flex flex-col items-center gap-1">
                <div
                  className={`flex h-8 w-8 items-center justify-center rounded-full border-2 ${
                    isComplete
                      ? "border-accent-500 bg-accent-500/20"
                      : isCurrent
                      ? "border-accent-500 bg-accent-500/10"
                      : "border-neutral-700 bg-surface-200"
                  }`}
                >
                  {isComplete ? (
                    <Check className="h-4 w-4 text-accent-400" />
                  ) : isCurrent ? (
                    <Loader2 className="h-4 w-4 animate-spin text-accent-400" />
                  ) : (
                    <Circle className="h-3 w-3 text-neutral-600" />
                  )}
                </div>
                <span
                  className={`text-xs whitespace-nowrap ${
                    isComplete || isCurrent
                      ? "text-white font-medium"
                      : "text-neutral-500"
                  }`}
                >
                  {step.label}
                </span>
              </div>
              {!isLast && (
                <div
                  className={`mb-5 h-0.5 w-12 ${
                    isComplete ? "bg-accent-500" : "bg-neutral-700"
                  }`}
                />
              )}
            </div>
          );
        })}
      </div>
      {clusterStatus === "error" && (
        <div className="mt-3 rounded-md bg-red-500/10 p-2 text-sm text-red-400">
          Cluster provisioning failed. Contact support or retry.
        </div>
      )}
    </div>
  );
}

// ---------- Cluster Info Panel ----------

function ClusterInfoPanel({
  customer,
  demo,
  onRefetch,
}: {
  customer: Customer;
  demo: boolean;
  onRefetch: () => void;
}) {
  const apiClient = getApi();
  const [showScaleModal, setShowScaleModal] = useState(false);
  const [showUpgradeModal, setShowUpgradeModal] = useState(false);
  const [scaleNodes, setScaleNodes] = useState(customer.clusterNodes);
  const [upgradeVersion, setUpgradeVersion] = useState("");

  const scaleMutation = useMutation(({ id, nodes }: { id: string; nodes: number }) =>
    apiClient.customers.scaleCluster(id, nodes)
  );
  const upgradeMutation = useMutation(({ id, version }: { id: string; version: string }) =>
    apiClient.customers.upgradeCluster(id, version)
  );

  const handleScale = async () => {
    try {
      await scaleMutation.execute({ id: customer.id, nodes: scaleNodes });
      setShowScaleModal(false);
      onRefetch();
    } catch {
      // error captured in mutation
    }
  };

  const handleUpgrade = async () => {
    try {
      await upgradeMutation.execute({ id: customer.id, version: upgradeVersion });
      setShowUpgradeModal(false);
      onRefetch();
    } catch {
      // error captured in mutation
    }
  };

  return (
    <>
      <div className="rounded-lg border border-border bg-surface-100 p-4">
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-medium text-white">Cluster</h2>
          {!demo && (
            <div className="flex gap-2">
              <button
                onClick={() => {
                  setScaleNodes(customer.clusterNodes);
                  setShowScaleModal(true);
                }}
                className="inline-flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1 text-xs text-neutral-400 transition-colors hover:text-white hover:border-neutral-600"
              >
                <Scaling className="h-3.5 w-3.5" />
                Scale
              </button>
              <button
                onClick={() => {
                  setUpgradeVersion("");
                  setShowUpgradeModal(true);
                }}
                className="inline-flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1 text-xs text-neutral-400 transition-colors hover:text-white hover:border-neutral-600"
              >
                <ArrowUpCircle className="h-3.5 w-3.5" />
                Upgrade
              </button>
            </div>
          )}
        </div>
        <div className="grid grid-cols-3 gap-4">
          <div className="rounded-md bg-surface-200 p-3">
            <div className="flex items-center gap-2 text-xs text-neutral-500">
              <Server className="h-3.5 w-3.5" />
              K8s Version
            </div>
            <div className="mt-1 font-mono text-sm text-white">
              {customer.clusterK8sVersion}
            </div>
          </div>
          <div className="rounded-md bg-surface-200 p-3">
            <div className="flex items-center gap-2 text-xs text-neutral-500">
              <Cpu className="h-3.5 w-3.5" />
              Nodes
            </div>
            <div className="mt-1 text-sm font-semibold text-white">
              {customer.clusterNodes}
            </div>
          </div>
          <div className="rounded-md bg-surface-200 p-3">
            <div className="flex items-center gap-2 text-xs text-neutral-500">
              <MapPin className="h-3.5 w-3.5" />
              Region
            </div>
            <div className="mt-1 text-sm text-white">
              {customer.clusterRegion}
            </div>
          </div>
        </div>
        <div className="mt-3 text-xs text-neutral-600">
          CAPI Cluster: <span className="font-mono text-neutral-400">{customer.capiClusterName}</span>
        </div>
      </div>

      {/* Scale Modal */}
      {showScaleModal && (
        <Modal title="Scale Cluster" onClose={() => setShowScaleModal(false)}>
          <p className="mb-3 text-sm text-neutral-300">
            Scale <span className="font-medium text-white">{customer.capiClusterName}</span> to a new node count.
          </p>
          <label className="mb-1 block text-xs text-neutral-400">
            Number of nodes
          </label>
          <input
            type="number"
            min={1}
            max={100}
            value={scaleNodes}
            onChange={(e) => setScaleNodes(Number(e.target.value))}
            className="mb-4 w-full rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
          />
          <div className="flex justify-end gap-3">
            <button
              onClick={() => setShowScaleModal(false)}
              className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleScale}
              disabled={scaleMutation.loading || scaleNodes < 1}
              className="rounded-lg bg-accent-600 px-4 py-2 text-sm font-medium text-white hover:bg-accent-500 transition-colors disabled:opacity-50"
            >
              {scaleMutation.loading ? "Scaling..." : "Scale Cluster"}
            </button>
          </div>
          {scaleMutation.error && (
            <p className="mt-2 text-xs text-red-400">{scaleMutation.error.message}</p>
          )}
        </Modal>
      )}

      {/* Upgrade Modal */}
      {showUpgradeModal && (
        <Modal title="Upgrade Cluster" onClose={() => setShowUpgradeModal(false)}>
          <p className="mb-3 text-sm text-neutral-300">
            Upgrade <span className="font-medium text-white">{customer.capiClusterName}</span> to a new Kubernetes version.
          </p>
          <label className="mb-1 block text-xs text-neutral-400">
            Target K8s version (e.g. v1.32.0)
          </label>
          <input
            type="text"
            value={upgradeVersion}
            onChange={(e) => setUpgradeVersion(e.target.value)}
            placeholder="v1.32.0"
            className="mb-4 w-full rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
          />
          <div className="flex justify-end gap-3">
            <button
              onClick={() => setShowUpgradeModal(false)}
              className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleUpgrade}
              disabled={upgradeMutation.loading || !upgradeVersion}
              className="rounded-lg bg-accent-600 px-4 py-2 text-sm font-medium text-white hover:bg-accent-500 transition-colors disabled:opacity-50"
            >
              {upgradeMutation.loading ? "Upgrading..." : "Upgrade Cluster"}
            </button>
          </div>
          {upgradeMutation.error && (
            <p className="mt-2 text-xs text-red-400">{upgradeMutation.error.message}</p>
          )}
        </Modal>
      )}
    </>
  );
}

// ---------- Resource Usage Section ----------

function ResourceUsageSection({ customerId }: { customerId: string }) {
  const apiClient = getApi();
  const { data: usage, loading, error } = useApi<CustomerUsage>(
    () => apiClient.customers.usage(customerId),
    [customerId]
  );

  if (loading) {
    return (
      <div className="rounded-lg border border-border bg-surface-100 p-4">
        <h2 className="mb-3 text-sm font-medium text-white">Resource Usage</h2>
        <div className="grid grid-cols-2 gap-4">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-10 w-full rounded" />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-border bg-surface-100 p-4">
        <h2 className="mb-3 text-sm font-medium text-white">Resource Usage</h2>
        <p className="text-sm text-neutral-500">Unable to load usage data.</p>
      </div>
    );
  }

  if (!usage) return null;

  const metrics = [
    { label: "CPU", used: `${usage.cpuCores}`, ceiling: `${usage.cpuCeiling} cores`, percent: usage.cpuPercent },
    { label: "RAM", used: `${usage.ramGb}`, ceiling: `${usage.ramCeiling} GB`, percent: usage.ramPercent },
    { label: "S3 Storage", used: `${usage.s3Tb}`, ceiling: `${usage.s3Ceiling} TB`, percent: usage.s3Percent },
    { label: "DB Storage", used: `${usage.dbStorageGb}`, ceiling: `${usage.dbCeiling} GB`, percent: usage.dbPercent },
    { label: "Volumes", used: `${usage.volumeGb}`, ceiling: `${usage.volCeiling} GB`, percent: usage.volPercent },
    { label: "Load Balancers", used: `${usage.lbCount}`, ceiling: `${usage.lbCeiling}`, percent: usage.lbPercent },
  ];

  return (
    <div className="rounded-lg border border-border bg-surface-100 p-4">
      <div className="mb-3 flex items-center justify-between">
        <h2 className="text-sm font-medium text-white">Resource Usage</h2>
        {usage.recordedAt && (
          <span className="text-xs text-neutral-600">
            Last updated: {new Date(usage.recordedAt).toLocaleString()}
          </span>
        )}
      </div>
      <div className="grid grid-cols-2 gap-4">
        {metrics.map((m) => (
          <div key={m.label}>
            <div className="mb-1 flex justify-between text-xs text-neutral-400">
              <span>{m.label}</span>
              <span>
                {m.used} / {m.ceiling}
              </span>
            </div>
            <ProgressBar percent={m.percent} label={`${m.percent}%`} />
          </div>
        ))}
      </div>
    </div>
  );
}

// ---------- Usage History Section ----------

function UsageHistorySection({ customerId }: { customerId: string }) {
  const apiClient = getApi();
  const { data: history, loading, error } = useApi<UsageHistoryEntry[]>(
    () => apiClient.customers.usageHistory(customerId, 30),
    [customerId]
  );

  if (loading) {
    return (
      <div className="rounded-lg border border-border bg-surface-100 p-4">
        <h2 className="mb-3 text-sm font-medium text-white">Usage History (30 days)</h2>
        <Skeleton className="h-48 w-full rounded" />
      </div>
    );
  }

  if (error || !history || history.length === 0) {
    return null;
  }

  return (
    <div className="rounded-lg border border-border bg-surface-100 p-4">
      <h2 className="mb-3 text-sm font-medium text-white">Usage History (30 days)</h2>
      <div className="overflow-hidden rounded-lg border border-border">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border bg-surface-200">
              <th className="px-3 py-2 text-left text-xs font-medium text-neutral-500">Date</th>
              <th className="px-3 py-2 text-right text-xs font-medium text-neutral-500">CPU Avg</th>
              <th className="px-3 py-2 text-right text-xs font-medium text-neutral-500">CPU Max</th>
              <th className="px-3 py-2 text-right text-xs font-medium text-neutral-500">RAM Avg</th>
              <th className="px-3 py-2 text-right text-xs font-medium text-neutral-500">RAM Max</th>
              <th className="px-3 py-2 text-right text-xs font-medium text-neutral-500">DB (GB)</th>
              <th className="px-3 py-2 text-right text-xs font-medium text-neutral-500">Vol (GB)</th>
              <th className="px-3 py-2 text-right text-xs font-medium text-neutral-500">LBs</th>
            </tr>
          </thead>
          <tbody>
            {history.slice(-10).map((entry) => (
              <tr key={entry.date} className="border-b border-border last:border-0">
                <td className="px-3 py-2 font-mono text-xs text-neutral-300">{entry.date}</td>
                <td className="px-3 py-2 text-right text-xs text-neutral-300">{entry.cpuAvg}</td>
                <td className="px-3 py-2 text-right text-xs text-neutral-300">{entry.cpuMax}</td>
                <td className="px-3 py-2 text-right text-xs text-neutral-300">{entry.ramAvg}</td>
                <td className="px-3 py-2 text-right text-xs text-neutral-300">{entry.ramMax}</td>
                <td className="px-3 py-2 text-right text-xs text-neutral-300">{entry.dbStorageGb}</td>
                <td className="px-3 py-2 text-right text-xs text-neutral-300">{entry.volumeGb}</td>
                <td className="px-3 py-2 text-right text-xs text-neutral-300">{entry.lbCount}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {history.length > 10 && (
        <p className="mt-2 text-xs text-neutral-600">
          Showing last 10 of {history.length} days
        </p>
      )}
    </div>
  );
}

// ---------- Main Page ----------

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

  // Auto-refresh during provisioning states
  useEffect(() => {
    if (
      !customer ||
      customer.clusterStatus === "running" ||
      customer.clusterStatus === "error"
    ) {
      return;
    }

    const interval = setInterval(() => {
      refetch();
    }, 5000);

    return () => clearInterval(interval);
  }, [customer, refetch]);

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

  const clusterStatusBadge = (status: string) => {
    switch (status) {
      case "running":
        return <StatusBadge status="healthy" label="Running" />;
      case "provisioning":
      case "installing":
        return <StatusBadge status="warning" label={status.charAt(0).toUpperCase() + status.slice(1)} />;
      case "error":
        return <StatusBadge status="error" label="Error" />;
      case "deleting":
        return <StatusBadge status="warning" label="Deleting" />;
      default:
        return <StatusBadge status="idle" label="Pending" />;
    }
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
              {clusterStatusBadge(customer.clusterStatus)}
            </div>

            {/* Cluster section */}
            {customer.clusterStatus !== "running" ? (
              <ProvisioningStepper clusterStatus={customer.clusterStatus} />
            ) : (
              <ClusterInfoPanel
                customer={customer}
                demo={demo}
                onRefetch={refetch}
              />
            )}

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

            {/* Resource Usage */}
            <ResourceUsageSection customerId={id} />

            {/* Usage History */}
            <UsageHistorySection customerId={id} />
          </>
        ) : null}
      </div>

      {/* Delete confirmation modal */}
      {showDeleteModal && (
        <Modal title="Delete Customer" onClose={() => setShowDeleteModal(false)}>
          <p className="mb-4 text-sm text-neutral-300">
            Are you sure you want to delete{" "}
            <span className="font-medium text-white">{customer?.name}</span>?
            This will also teardown the associated cluster. This action cannot be undone.
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
