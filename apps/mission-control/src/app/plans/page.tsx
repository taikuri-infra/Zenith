"use client";

import { useState } from "react";
import { Shell } from "@/components/shell";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { TableSkeleton } from "@/components/loading-skeleton";
import { Modal } from "@/components/modal";
import { getApi } from "@/lib/get-api";
import { isDemoMode } from "@/lib/get-api";
import type { Plan, CreatePlanInput } from "@/lib/api";
import { useApi } from "@/hooks/use-api";
import { useMutation } from "@/hooks/use-api";
import { CreditCard } from "lucide-react";

export default function PlansPage() {
  const apiClient = getApi();
  const demo = isDemoMode();
  const { data: plans, loading, error, refetch } = useApi<Plan[]>(
    () => apiClient.plans.list()
  );

  const [showModal, setShowModal] = useState(false);
  const [formName, setFormName] = useState("");
  const [formCpu, setFormCpu] = useState("4");
  const [formRam, setFormRam] = useState("8");
  const [formDb, setFormDb] = useState("10");
  const [formVolume, setFormVolume] = useState("50");
  const [formLb, setFormLb] = useState("1");
  const [formPrice, setFormPrice] = useState("9900");

  const createMutation = useMutation((input: CreatePlanInput) =>
    apiClient.plans.create(input)
  );

  const handleCreate = async () => {
    if (demo) return;
    try {
      await createMutation.execute({
        name: formName,
        cpuCores: parseInt(formCpu) || 0,
        ramGb: parseInt(formRam) || 0,
        dbStorageGb: parseInt(formDb) || 0,
        volumeGb: parseInt(formVolume) || 0,
        lbCount: parseInt(formLb) || 0,
        priceCents: parseInt(formPrice) || 0,
      });
      setShowModal(false);
      setFormName("");
      setFormCpu("4");
      setFormRam("8");
      setFormDb("10");
      setFormVolume("50");
      setFormLb("1");
      setFormPrice("9900");
      refetch();
    } catch {
      // error captured in mutation
    }
  };

  const formatPrice = (cents: number, currency: string) => {
    const symbol = currency === "EUR" ? "\u20AC" : "$";
    return `${symbol}${(cents / 100).toLocaleString()}`;
  };

  const inputClass =
    "w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none";

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Plans</h1>
            {plans && plans.length > 0 && (
              <p className="mt-1 text-sm text-neutral-500">
                {plans.length} plans &middot;{" "}
                {plans.filter((p) => p.active).length} active
              </p>
            )}
          </div>
          <button
            onClick={() => setShowModal(true)}
            className="rounded-lg bg-accent-600 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-accent-500"
          >
            + New Plan
          </button>
        </div>

        {loading ? (
          <TableSkeleton columns={8} rows={3} />
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : !plans || plans.length === 0 ? (
          <EmptyState
            title="No plans"
            description="No plans have been created yet."
            icon={CreditCard}
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
                    CPU
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    RAM
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    DB Storage
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    Volumes
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    LBs
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    Price
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                    Status
                  </th>
                </tr>
              </thead>
              <tbody>
                {plans.map((plan) => (
                  <tr
                    key={plan.id}
                    className="border-b border-border last:border-0 transition-colors hover:bg-surface-200"
                  >
                    <td className="px-4 py-3 font-medium text-white">
                      {plan.name}
                    </td>
                    <td className="px-4 py-3 text-neutral-300">
                      {plan.cpuCores} cores
                    </td>
                    <td className="px-4 py-3 text-neutral-300">
                      {plan.ramGb} GB
                    </td>
                    <td className="px-4 py-3 text-neutral-300">
                      {plan.dbStorageGb} GB
                    </td>
                    <td className="px-4 py-3 text-neutral-300">
                      {plan.volumeGb} GB
                    </td>
                    <td className="px-4 py-3 text-neutral-300">
                      {plan.lbCount}
                    </td>
                    <td className="px-4 py-3 font-medium text-accent-400">
                      {formatPrice(plan.priceCents, plan.currency)}
                      <span className="text-xs font-normal text-neutral-500">
                        /{plan.billingCycle}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <span
                        className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                          plan.active
                            ? "bg-emerald-500/10 text-emerald-400"
                            : "bg-neutral-500/10 text-neutral-400"
                        }`}
                      >
                        {plan.active ? "Active" : "Inactive"}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {showModal && (
        <Modal title="Create Plan" onClose={() => setShowModal(false)}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              handleCreate();
            }}
            className="space-y-4"
          >
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">
                Plan Name *
              </label>
              <input
                type="text"
                value={formName}
                onChange={(e) => setFormName(e.target.value)}
                placeholder="Business"
                required
                className={inputClass}
              />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="mb-1 block text-xs font-medium text-neutral-400">
                  CPU Cores *
                </label>
                <input
                  type="number"
                  value={formCpu}
                  onChange={(e) => setFormCpu(e.target.value)}
                  min="1"
                  required
                  className={inputClass}
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-neutral-400">
                  RAM (GB) *
                </label>
                <input
                  type="number"
                  value={formRam}
                  onChange={(e) => setFormRam(e.target.value)}
                  min="1"
                  required
                  className={inputClass}
                />
              </div>
            </div>
            <div className="grid grid-cols-3 gap-3">
              <div>
                <label className="mb-1 block text-xs font-medium text-neutral-400">
                  DB Storage (GB)
                </label>
                <input
                  type="number"
                  value={formDb}
                  onChange={(e) => setFormDb(e.target.value)}
                  min="0"
                  className={inputClass}
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-neutral-400">
                  Volume (GB)
                </label>
                <input
                  type="number"
                  value={formVolume}
                  onChange={(e) => setFormVolume(e.target.value)}
                  min="0"
                  className={inputClass}
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-neutral-400">
                  Load Balancers
                </label>
                <input
                  type="number"
                  value={formLb}
                  onChange={(e) => setFormLb(e.target.value)}
                  min="0"
                  className={inputClass}
                />
              </div>
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">
                Price (cents/month) *
              </label>
              <input
                type="number"
                value={formPrice}
                onChange={(e) => setFormPrice(e.target.value)}
                min="1"
                required
                className={inputClass}
              />
              <p className="mt-1 text-xs text-neutral-600">
                {parseInt(formPrice) > 0
                  ? `= \u20AC${(parseInt(formPrice) / 100).toLocaleString()}/month`
                  : "Enter price in cents"}
              </p>
            </div>

            {demo && (
              <p className="rounded-md bg-amber-500/10 px-3 py-2 text-xs text-amber-400">
                Creating plans is not available in demo mode.
              </p>
            )}

            {createMutation.error && (
              <p className="rounded-md bg-red-500/10 px-3 py-2 text-xs text-red-400">
                {createMutation.error.message}
              </p>
            )}

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
                disabled={demo || createMutation.loading}
                className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:cursor-not-allowed disabled:opacity-50"
              >
                {createMutation.loading ? "Creating..." : "Create Plan"}
              </button>
            </div>
          </form>
        </Modal>
      )}
    </Shell>
  );
}
