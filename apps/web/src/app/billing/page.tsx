"use client";

import { Shell } from "@/components/shell";
import { PageHeaderSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { useApi } from "@/hooks/use-api";
import { getApi } from "@/lib/get-api";
import { Suspense, useState } from "react";
import { useSearchParams } from "next/navigation";
import type { BillingStatus, InvoiceRecord } from "@/lib/api";
import { IS_STANDALONE } from "@/lib/runtime-env";

const isStandalone = IS_STANDALONE;

const plans = [
  {
    tier: "free",
    name: "Free",
    price: 0,
    priceSuffix: "",
    highlight: false,
    features: [
      "1 app",
      "1 database (100 MB)",
      "1 GB storage",
      "Scale-to-zero (15 min idle)",
      "Community support",
    ],
  },
  {
    tier: "pro",
    name: "Pro",
    price: 29,
    priceSuffix: "/mo",
    highlight: false,
    features: [
      "5 apps, 3 databases (5 GB)",
      "10 GB storage, 2 Redis",
      "Always-on, custom domains",
      "Container registry",
      "Email support",
    ],
  },
  {
    tier: "team",
    name: "Team",
    price: 99,
    priceSuffix: "/seat/mo",
    highlight: false,
    features: [
      "20 apps, 10 databases (20 GB)",
      "100 GB storage, 5 Redis",
      "RBAC, preview deploys",
      "SSO (SAML/OIDC)",
      "Standard support",
    ],
  },
  {
    tier: "business",
    name: "Business",
    price: 149,
    priceSuffix: "/seat/mo",
    highlight: true,
    features: [
      "Unlimited apps & databases",
      "Dedicated infrastructure",
      "SSO, audit logs, compliance",
      "IP whitelisting, WAF config",
      "Priority support, 99.5% SLA",
    ],
  },
  {
    tier: "enterprise",
    name: "Enterprise",
    price: -1,
    priceSuffix: "",
    highlight: false,
    features: [
      "Everything in Business",
      "Full isolation (Temporal)",
      "Custom metrics & alerts",
      "Dedicated engineer",
      "99.9% SLA, custom terms",
    ],
  },
];

function formatCents(cents: number, currency: string): string {
  const amount = (cents / 100).toFixed(2);
  const symbol = currency === "eur" ? "\u20ac" : "$";
  return `${symbol}${amount}`;
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

function UsageBar({
  label,
  used,
  max,
}: {
  label: string;
  used: number;
  max: number;
}) {
  const pct = max > 0 ? Math.min((used / max) * 100, 100) : 0;
  const color =
    pct > 90 ? "bg-red-500" : pct > 70 ? "bg-yellow-500" : "bg-accent-500";

  return (
    <div>
      <div className="mb-1 flex items-center justify-between text-xs">
        <span className="text-neutral-400">{label}</span>
        <span className="text-neutral-500">
          {used} / {max}
        </span>
      </div>
      <div className="h-1.5 w-full rounded-full bg-surface-300">
        <div
          className={`h-1.5 rounded-full ${color} transition-all`}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  );
}

export default function BillingPage() {
  return (
    <Suspense
      fallback={
        <Shell>
          <div className="space-y-6">
            <PageHeaderSkeleton />
            <div className="rounded-lg border border-border bg-surface-100 p-5">
              <div className="animate-pulse rounded bg-surface-300 h-12 w-full" />
            </div>
          </div>
        </Shell>
      }
    >
      <BillingContent />
    </Suspense>
  );
}

function BillingContent() {
  const api = getApi();
  const searchParams = useSearchParams();
  const success = searchParams.get("success") === "true";
  const canceled = searchParams.get("canceled") === "true";

  const {
    data: status,
    loading,
    error,
    refetch,
  } = useApi<BillingStatus>(() => api.billing.getStatus(), []);

  const { data: invoiceData } = useApi<{
    items: InvoiceRecord[];
    total: number;
  }>(() => api.billing.listInvoices(), []);

  const [upgrading, setUpgrading] = useState<string | null>(null);
  const [showCancel, setShowCancel] = useState(false);
  const [exitReason, setExitReason] = useState("");
  const [exitDetails, setExitDetails] = useState("");
  const [canceling, setCanceling] = useState(false);

  const handleUpgrade = async (tier: string) => {
    if (!status?.stripe_enabled) return;
    setUpgrading(tier);
    try {
      const result = await api.billing.createCheckout(tier);
      window.location.href = result.checkout_url;
    } catch {
      setUpgrading(null);
    }
  };

  const handleManageSubscription = async () => {
    try {
      const result = await api.billing.createPortal();
      window.location.href = result.portal_url;
    } catch {
      // Portal creation failed
    }
  };

  const handleCancel = async () => {
    if (!exitReason) return;
    setCanceling(true);
    try {
      await api.exitSurvey.submit({ reason: exitReason, details: exitDetails });
      setShowCancel(false);
      setExitReason("");
      setExitDetails("");
      refetch();
    } catch {
      // Fallback: try direct cancel if exit survey endpoint fails
      try {
        await api.billing.cancel(false);
        setShowCancel(false);
        refetch();
      } catch {
        // Cancel failed
      }
    } finally {
      setCanceling(false);
    }
  };

  if (loading) {
    return (
      <Shell>
        <div className="space-y-6">
          <PageHeaderSkeleton />
          <div className="rounded-lg border border-border bg-surface-100 p-5">
            <div className="animate-pulse rounded bg-surface-300 h-12 w-full" />
          </div>
        </div>
      </Shell>
    );
  }

  if (error) {
    return (
      <Shell>
        <ErrorState message={error} onRetry={refetch} />
      </Shell>
    );
  }

  const invoices = invoiceData?.items || [];
  const currentTier = status?.tier || "free";
  const isActive = status?.billing_status === "active";
  const isPastDue = status?.billing_status === "past_due";

  if (isStandalone) {
    return (
      <Shell>
        <div className="space-y-6">
          <div>
            <h1 className="text-lg font-semibold text-white">Plan</h1>
            <p className="text-sm text-neutral-500">
              Self-hosted instance — no billing required
            </p>
          </div>

          <div className="rounded-lg border border-border bg-surface-100 p-5">
            <div className="flex items-center justify-between">
              <div>
                <div className="flex items-center gap-2">
                  <p className="text-sm font-medium text-white">Current Plan</p>
                  <span className="inline-flex items-center rounded-full bg-accent-500/10 px-2.5 py-0.5 text-xs font-medium text-accent-400 capitalize">
                    {currentTier}
                  </span>
                  <span className="inline-flex items-center rounded-full bg-green-500/10 px-2 py-0.5 text-xs text-green-400">
                    Self-Hosted
                  </span>
                </div>
                <p className="mt-1 text-xs text-neutral-500">
                  All features are available in your self-hosted deployment
                </p>
              </div>
            </div>
          </div>

          {status && (
            <section>
              <div className="mb-3">
                <h2 className="text-sm font-medium text-white">Resource Usage</h2>
              </div>
              <div className="rounded-lg border border-border bg-surface-100 p-5">
                <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                  <UsageBar label="Apps" used={status.usage.apps} max={status.limits.max_apps} />
                  <UsageBar label="Databases" used={status.usage.databases} max={status.limits.max_databases} />
                  <UsageBar label="Storage Buckets" used={status.usage.buckets} max={status.limits.max_buckets} />
                  <UsageBar label="Storage (MB)" used={status.usage.storage_mb} max={status.limits.max_storage_mb} />
                  <UsageBar label="Auth Users" used={status.usage.auth_users} max={status.limits.max_auth_users} />
                </div>
              </div>
            </section>
          )}
        </div>
      </Shell>
    );
  }

  return (
    <Shell>
      <div className="space-y-6">
        {/* Header */}
        <div>
          <h1 className="text-lg font-semibold text-white">Billing</h1>
          <p className="text-sm text-neutral-500">
            Manage your subscription, view invoices, and monitor usage
          </p>
        </div>

        {/* Success / Canceled banners */}
        {success && (
          <div className="rounded-lg border border-green-500/30 bg-green-500/10 px-4 py-3 text-sm text-green-400">
            Payment successful! Your plan has been upgraded.
          </div>
        )}
        {canceled && (
          <div className="rounded-lg border border-yellow-500/30 bg-yellow-500/10 px-4 py-3 text-sm text-yellow-400">
            Checkout was canceled. No changes were made to your plan.
          </div>
        )}
        {isPastDue && (
          <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-400">
            Your payment is past due. Please update your payment method to avoid
            losing access to paid features.
          </div>
        )}

        {/* Current Plan Card */}
        <section>
          <div className="rounded-lg border border-border bg-surface-100 p-5">
            <div className="flex items-center justify-between">
              <div>
                <div className="flex items-center gap-2">
                  <p className="text-sm font-medium text-white">Current Plan</p>
                  <span className="inline-flex items-center rounded-full bg-accent-500/10 px-2.5 py-0.5 text-xs font-medium text-accent-400 capitalize">
                    {currentTier}
                  </span>
                  {isActive && (
                    <span className="inline-flex items-center rounded-full bg-green-500/10 px-2 py-0.5 text-xs text-green-400">
                      Active
                    </span>
                  )}
                  {status?.cancel_at_period_end && (
                    <span className="inline-flex items-center rounded-full bg-yellow-500/10 px-2 py-0.5 text-xs text-yellow-400">
                      Cancels at period end
                    </span>
                  )}
                </div>
                {status?.period_end && (
                  <p className="mt-1 text-xs text-neutral-500">
                    Next billing date: {formatDate(status.period_end)}
                  </p>
                )}
              </div>
              <div className="text-right">
                <p className="text-2xl font-semibold text-white">
                  {status
                    ? formatCents(status.price_cents, status.currency)
                    : "--"}
                </p>
                <p className="text-xs text-neutral-500">per month</p>
              </div>
            </div>

            {/* Manage subscription button (for paying customers) */}
            {status?.stripe_enabled &&
              status.billing_status !== "none" &&
              currentTier !== "free" && (
                <div className="mt-4 flex gap-2">
                  <button
                    onClick={handleManageSubscription}
                    className="rounded-lg border border-border px-3 py-1.5 text-sm text-neutral-300 hover:text-white transition-colors"
                  >
                    Manage Subscription
                  </button>
                  {!status.cancel_at_period_end && (
                    <button
                      onClick={() => setShowCancel(true)}
                      className="rounded-lg border border-red-500/30 px-3 py-1.5 text-sm text-red-400 hover:bg-red-500/10 transition-colors"
                    >
                      Cancel Plan
                    </button>
                  )}
                </div>
              )}
          </div>
        </section>

        {/* Plan Comparison Grid */}
        <section>
          <div className="mb-3">
            <h2 className="text-sm font-medium text-white">Plans</h2>
          </div>
          <div className="grid gap-3 md:grid-cols-5">
            {plans.map((plan) => {
              const isCurrent = currentTier === plan.tier;
              const currentIdx = plans.findIndex((p) => p.tier === currentTier);
              const planIdx = plans.findIndex((p) => p.tier === plan.tier);
              const isDowngrade = currentIdx > planIdx;

              return (
                <div
                  key={plan.tier}
                  className={`rounded-lg border p-4 ${
                    isCurrent
                      ? "border-accent-500 bg-accent-500/5"
                      : plan.highlight
                        ? "border-amber-500/50 bg-amber-500/5"
                        : "border-border bg-surface-100"
                  }`}
                >
                  <div className="mb-3">
                    <div className="flex items-center gap-1.5">
                      <h3 className="text-sm font-medium text-white">{plan.name}</h3>
                      {plan.highlight && (
                        <span className="rounded bg-amber-500/20 px-1.5 py-0.5 text-[9px] font-bold text-amber-400">
                          BEST VALUE
                        </span>
                      )}
                    </div>
                    <p className="mt-1 text-xl font-semibold text-white">
                      {plan.price === 0 ? (
                        "Free"
                      ) : plan.price === -1 ? (
                        "Custom"
                      ) : (
                        <>
                          <span className="text-sm text-neutral-500">&euro;</span>
                          {plan.price}
                          <span className="text-[10px] font-normal text-neutral-500">
                            {plan.priceSuffix}
                          </span>
                        </>
                      )}
                    </p>
                  </div>

                  <ul className="mb-3 space-y-1.5">
                    {plan.features.map((f) => (
                      <li key={f} className="flex items-start gap-1.5 text-[11px] text-neutral-400">
                        <svg className="mt-0.5 h-2.5 w-2.5 flex-shrink-0 text-accent-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={3}>
                          <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                        </svg>
                        {f}
                      </li>
                    ))}
                  </ul>

                  {isCurrent ? (
                    <div className="rounded-md bg-accent-500/10 px-2 py-1.5 text-center text-[11px] font-medium text-accent-400">
                      Current Plan
                    </div>
                  ) : plan.tier === "free" ? null : plan.tier === "enterprise" ? (
                    <a
                      href="mailto:sales@freezenith.com"
                      className="block w-full rounded-md border border-border px-2 py-1.5 text-center text-[11px] text-neutral-400 hover:text-white transition-colors"
                    >
                      Contact Sales
                    </a>
                  ) : isDowngrade ? (
                    <div className="rounded-md bg-surface-300 px-2 py-1.5 text-center text-[11px] text-neutral-500">
                      Downgrade via portal
                    </div>
                  ) : (
                    <button
                      onClick={() => handleUpgrade(plan.tier)}
                      disabled={upgrading !== null}
                      className={`w-full rounded-md px-2 py-1.5 text-[11px] font-medium text-white transition-colors disabled:opacity-50 ${
                        plan.highlight
                          ? "bg-amber-500 hover:bg-amber-600"
                          : "bg-accent-500 hover:bg-accent-600"
                      }`}
                    >
                      {upgrading === plan.tier ? "Redirecting..." : `Upgrade`}
                    </button>
                  )}
                </div>
              );
            })}
          </div>
        </section>

        {/* Resource Usage */}
        {status && (
          <section>
            <div className="mb-3">
              <h2 className="text-sm font-medium text-white">Resource Usage</h2>
            </div>
            <div className="rounded-lg border border-border bg-surface-100 p-5">
              <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                <UsageBar
                  label="Apps"
                  used={status.usage.apps}
                  max={status.limits.max_apps}
                />
                <UsageBar
                  label="Databases"
                  used={status.usage.databases}
                  max={status.limits.max_databases}
                />
                <UsageBar
                  label="Storage Buckets"
                  used={status.usage.buckets}
                  max={status.limits.max_buckets}
                />
                <UsageBar
                  label="Storage (MB)"
                  used={status.usage.storage_mb}
                  max={status.limits.max_storage_mb}
                />
                <UsageBar
                  label="Auth Users"
                  used={status.usage.auth_users}
                  max={status.limits.max_auth_users}
                />
              </div>
            </div>
          </section>
        )}

        {/* Invoice History */}
        <section>
          <div className="mb-3">
            <h2 className="text-sm font-medium text-white">Invoices</h2>
          </div>
          {invoices.length === 0 ? (
            <div className="rounded-lg border border-border bg-surface-100 p-8 text-center">
              <p className="text-sm text-neutral-500">No invoices yet.</p>
            </div>
          ) : (
            <div className="overflow-hidden rounded-lg border border-border">
              <table className="w-full text-left text-sm">
                <thead className="bg-surface-200 text-xs text-neutral-500">
                  <tr>
                    <th className="px-4 py-2 font-medium">Date</th>
                    <th className="px-4 py-2 font-medium">Period</th>
                    <th className="px-4 py-2 font-medium">Amount</th>
                    <th className="px-4 py-2 font-medium">Status</th>
                    <th className="px-4 py-2 font-medium" />
                  </tr>
                </thead>
                <tbody className="divide-y divide-border bg-surface-100">
                  {invoices.map((inv) => (
                    <tr key={inv.id}>
                      <td className="px-4 py-3 text-neutral-300">
                        {formatDate(inv.created_at)}
                      </td>
                      <td className="px-4 py-3 text-neutral-400">
                        {formatDate(inv.period_start)} &ndash;{" "}
                        {formatDate(inv.period_end)}
                      </td>
                      <td className="px-4 py-3 text-white font-medium">
                        {formatCents(inv.amount_cents, inv.currency)}
                      </td>
                      <td className="px-4 py-3">
                        <span
                          className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${
                            inv.status === "paid"
                              ? "bg-green-500/10 text-green-400"
                              : inv.status === "open"
                                ? "bg-yellow-500/10 text-yellow-400"
                                : "bg-neutral-500/10 text-neutral-400"
                          }`}
                        >
                          {inv.status}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-right">
                        {inv.invoice_pdf && (
                          <a
                            href={inv.invoice_pdf}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-xs text-accent-500 hover:underline"
                          >
                            PDF
                          </a>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </div>

      {/* Exit Survey + Cancel Modal */}
      {showCancel && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
          <div className="mx-4 w-full max-w-md rounded-lg border border-border bg-surface-100 p-6">
            <h3 className="text-sm font-medium text-white">
              We&apos;re sorry to see you go
            </h3>
            <p className="mt-2 text-xs text-neutral-400">
              What&apos;s the main reason for canceling?
            </p>
            <div className="mt-4 space-y-2">
              {[
                { value: "too_expensive", label: "Too expensive" },
                { value: "missing_features", label: "Missing features I need" },
                { value: "found_alternative", label: "Found a better alternative" },
                { value: "not_using", label: "Not using it enough" },
                { value: "technical_issues", label: "Technical issues / reliability" },
                { value: "temporary", label: "Taking a break (temporary)" },
                { value: "other", label: "Other" },
              ].map((opt) => (
                <label
                  key={opt.value}
                  className={`flex items-center gap-3 rounded-lg border px-3 py-2 text-sm cursor-pointer transition-colors ${
                    exitReason === opt.value
                      ? "border-accent-500 bg-accent-500/10 text-accent-400"
                      : "border-border bg-surface-200 text-neutral-300 hover:border-neutral-600"
                  }`}
                >
                  <input
                    type="radio"
                    name="exit_reason"
                    value={opt.value}
                    checked={exitReason === opt.value}
                    onChange={() => setExitReason(opt.value)}
                    className="sr-only"
                  />
                  <span className={`h-3 w-3 rounded-full border ${exitReason === opt.value ? "border-accent-500 bg-accent-500" : "border-neutral-600"}`} />
                  {opt.label}
                </label>
              ))}
            </div>
            <div className="mt-4">
              <label className="text-xs text-neutral-500">Tell us more (optional)</label>
              <textarea
                value={exitDetails}
                onChange={(e) => setExitDetails(e.target.value)}
                placeholder="Any additional feedback..."
                rows={3}
                className="mt-1 w-full rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
            </div>
            <p className="mt-3 text-xs text-neutral-500">
              Your subscription will remain active until the end of the current billing period.
            </p>
            <div className="mt-4 flex justify-end gap-2">
              <button
                onClick={() => { setShowCancel(false); setExitReason(""); setExitDetails(""); }}
                className="rounded-md border border-border px-3 py-1.5 text-xs text-neutral-400 hover:text-white transition-colors"
              >
                Keep My Plan
              </button>
              <button
                onClick={handleCancel}
                disabled={!exitReason || canceling}
                className="rounded-md bg-red-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-red-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {canceling ? "Canceling..." : "Cancel Subscription"}
              </button>
            </div>
          </div>
        </div>
      )}
    </Shell>
  );
}
