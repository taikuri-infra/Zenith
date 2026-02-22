"use client";

import { Shell } from "@/components/shell";
import { PageHeaderSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { useApi } from "@/hooks/use-api";
import { getApi } from "@/lib/get-api";
import { Suspense, useState } from "react";
import { useSearchParams } from "next/navigation";
import type { BillingStatus, InvoiceRecord } from "@/lib/api";

const isStandalone = process.env.NEXT_PUBLIC_ZENITH_MODE !== "saas";

const plans = [
  {
    tier: "free",
    name: "Free",
    price: 0,
    features: [
      "1 app",
      "1 database (500 MB)",
      "1 GB storage",
      "1,000 auth users",
      "Scale-to-zero (15 min idle)",
      "Community support",
    ],
  },
  {
    tier: "pro",
    name: "Pro",
    price: 29,
    features: [
      "5 apps",
      "3 databases (5 GB each)",
      "10 GB storage",
      "10,000 auth users",
      "Always-on, custom domains",
      "Daily backups, email support",
    ],
  },
  {
    tier: "team",
    name: "Team",
    price: 199,
    features: [
      "20 apps",
      "10 databases (20 GB each)",
      "100 GB storage",
      "100,000 auth users",
      "SSO, RBAC, preview deploys",
      "Priority support, SLA",
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
    try {
      await api.billing.cancel(false);
      setShowCancel(false);
      refetch();
    } catch {
      // Cancel failed
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
          <div className="grid gap-4 md:grid-cols-3">
            {plans.map((plan) => {
              const isCurrent = currentTier === plan.tier;
              const isDowngrade =
                plans.findIndex((p) => p.tier === currentTier) >
                plans.findIndex((p) => p.tier === plan.tier);

              return (
                <div
                  key={plan.tier}
                  className={`rounded-lg border p-5 ${
                    isCurrent
                      ? "border-accent-500 bg-accent-500/5"
                      : "border-border bg-surface-100"
                  }`}
                >
                  <div className="mb-4">
                    <h3 className="text-sm font-medium text-white">
                      {plan.name}
                    </h3>
                    <p className="mt-1 text-2xl font-semibold text-white">
                      {plan.price === 0 ? (
                        "Free"
                      ) : (
                        <>
                          <span className="text-lg text-neutral-500">
                            &euro;
                          </span>
                          {plan.price}
                          <span className="text-sm font-normal text-neutral-500">
                            /mo
                          </span>
                        </>
                      )}
                    </p>
                  </div>

                  <ul className="mb-4 space-y-2">
                    {plan.features.map((f) => (
                      <li
                        key={f}
                        className="flex items-start gap-2 text-xs text-neutral-400"
                      >
                        <svg
                          className="mt-0.5 h-3 w-3 flex-shrink-0 text-accent-500"
                          fill="none"
                          viewBox="0 0 24 24"
                          stroke="currentColor"
                          strokeWidth={3}
                        >
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            d="M5 13l4 4L19 7"
                          />
                        </svg>
                        {f}
                      </li>
                    ))}
                  </ul>

                  {isCurrent ? (
                    <div className="rounded-md bg-accent-500/10 px-3 py-2 text-center text-xs font-medium text-accent-400">
                      Current Plan
                    </div>
                  ) : plan.tier === "free" ? null : isDowngrade ? (
                    <div className="rounded-md bg-surface-300 px-3 py-2 text-center text-xs text-neutral-500">
                      Downgrade via Manage Subscription
                    </div>
                  ) : (
                    <button
                      onClick={() => handleUpgrade(plan.tier)}
                      disabled={upgrading !== null}
                      className="w-full rounded-md bg-accent-500 px-3 py-2 text-xs font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50"
                    >
                      {upgrading === plan.tier
                        ? "Redirecting..."
                        : `Upgrade to ${plan.name}`}
                    </button>
                  )}
                </div>
              );
            })}
          </div>
          <p className="mt-2 text-xs text-neutral-600">
            Enterprise pricing is custom.{" "}
            <a href="mailto:sales@freezenith.com" className="text-accent-500 hover:underline">
              Contact sales
            </a>{" "}
            for details.
          </p>
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

      {/* Cancel Confirmation Modal */}
      {showCancel && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
          <div className="mx-4 w-full max-w-sm rounded-lg border border-border bg-surface-100 p-6">
            <h3 className="text-sm font-medium text-white">
              Cancel Subscription
            </h3>
            <p className="mt-2 text-xs text-neutral-400">
              Your subscription will remain active until the end of the current
              billing period. After that, you&apos;ll be downgraded to the Free
              plan. Existing resources won&apos;t be deleted, but you won&apos;t
              be able to create new ones beyond Free limits.
            </p>
            <div className="mt-4 flex justify-end gap-2">
              <button
                onClick={() => setShowCancel(false)}
                className="rounded-md border border-border px-3 py-1.5 text-xs text-neutral-400 hover:text-white transition-colors"
              >
                Keep Plan
              </button>
              <button
                onClick={handleCancel}
                className="rounded-md bg-red-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-red-700 transition-colors"
              >
                Cancel at Period End
              </button>
            </div>
          </div>
        </div>
      )}
    </Shell>
  );
}
