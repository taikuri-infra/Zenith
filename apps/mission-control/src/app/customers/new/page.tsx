"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Shell } from "@/components/shell";
import { ErrorState } from "@/components/error-state";
import { Skeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import { isDemoMode } from "@/lib/get-api";
import type { Plan, CreateCustomerInput } from "@/lib/api";
import { useApi } from "@/hooks/use-api";
import { useMutation } from "@/hooks/use-api";
import { ArrowLeft, Check } from "lucide-react";
import Link from "next/link";

type Step = "info" | "plan" | "review";

export default function NewCustomerPage() {
  const router = useRouter();
  const apiClient = getApi();
  const demo = isDemoMode();

  const { data: plans, loading: plansLoading, error: plansError } = useApi<Plan[]>(
    () => apiClient.plans.list()
  );

  const createMutation = useMutation((input: CreateCustomerInput) =>
    apiClient.customers.create(input)
  );

  const [step, setStep] = useState<Step>("info");
  const [name, setName] = useState("");
  const [domain, setDomain] = useState("");
  const [contactEmail, setContactEmail] = useState("");
  const [contactName, setContactName] = useState("");
  const [selectedPlanId, setSelectedPlanId] = useState("");

  const selectedPlan = plans?.find((p) => p.id === selectedPlanId);

  const canProceedInfo = name.trim() && domain.trim() && contactEmail.trim();
  const canProceedPlan = !!selectedPlanId;

  const formatPrice = (cents: number, currency: string) => {
    const symbol = currency === "EUR" ? "\u20AC" : "$";
    return `${symbol}${(cents / 100).toLocaleString()}`;
  };

  const handleCreate = async () => {
    if (demo) return;
    try {
      const customer = await createMutation.execute({
        name: name.trim(),
        domain: domain.trim(),
        planId: selectedPlanId,
        contactEmail: contactEmail.trim(),
        contactName: contactName.trim(),
      });
      router.push(`/customers/${customer.id}`);
    } catch {
      // error captured in mutation
    }
  };

  const inputClass =
    "w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none";

  return (
    <Shell>
      <div className="mx-auto max-w-2xl space-y-6">
        {/* Back link */}
        <Link
          href="/customers"
          className="inline-flex items-center gap-1.5 text-sm text-neutral-500 hover:text-white transition-colors"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Customers
        </Link>

        <h1 className="text-lg font-semibold text-white">New Customer</h1>

        {/* Step indicator */}
        <div className="flex items-center gap-2">
          {(["info", "plan", "review"] as Step[]).map((s, i) => (
            <div key={s} className="flex items-center gap-2">
              {i > 0 && (
                <div
                  className={`h-px w-8 ${
                    (["info", "plan", "review"] as Step[]).indexOf(step) >= i
                      ? "bg-accent-500"
                      : "bg-border"
                  }`}
                />
              )}
              <div
                className={`flex h-7 w-7 items-center justify-center rounded-full text-xs font-medium ${
                  step === s
                    ? "bg-accent-600 text-white"
                    : (["info", "plan", "review"] as Step[]).indexOf(step) > i
                    ? "bg-accent-600/20 text-accent-400"
                    : "bg-surface-200 text-neutral-500"
                }`}
              >
                {(["info", "plan", "review"] as Step[]).indexOf(step) > i ? (
                  <Check className="h-3.5 w-3.5" />
                ) : (
                  i + 1
                )}
              </div>
              <span
                className={`text-xs ${
                  step === s ? "text-white" : "text-neutral-500"
                }`}
              >
                {s === "info"
                  ? "Company Info"
                  : s === "plan"
                  ? "Select Plan"
                  : "Review"}
              </span>
            </div>
          ))}
        </div>

        {/* Step: Company Info */}
        {step === "info" && (
          <div className="rounded-lg border border-border bg-surface-100 p-6">
            <h2 className="mb-4 text-sm font-medium text-white">
              Company Information
            </h2>
            <div className="space-y-4">
              <div>
                <label className="mb-1 block text-xs font-medium text-neutral-400">
                  Company Name *
                </label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Acme Corp"
                  className={inputClass}
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-neutral-400">
                  Domain *
                </label>
                <input
                  type="text"
                  value={domain}
                  onChange={(e) => setDomain(e.target.value)}
                  placeholder="acme-corp.com"
                  className={inputClass}
                />
                <p className="mt-1 text-xs text-neutral-600">
                  The customer&apos;s domain. Will be used for ms.{domain || "example.com"} and
                  cloud.{domain || "example.com"}.
                </p>
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-neutral-400">
                  Contact Email *
                </label>
                <input
                  type="email"
                  value={contactEmail}
                  onChange={(e) => setContactEmail(e.target.value)}
                  placeholder="admin@acme-corp.com"
                  className={inputClass}
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-neutral-400">
                  Contact Name
                </label>
                <input
                  type="text"
                  value={contactName}
                  onChange={(e) => setContactName(e.target.value)}
                  placeholder="Jane Doe"
                  className={inputClass}
                />
              </div>
              <div className="flex justify-end pt-2">
                <button
                  onClick={() => setStep("plan")}
                  disabled={!canProceedInfo}
                  className="rounded-lg bg-accent-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-accent-500 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  Next: Select Plan
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Step: Select Plan */}
        {step === "plan" && (
          <div className="rounded-lg border border-border bg-surface-100 p-6">
            <h2 className="mb-4 text-sm font-medium text-white">
              Select a Plan
            </h2>
            {plansLoading ? (
              <div className="space-y-3">
                <Skeleton className="h-20 w-full rounded-lg" />
                <Skeleton className="h-20 w-full rounded-lg" />
                <Skeleton className="h-20 w-full rounded-lg" />
              </div>
            ) : plansError ? (
              <ErrorState error={plansError} />
            ) : (
              <div className="space-y-3">
                {plans
                  ?.filter((p) => p.active)
                  .map((plan) => (
                    <button
                      key={plan.id}
                      onClick={() => setSelectedPlanId(plan.id)}
                      className={`w-full rounded-lg border p-4 text-left transition-colors ${
                        selectedPlanId === plan.id
                          ? "border-accent-500 bg-accent-600/5"
                          : "border-border hover:border-neutral-600"
                      }`}
                    >
                      <div className="flex items-center justify-between">
                        <div>
                          <span className="text-sm font-medium text-white">
                            {plan.name}
                          </span>
                          <span className="ml-3 text-xs text-neutral-500">
                            {plan.cpuCores} CPU &middot; {plan.ramGb} GB RAM
                            &middot; {plan.dbStorageGb} GB DB
                            {plan.s3Tb > 0 && ` \u00b7 ${plan.s3Tb} TB S3`}
                          </span>
                        </div>
                        <span className="text-sm font-semibold text-accent-400">
                          {formatPrice(plan.priceCents, plan.currency)}
                          <span className="text-xs font-normal text-neutral-500">
                            /mo
                          </span>
                        </span>
                      </div>
                    </button>
                  ))}
              </div>
            )}
            <div className="mt-4 flex justify-between">
              <button
                onClick={() => setStep("info")}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Back
              </button>
              <button
                onClick={() => setStep("review")}
                disabled={!canProceedPlan}
                className="rounded-lg bg-accent-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-accent-500 disabled:cursor-not-allowed disabled:opacity-50"
              >
                Next: Review
              </button>
            </div>
          </div>
        )}

        {/* Step: Review */}
        {step === "review" && (
          <div className="rounded-lg border border-border bg-surface-100 p-6">
            <h2 className="mb-4 text-sm font-medium text-white">
              Review &amp; Create
            </h2>
            <div className="space-y-3 text-sm">
              <div className="flex justify-between border-b border-border pb-2">
                <span className="text-neutral-400">Company</span>
                <span className="text-white">{name}</span>
              </div>
              <div className="flex justify-between border-b border-border pb-2">
                <span className="text-neutral-400">Domain</span>
                <span className="font-mono text-white">{domain}</span>
              </div>
              <div className="flex justify-between border-b border-border pb-2">
                <span className="text-neutral-400">Contact</span>
                <span className="text-white">
                  {contactName ? `${contactName} <${contactEmail}>` : contactEmail}
                </span>
              </div>
              <div className="flex justify-between border-b border-border pb-2">
                <span className="text-neutral-400">Plan</span>
                <span className="text-white">
                  {selectedPlan?.name}{" "}
                  {selectedPlan && (
                    <span className="text-accent-400">
                      ({formatPrice(selectedPlan.priceCents, selectedPlan.currency)}
                      /mo)
                    </span>
                  )}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-neutral-400">Endpoints</span>
                <div className="text-right font-mono text-xs text-neutral-300">
                  <div>ms.{domain}</div>
                  <div>cloud.{domain}</div>
                </div>
              </div>
            </div>

            {demo && (
              <p className="mt-4 rounded-md bg-amber-500/10 px-3 py-2 text-xs text-amber-400">
                Creating customers is not available in demo mode.
              </p>
            )}

            {createMutation.error && (
              <p className="mt-4 rounded-md bg-red-500/10 px-3 py-2 text-xs text-red-400">
                {createMutation.error.message}
              </p>
            )}

            <div className="mt-4 flex justify-between">
              <button
                onClick={() => setStep("plan")}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Back
              </button>
              <button
                onClick={handleCreate}
                disabled={demo || createMutation.loading}
                className="rounded-lg bg-accent-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-accent-500 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {createMutation.loading ? "Creating..." : "Create Customer"}
              </button>
            </div>
          </div>
        )}
      </div>
    </Shell>
  );
}
