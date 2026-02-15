"use client";

import { Shell } from "@/components/shell";
import { PageHeaderSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { useApi } from "@/hooks/use-api";
import { useProject } from "@/hooks/use-project";
import { projects, type Project } from "@/lib/api";

export default function BillingPage() {
  const projectId = useProject();

  const {
    data: project,
    loading,
    error,
    refetch,
  } = useApi(() => projects.get(projectId), [projectId]);

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

  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">Billing</h1>
          <p className="text-sm text-neutral-500">
            Plan, usage, and cost breakdown
          </p>
        </div>

        {/* Current Plan */}
        <section>
          <div className="rounded-lg border border-border bg-surface-100 p-5">
            <div className="flex items-center justify-between">
              <div>
                <div className="flex items-center gap-2">
                  <p className="text-sm font-medium text-white">
                    Current Plan
                  </p>
                  <span className="inline-flex items-center rounded-full bg-accent-500/10 px-2.5 py-0.5 text-xs font-medium text-accent-400">
                    {project?.plan || "--"}
                  </span>
                </div>
                <p className="mt-1 text-xs text-neutral-500">
                  Billing details will be available once the billing API is
                  connected.
                </p>
              </div>
              <div className="text-right">
                <p className="text-2xl font-semibold text-white">--</p>
                <p className="text-xs text-neutral-500">per month</p>
              </div>
            </div>
          </div>
        </section>

        {/* Resource Usage Breakdown */}
        <section>
          <div className="mb-3">
            <h2 className="text-sm font-medium text-white">Resource Usage</h2>
          </div>
          <div className="rounded-lg border border-border bg-surface-100 p-8 text-center">
            <p className="text-sm text-neutral-500">
              Detailed usage and cost breakdown will be available once the
              billing API is connected.
            </p>
          </div>
        </section>

        {/* Billing Info */}
        <section>
          <div className="mb-3">
            <h2 className="text-sm font-medium text-white">
              Billing Information
            </h2>
          </div>
          <div className="rounded-lg border border-border bg-surface-100 p-5">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-surface-300">
                  <svg
                    className="h-4.5 w-4.5 text-neutral-400"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z"
                    />
                  </svg>
                </div>
                <div>
                  <p className="text-sm text-neutral-300">
                    No payment method configured
                  </p>
                  <p className="text-xs text-neutral-500">
                    Add a payment method to unlock paid features
                  </p>
                </div>
              </div>
              <button className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors">
                Add Payment Method
              </button>
            </div>
          </div>
        </section>
      </div>
    </Shell>
  );
}
